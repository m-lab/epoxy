package nextboot

import (
	"io"
)

// ConfigImpl implements a nextboot configuration for an ePoxy client. The two
// supported actions are:
//
//   1. Load another nextboot configuration from a URL, i.e. Chain.
//   2. Execute a command, i.e. Exec.
//
// These options are mutually exclusive. If both are specified, the Exec spec
// is ignored.
//
// These configurations could be generated by the ePoxy server, another service,
// or written by operator.
type ConfigImpl struct {
	// Version indicates the version of the nextboot config. Used by clients to
	// confirm compatibility.
	Version string `json:"version"`

	// Chain is a URL to source a new nextboot configuration. The contents of
	// that configuration may contain a new Chain URL or an Exec specification.
	Chain string `json:"chain,omitempty"`

	// Exec is a specification for running one or more commands.
	Exec Exec `json:"exec,omitempty"`

	// Kargs contains kernel command line parameters, typically read from
	// /proc/cmdline.
	Kargs map[string]interface{} `json:"kargs,omitempty"`
}

// Exec defines an environment, files to download, and commands to execute.
//
// Many Exec element values are evaluated as templates. The templates are
// evaluated in dependency order.
//
//   1. Vars may reference values in Kargs.
//   2. Files may reference values in Vars or Kargs.
//   3. Env may reference values in Files, Vars, or Kargs.
//   4. Commands may reference values in Files, Vars, or Kargs.
//
// Commands may access environment variables programatically once run, but
// environment values are not accessible to templates.
type ExecImpl struct {
	// Vars contains key/value pairs. Every string value is evaluated as
	// a template. Every template value may only reference kernel parameters
	// from the ".kargs" namespace. For example, if there was originally a
	// kernel parameter "net.arg1=test", then a variable template like:
	//
	//     "this is a {{.kargs.net.arg1}}"
	//
	// would evaluate as and be replaced with:
	//
	//     "this is a test"
	//
	// For compatibility with template lookup, Vars keys must not contain ".".
	//
	// Vars supports three value types:
	//
	//   1. string - evaluated as a template.
	//   2. []string -- as a convenience, an array of strings is first converted
	//         to a single string by joining each element with a space. After
	//         converstion the result is evaluated as a template.
	//   3. map[string]interface{} -- as a convenience, allows creation of
	//         recursively nested names.
	//
	// Other types are ignored.
	//
	// Vars may be empty.
	Vars map[string]interface{} `json:"vars,omitempty"`

	// Files is a map of names to source URLs. Every source URL is downloaded
	// locally and the resulting filename is available in templates using the
	// ".files" namespace. Each URL is evaluated as a template. File URLs may
	// reference kernel parameters using the ".kargs" namespace, and variable
	// values using the ".vars" namespace.
	//
	// For compatibility with template lookup, Files keys must not contain ".".
	//
	// Files may be empty.
	Files map[string]string `json:"files,omitempty"`

	// Env is a map of environment variable names to values. These values are
	// added to the environment when running Commands. Values are evaluated as
	// a template, allowing substitution of values from the ".kargs", ".vars",
	// and ".files" namespaces.
	//
	// Env may be empty. Unless overridden, a default environment will include:
	//   PATH=/usr/bin:/bin
	//   SHELL=/bin/sh
	//   USER=root
	//   HOME=/
	Env map[string]string `json:"env,omitempty"`

	// Commands is a list of commands to execute, using Env. Every command is
	// first evaluated as a template, allowing substitution of values from the
	// ".kargs", ".vars", and ".files" namespaces.
	//
	// Two value types are supported for elements of Commands:
	//
	//   1. string - a full command line, supporting shell-style quotation.
	//   2. []string - an argv form of command, where the first element is the
	//   command to execute and following elements are separate parameters.
	//   Quotes are left as-is.
	//
	// Other types are ignored.
	//
	// Formatting:
	//
	// Commands are ignored when the first character is '#'. This may be helpful
	// for adding in-line comments to JSON files. Shell-style quotation is
	// supported for command arguments. So, the following would be split into
	// three elements:
	//
	//   /bin/argv0 --command="argv1 with spaces" argv2
	//
	// Execution:
	//
	// Commands are executed by the 'root' user with a working directory in
	// '/'. Commands may log to stdout and stderr as usual. Commands should
	// terminate successfully (i.e. zero exit code). If a command does not
	// appear to be making progress, it will be forcibly terminated and
	// reported as an error.
	Commands []interface{} `json:"commands,omitempty"`
}

type Config interface {
	// ParseKernelCmdline reads and parses the kernel command line parameters.
	ParseKernelCmdline(cmdline io.Reader) error

	// Run evaluates the nextboot configuration.
	// if Chain, download, parse, and recursively Run new config.
	// if Exec, evaluate vars, download files, and execute command.
	Run() error

	// Report uses the epoxy.report URL to report a success or failure.
	Report(err error) error
}

type Exec interface {
	// EvaluateVars processes the template values of Vars using kargs.
	EvaluateVars(kargs map[string]interface{}) error

	// DownloadFiles processes the template values of Files and then downloads
	// the resulting URL.
	DownloadFiles(kargs map[string]interface{}) error

	// Run evaluates the Env, and Commands before executing the resulting
	// commands.
	Run(kargs map[string]interface{}) error
}
