package nextboot

import (
	"log"
	"net/url"
	"strings"
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
	log.Printf("Reporting values to: %s", c.Kargs[report])
	return nil
}
