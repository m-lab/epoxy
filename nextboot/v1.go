package nextboot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
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
	return nil
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
	resp, err := postWithTimeout(reportURL, values, 10*time.Minute)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// TODO: what statuses should we support?
	// Note: we expect http.StatusNoContent, but accept any 200 code.
	// Note: the go client automatically handles standard redirects.
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return fmt.Errorf("Bad status code: got %d, expected %d",
		resp.StatusCode, http.StatusNoContent)
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
