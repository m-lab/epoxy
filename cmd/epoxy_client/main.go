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
	"time"

	"github.com/m-lab/epoxy/nextboot"
	"github.com/m-lab/go/rtx"
)

const timeout = 6 * time.Hour

var (
	flagCmdline = flag.String("cmdline", "/proc/cmdline",
		"Read kernel cmdline parameters from the contents of this file.")
	flagAction = flag.String("action", "epoxy.stage2",
		"Execute the config loaded from the URL in this kernel parameter.")
	flagAddKargs = flag.Bool("add-kargs", false,
		"Combine the local kargs with those returned from the action url. "+
			"Existing kargs are never replaced. Only useful for stage1.")
	flagReport = flag.String("report", "epoxy.report",
		"Report success or errors with the URL in this kernel parameter.")
	flagDryrun = flag.Bool("dryrun", false,
		"Request all configs but do not run commands. May change state in the ePoxy server.")
	flagNoRetry = flag.Bool("no-retry", false, "Do not retry in case of failure")
)

func main() {
	var result string
	var runErr error

	flag.Parse()
	c := &nextboot.Config{}

	b, err := ioutil.ReadFile(*flagCmdline)
	if err != nil {
		log.Fatal(err)
	}
	// Read and parse parameters from *flagCmdline.
	c.ParseCmdline(string(b))

	deadline := time.Now().Add(timeout)

	for {
		// Run the config loaded from the action URL.
		runErr := c.Run(*flagAction, *flagAddKargs, *flagDryrun)
		if runErr != nil {
			// Define a successful result.
			result = "error: " + runErr.Error()
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
			log.Print(err)
		}

		// Stop the retry loop if the -no-retry flag has been provided,
		// the last execution succeeded or enough time has passed.
		if *flagNoRetry || runErr == nil || time.Now().After(deadline) {
			break
		}

		log.Println("Waiting 1 minute before retrying...")
		time.Sleep(1 * time.Minute)
	}

	// If the run step failed, reboot the machine
	if runErr != nil {
		reboot()
	}
}

func reboot() {
	err := ioutil.WriteFile("/proc/sys/kernel/sysrq", []byte{'1'}, 0644)
	rtx.Must(err, "Error while writing sysrq")

	err = ioutil.WriteFile("/proc/sysrq-trigger", []byte{'b'}, 0644)
	rtx.Must(err, "Error while sending sysrq")
}
