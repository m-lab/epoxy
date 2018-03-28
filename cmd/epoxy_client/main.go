// epoxy_client is a command line utility for requesting nextboot configurations
// from the ePoxy server and executing them.
//
// epoxy_client should be embedded in initram images served by ePoxy. Once the
// network is initialized, epoxy_client can complete actions for the current
// boot stage. i.e. download config from epoxy, download kernel for stage3,
// kexec kernel.
package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/url"

	"github.com/m-lab/epoxy/nextboot"
)

var (
	flagCmdline = flag.String("cmdline", "/proc/cmdline",
		"Read kernel cmdline parameters from the contents of this file.")
	flagAction = flag.String("action", "epoxy.stage2",
		"Execute the config loaded from the URL in this kernel parameter.")
	flagReport = flag.String("report", "epoxy.report",
		"Report success or errors with the URL in this kernel parameter.")
	flagDryrun = flag.Bool("dryrun", false,
		"Request all configs but do not run commands. May change state in the ePoxy server.")
)

func main() {
	var result string

	flag.Parse()
	// TODO: Optionally retry in a loop until success or 6 hours of
	// failure have occurred. Automatically reboot after 6 hours of failure.
	c := &nextboot.Config{}

	b, err := ioutil.ReadFile(*flagCmdline)
	if err != nil {
		log.Fatal(err)
	}
	// Read and parse parameters from *flagCmdline.
	c.ParseCmdline(string(b))

	// Run the config loaded from the action URL.
	err = c.Run(*flagAction, *flagDryrun)
	if err != nil {
		// Define a successful result.
		result = "error: " + err.Error()
	} else {
		result = "success"
	}
	log.Println("Result:", result)

	// Report a message to the ePoxy server after running.
	values := url.Values{}
	// TODO: report additional host information.
	// TODO: log the evaluate state of c.V1 -- helpful especially for errors.
	values.Set("message", result)

	err = c.Report(*flagReport, values, *flagDryrun)
	if err != nil {
		log.Fatal(err)
	}

	// Note: we may reboot without depending on the reboot command using:
	//   echo 1 > /proc/sys/kernel/sysrq
	//   echo b > /proc/sysrq-trigger
}
