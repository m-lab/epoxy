package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/url"

	"github.com/stephen-soltesz/epoxy-1/nextboot"
)

var (
	cmdline = flag.String("cmdline", "/proc/cmdline",
		"Read kernel cmdline parameters from the contents of this file.")
	action = flag.String("action", "epoxy.stage2",
		"Execute the config loaded from the URL in this kernel parameter.")
	report = flag.String("report", "epoxy.report",
		"Report success or errors with the URL in this kernel parameter.")
	dryrun = flag.Bool("dry-run", false,
		"Request all configs but do not run commands. May change state in the ePoxy server.")
)

func main() {
	var result string

	flag.Parse()
	// TODO: optionally retry in a loop until success, or automatically reboot
	// after 8hrs of failure.
	c := &nextboot.Config{}

	b, err := ioutil.ReadFile(*cmdline)
	if err != nil {
		log.Fatal(err)
	}
	// Trim leading and trailing whitespace, then split on space.
	// Read and parse parameters from *cmdline.
	c.ParseCmdline(string(b))

	// Run the config loaded from the action URL.
	err = c.Run(*action, *dryrun)
	if err != nil {
		// Define a successful result.
		result = err.Error()
	} else {
		result = "success"
	}

	// Report a message to the ePoxy server after running.
	values := url.Values{}
	// TODO: report additional host information.
	values.Set("message", result)

	err = c.Report(*report, values)
	if err != nil {
		log.Fatal(err)
	}

	// Note: we may reboot without depending on the reboot command using:
	//   echo 1 > /proc/sys/kernel/sysrq
	//   echo b > /proc/sysrq-trigger
}