package nextboot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var (
	// ErrActionURLNotFound is returned when the Kargs key is missing.
	ErrActionURLNotFound = errors.New("Action URL key not found")
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
// Note that this may change state in the ePoxy server.
func (c *Config) Run(action string, dryrun bool) error {
	log.Printf("Loading config from: %s", c.Kargs[action])
	actionURL, ok := c.Kargs[action]
	if !ok {
		return ErrActionURLNotFound
	}
	// Load config from ePoxy server.
	err := c.loadConfig(actionURL, "POST")
	if err != nil {
		return err
	}
	return c.runChainOrCommands()
}

func (c *Config) runChainOrCommands() error {
	if c.V1.Chain != "" {
		// If the Chain URL is present, run it.
		log.Println("Running chain", c.V1.Chain)
		err := c.loadConfig(c.V1.Chain, "GET")
		if err != nil {
			return err
		}
		// Since we've loaded a new config, restart.
		return c.runChainOrCommands()
	}
	// There is no Chain URL, so attempt to run Commands.
	log.Println("Running commands")
	return c.runCommands()
}

func (c *Config) runCommands() error {
	// TODO: implement var, files, env, and commands template evaluation
	// and command execution.
	log.Println(c.String())
	return nil
}

func (c *Config) loadConfig(source, method string) error {
	var err error
	var body io.ReadCloser
	switch {
	case strings.HasPrefix(source, "file://"):
		// Strip off the file:// prefix. Useful for testing and possibly stage1 legacy boot CDs.
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
