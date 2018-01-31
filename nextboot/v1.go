package nextboot

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"github.com/cavaliercoder/grab"
	"github.com/google/shlex"
)

var (
	// ErrActionURLNotFound is returned when the Kargs key is missing.
	ErrActionURLNotFound = errors.New("Action URL key not found")

	// ErrFileURLNotFound is returned with a file spec does not include a "url" key.
	ErrFileURLNotFound = errors.New("URL key not found in file spec")
)

// useVars and useFiles are flags for evaluating templates.
const (
	useVars uint32 = 1 << iota
	useFiles
)

// largeTimeout sets an upper limit on time taken to run commands or large file downloads.
const largeTimeout = 2 * time.Hour

// ParseCmdline parses the contents of `cmdline` as kernel parameters to
// initialize `Kargs`. The current value of Kargs is unconditionally overwritten.
func (c *Config) ParseCmdline(cmdline string) error {
	log.Printf("Parsing kernel parameters from: %s", cmdline)

	// Trim leading and trailing whitespace, then split on spaces.
	params := strings.Split(strings.Trim(cmdline, " \n"), " ")
	kargs := make(map[string]string, len(params))
	for _, param := range params {
		// For each parameter, split on first `=` if present.
		keyval := strings.SplitN(param, "=", 2)
		// Note: an unsupported kernel parameter may not work correctly.
		// e.g. if a parameter were just "http://foo.com?a=b".
		if len(keyval) == 2 {
			kargs[keyval[0]] = keyval[1]
		} else {
			// A single value parameter.
			kargs[keyval[0]] = ""
		}
	}

	// Set Kargs to the values we just parsed.
	c.Kargs = kargs
	return nil
}

// Run requests the URL stored in `Kargs[action]` and loads the returned config.
// After loading the returned config, Run executes the `V1` actions until an
// error occurs or all commands are successfully executed.
//
// If dryrun is true, then configuration commands are printed but not executed.
// Note: even in dryrun mode action URLS **may change state** in the ePoxy server.
func (c *Config) Run(action string, dryrun bool) error {
	log.Printf("Loading config from: %s", c.Kargs[action])
	actionURL, ok := c.Kargs[action]
	if !ok {
		return ErrActionURLNotFound
	}
	// Load config from ePoxy server.
	err := c.loadAction(actionURL, "POST")
	if err != nil {
		return err
	}
	err = c.maybeLoadChain()
	if err != nil {
		return err
	}
	// There is no Chain URL, so attempt to run Commands.
	log.Println("Running commands")
	return c.runCommands(dryrun)
}

func (c *Config) maybeLoadChain() error {
	for c.V1.Chain != "" {
		// If the Chain URL is present, run it.
		log.Println("Running chain", c.V1.Chain)
		err := c.loadAction(c.V1.Chain, "GET")
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) runCommands(dryrun bool) error {
	err := c.evaluateVars()
	if err != nil {
		return err
	}
	err = c.evaluateAndDownloadFiles(dryrun)
	defer c.cleanupFiles()
	if err != nil {
		return err
	}
	err = c.evaluateEnv()
	if err != nil {
		return err
	}
	err = c.evaluateCommands()
	if err != nil {
		return err
	}

	// Update, backup, and restore the current process environment. updateCurrentEnv
	// is necessary to use user-specified PATH for command lookup and avoid more
	// complex fork/exec steps.
	changed, added := updateCurrentEnv(c.V1.Env, map[string]string{})
	defer updateCurrentEnv(changed, added)

	for _, fields := range c.V1.Commands {
		// Convert the native Commands []interface{} type to []string.
		args := interfaceToStringArray(fields)
		if len(args) == 0 {
			// shlex.Split on comment strings result in zero length args arrays.
			continue
		}
		// Print command in a copy/paste-able form.
		log.Printf("Command: \"%s\"", strings.Join(args, `" "`))
		if dryrun {
			continue
		}
		// TODO: make timeout a parameter.
		// Note: after ctx timeout, command receives SIGKILL.
		ctx, cancel := context.WithTimeout(context.Background(), largeTimeout)
		defer cancel()

		// cmd inherits the current process environment.
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)

		// Use the current stdout and stderr for subcommands.
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			// Report error with the command args and error.
			return fmt.Errorf("%q : %v", args, err)
		}
	}
	return nil
}

// updateCurrentEnv sets variables from setenv in the current process
// environment, deletes variables in delenv, and returns two maps indicating
// whether variables where "changed" or "added" to the env. To restore the
// environment, call updateCurrentEnv again with those return values.
func updateCurrentEnv(setenv, delenv map[string]string) (map[string]string, map[string]string) {
	changed := map[string]string{}
	added := map[string]string{}
	// Unset all values in delenv that were previously "not found".
	for key := range delenv {
		os.Unsetenv(key)
	}
	for key, val := range setenv {
		// Lookup the current value of key from environment.
		orig, found := os.LookupEnv(key)
		// Record whether this is a value we're changing or adding.
		if found {
			changed[key] = orig
		} else {
			added[key] = ""
		}
		// Set the new value.
		os.Setenv(key, val)
	}
	return changed, added
}

func (c *Config) evaluateVars() error {
	// In a single pass, convert each element to a string.
	for key, value := range c.V1.Vars {
		switch val := value.(type) {
		case []interface{}:
			// Reconstruct a single string from an array of strings.
			c.V1.Vars[key] = strings.Join(interfaceToStringArray(val), " ")
		case string:
			// No-op, this is good as-is.
		default:
			// TODO: either ignore these values or update notes in nextboot.go definition.
			return fmt.Errorf("Unsupported type: %T %#v", val, val)
		}
		// Due to the above, all types be strings.
		sval, err := c.evaluateAsTemplate(c.V1.Vars[key].(string), 0)
		if err != nil {
			return err
		}
		// Replace with the new value.
		c.V1.Vars[key] = sval
	}
	return nil
}

// TODO: separate these operations to allow user-provided "names".
func (c *Config) evaluateAndDownloadFiles(dryrun bool) error {
	for name, urlspec := range c.V1.Files {
		url, ok := urlspec["url"]
		if !ok {
			return ErrFileURLNotFound
		}
		url, err := c.evaluateAsTemplate(url, useVars)
		if err != nil {
			return err
		}
		// Update spec with evaluated URL.
		urlspec["url"] = url

		// Create a tempfile for saving file locally.
		tmpfile, err := ioutil.TempFile("", name+"-")
		if err != nil {
			return err
		}
		tmpfile.Close()

		// TODO: make timeout a parameter.
		if !dryrun {
			err = fileDownload(tmpfile.Name(), url, urlspec, largeTimeout)
			if err != nil {
				os.Remove(tmpfile.Name())
				return err
			}
		}

		// Update the Files map with local file name.
		c.V1.Files[name]["name"] = tmpfile.Name()
	}
	return nil
}

func (c *Config) cleanupFiles() {
	for _, urlspec := range c.V1.Files {
		// Remove local files once we no longer need them.
		if fname, ok := urlspec["name"]; ok {
			log.Printf("Removing tmpfile: %s", fname)
			os.Remove(fname)
		}
	}
	return
}

func (c *Config) evaluateEnv() error {
	for key, val := range c.V1.Env {
		s, err := c.evaluateAsTemplate(val, useVars|useFiles)
		if err != nil {
			return err
		}
		c.V1.Env[key] = s
	}
	return nil
}

// evaluateCommands normalizes the underlying Commands types, converting
// every element to []interface{}.
func (c *Config) evaluateCommands() error {
	// Run in two passes.
	// 1. split all strings into []interface{}, the default type used
	// by JSON Unmarshal for array types.
	for i, value := range c.V1.Commands {
		switch cmdTmpl := value.(type) {
		case string:
			// To support kargs we must evaluate the template before splitting.
			cmd, err := c.evaluateAsTemplate(cmdTmpl, useVars|useFiles)
			if err != nil {
				return err
			}
			// Note: shlex.Split returns an empty list for comments.
			args, err := shlex.Split(cmd)
			if err != nil {
				// Split may fail due to incomplete quotes.
				return err
			}
			// Convert []string to []interface{}.
			c.V1.Commands[i] = stringToInterfaceArray(args)
		}
	}
	// 2. Now every element of Commands is an []interface{}. Evaluate every
	// element of every []interface{} as a template.
	for _, value := range c.V1.Commands {
		switch args := value.(type) {
		case []interface{}:
			for i, argTmpl := range args {
				arg, err := c.evaluateAsTemplate(fmt.Sprint(argTmpl), useVars|useFiles)
				if err != nil {
					return err
				}
				args[i] = arg
			}
		}
	}
	return nil
}

func interfaceToStringArray(array interface{}) []string {
	a, ok := array.([]interface{})
	if !ok {
		// Ignore invalid types.
		return []string{}
	}
	s := make([]string, len(a))
	for i, v := range a {
		s[i] = fmt.Sprint(v)
	}
	return s
}

func stringToInterfaceArray(a []string) []interface{} {
	s := make([]interface{}, len(a))
	for i, v := range a {
		s[i] = fmt.Sprint(v)
	}
	return s
}

// evaluateAsTemplate accepts a value string that is evaluated as a Go template.
// The template may reference elements from Config.
//
// Kargs values are always accessible using the kargs template function:
//
//     {{kargs `keyname`}}
//
// Optionally, V1.Vars and V1.Files are accessible with the appropriate flags,
// using dot notation. For example:
//
//     {{.vars.keyname}}
//     {{.files.keyname.name}}
func (c *Config) evaluateAsTemplate(value string, flags uint32) (string, error) {
	var t *template.Template
	// Construct the "func map" for the custom `kargs` template function.
	kmap := template.FuncMap{
		"kargs": func(s string) string {
			return c.Kargs[s]
		},
	}
	t, err := template.New("tmpl").Funcs(kmap).Parse(value)
	if err != nil {
		return "", err
	}
	// Allocate an empty namespace map, and conditionally set vars and files.
	ns := map[string]interface{}{}
	if flags&useVars != 0 {
		ns["vars"] = c.V1.Vars
	}
	if flags&useFiles != 0 {
		ns["files"] = c.V1.Files
	}
	// Execute the template, saving the result in the bytes buffer.
	var b bytes.Buffer
	err = t.Execute(&b, ns)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

func (c *Config) loadAction(source, method string) error {
	var err error
	var body io.ReadCloser
	var file *os.File
	switch {
	case strings.HasPrefix(source, "file://"):
		// Useful for testing and possibly stage1 legacy boot CDs.
		// Strip off the file:// prefix before opening named file.
		body, err = os.Open(source[7:])
	case method == "POST":
		// TODO: send additional host metadata in values.
		// TODO: make timeout configurable.
		// Note: this will typically be a state-changing request to the ePoxy server.
		body, err = postDownload(source, url.Values{}, 10*time.Minute)
	case method == "GET":
		// TODO: make timeout configurable.
		// Note: this will typically be a simple file download from GCS.
		file, err = getDownload(source, 10*time.Minute)
		body = file
		if file != nil {
			defer os.Remove(file.Name())
		}
	}
	if err != nil {
		return err
	}
	defer body.Close()

	n := &Config{}
	err = json.NewDecoder(body).Decode(&n)
	if err != nil {
		return err
	}
	// Note: we never overwrite Kargs from an external source.
	// Note: only overwrite the V1 action with what we just loaded above.
	c.V1 = n.V1
	return nil
}

func getDownload(source string, timeout time.Duration) (*os.File, error) {
	// Create a tempfile for saving file locally.
	tmpfile, err := ioutil.TempFile("", "getdownload-")
	if err != nil {
		return nil, err
	}
	err = fileDownload(tmpfile.Name(), source, nil, timeout)
	if err != nil {
		os.Remove(tmpfile.Name())
		return nil, err
	}
	return tmpfile, nil
}

func postDownload(source string, values url.Values, timeout time.Duration) (io.ReadCloser, error) {
	resp, err := postWithTimeout(source, values, timeout)
	if err != nil {
		return nil, err
	}
	// TODO: what statuses should we support?
	// Note: the go client automatically handles standard redirects.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Bad status code: got %d, expected 200 code", resp.StatusCode)
	}
	return resp.Body, nil
}

func watchDownload(resp *grab.Response, update time.Duration) {
	// Update message every few seconds.
	if update < time.Second {
		update = time.Second
	}
	tick := time.NewTicker(update)
	defer tick.Stop()

	lastCount := resp.BytesComplete()
	for {
		select {
		case <-tick.C:
			current := resp.BytesComplete()
			log.Printf("  transferred %v / %v bytes (%.2f%%) at %.2f Mbps",
				current,
				resp.Size,
				100*resp.Progress(),
				float64(current-lastCount)/1.0e6/float64(update/time.Second))
			lastCount = current

		case <-resp.Done:
			// Download has stopped. Check resp.Err() for possible errors.
			return
		}
	}
}

func fileDownload(dest, source string, urlspec map[string]string, timeout time.Duration) error {
	client := grab.NewClient()
	req, err := grab.NewRequest(dest, source)
	if err != nil {
		return err
	}

	if timeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		req = req.WithContext(ctx)
	}

	if checksum, ok := urlspec["sha256"]; ok {
		rawSum, err := hex.DecodeString(checksum)
		if err != nil {
			return err
		}
		req.SetChecksum(sha256.New(), rawSum, true)
	}

	// Start download.
	log.Printf("Download from: %v", req.URL())
	resp := client.Do(req)

	// Report download progress every 5 seconds.
	watchDownload(resp, 5*time.Second)

	// Check for errors.
	if err := resp.Err(); err != nil {
		log.Printf("Download failed: %v", err)
		return err
	}

	log.Printf("Download saved to: %v", resp.Filename)
	return nil
}

// Report reports values to the URL stored in `Kargs[report]`.
func (c *Config) Report(report string, values url.Values, dryrun bool) error {
	log.Printf("Reporting values using %s=%s", report, c.Kargs[report])
	reportURL, ok := c.Kargs[report]
	if !ok {
		return ErrActionURLNotFound
	}
	// Add the current config as a debug parameter on every Report.
	values.Set("debug.config", c.String())

	if dryrun {
		log.Print(values)
	} else {
		// TODO: make timeout configurable.
		body, err := postDownload(reportURL, values, 10*time.Minute)
		if err != nil {
			return err
		}
		// Unconditionally close body, since don't expect any content.
		body.Close()
	}
	return nil
}

func postWithTimeout(url string, values url.Values, timeout time.Duration) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel() // cancel the context if Do() returns before timeout.
	req = req.WithContext(ctx)

	client := &http.Client{}
	return client.Do(req)
}

// String converts the Config instance into a string representation.
func (c *Config) String() string {
	b, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		// Marshal errors occur if there is an unmarshalable type, which Config does not have.
		return err.Error()
	}
	return string(b)
}
