package nextboot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"github.com/google/shlex"
)

var (
	// ErrActionURLNotFound is returned when the Kargs key is missing.
	ErrActionURLNotFound = errors.New("Action URL key not found")
)

// useVars and useFiles are flags for evaluating templates.
const (
	useVars int = 1 << iota
	useFiles
)

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
	return c.runCommands()
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

func (c *Config) runCommands() error {
	err := c.evaluateVars()
	if err != nil {
		return err
	}
	err = c.evaluateFiles()
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

	// backup and restore the current process environment. updateCurrentEnv is
	// necessary to use user-specified PATH for command lookup and prevent more
	// complex fork/exec hoops.
	backupEnv := updateCurrentEnv(c.V1.Env)
	defer updateCurrentEnv(backupEnv)

	for _, fields := range c.V1.Commands {
		args := interfaceToStringArray(fields)
		if len(args) == 0 {
			// Comments are zero length.
			continue
		}
		// TODO: make timeout a parameter.
		// Note: after ctx timeout, command receives SIGKILL.
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
		defer cancel()

		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Env = append(os.Environ(), mapToEnv(c.V1.Env)...)
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

func updateCurrentEnv(env map[string]string) map[string]string {
	backup := map[string]string{}
	for key, val := range env {
		backup[key] = os.Getenv(key)
		os.Setenv(key, val)
	}
	return backup
}

func (c *Config) evaluateVars() error {
	// In a single pass, convert each element to a string.
	for key, value := range c.V1.Vars {
		switch val := value.(type) {
		case []interface{}:
			// Replace with a string.
			c.V1.Vars[key] = strings.Join(interfaceToStringArray(val), " ")
		case string:
			// No-op, this is good as-is.
		default:
			return fmt.Errorf("Unsupported type: %T %#v", val, val)
		}
		sval, err := c.evaluateAsTemplate(c.V1.Vars[key].(string), 0)
		if err != nil {
			return err
		}
		// Replace with the new value.
		c.V1.Vars[key] = sval
	}
	return nil
}

func (c *Config) evaluateFiles() error {
	// TODO: implement large file download.
	return nil
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

// evaluateCommands normalies the underlying Commands types, converting
// every element to []string.
func (c *Config) evaluateCommands() error {
	// Run in two passes.
	// 1. split all strings into []interface{}, the default type used
	// by JSON Unmarshal for array types.
	for i, value := range c.V1.Commands {
		switch cmd := value.(type) {
		case string:
			// To support kargs we must evaluate the template before splitting.
			args, err := c.evaluateAsTemplate(cmd, useVars|useFiles)
			if err != nil {
				return err
			}
			// Note: shlex.Split returns an empty list for comments.
			fields, err := shlex.Split(args)
			if err != nil {
				// Split may fail due to incomplete quotes.
				return err
			}
			// Convert []string to []interface{}.
			c.V1.Commands[i] = stringToInterfaceArray(fields)
		}
	}
	// 2. Now every element of Commands is an []interface{}. Evaluate every
	// element of every []interface{} as a template.
	for _, value := range c.V1.Commands {
		switch fields := value.(type) {
		case []interface{}:
			for i, e := range fields {
				arg, err := c.evaluateAsTemplate(fmt.Sprint(e), useVars|useFiles)
				if err != nil {
					return err
				}
				fields[i] = arg
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

func (c *Config) evaluateAsTemplate(value string, flags int) (string, error) {
	var t *template.Template
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
	if flags&useVars > 0 {
		ns["vars"] = c.V1.Vars
	}
	if flags&useFiles > 0 {
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

func mapToEnv(m map[string]string) []string {
	env := []string{}
	for key, val := range m {
		env = append(env, key+"="+val)
	}
	return env
}

func (c *Config) loadAction(source, method string) error {
	var err error
	var body io.ReadCloser
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
		body, err = getDownload(source, 10*time.Minute)
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

func getDownload(source string, timeout time.Duration) (io.ReadCloser, error) {
	// TODO: implement a reliable, large file download.
	// TODO: use timeout.
	client := &http.Client{}
	resp, err := client.Get(source)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Bad status code: got %d, expected 200 code", resp.StatusCode)
	}
	return resp.Body, nil
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

// Report reports values to the URL stored in `Kargs[report]`.
func (c *Config) Report(report string, values url.Values) error {
	log.Printf("Reporting values using %s=%s", report, c.Kargs[report])
	reportURL, ok := c.Kargs[report]
	if !ok {
		return ErrActionURLNotFound
	}

	// Add the current config as a debug parameter on every Report.
	values.Set("debug.config", c.String())
	// TODO: make timeout configurable.
	body, err := postDownload(reportURL, values, 10*time.Minute)
	if err != nil {
		return err
	}
	// Unconditionally close body, since don't expect any content.
	body.Close()
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
