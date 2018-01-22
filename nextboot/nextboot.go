package nextboot

// Config contains a nextboot configuration for an ePoxy client.
type Config struct {
	// Kargs contains kernel command line parameters, typically read from
	// /proc/cmdline. Kernel parameters are split on the first `=`, taking the
	// left hand side as the Kargs key, and the right hand side as the value.
	// If there is no `=` in the parameter, the entire parameter becomes the
	// key with an empty value. All keys and values are strings.
	//
	// For example, if there were a parameter `ide-core.nodma=0.1`, then Kargs
	// will contain a key `ide-core.nodma` with a value of `0.1`. Kargs may be
	// referenced in templates using the `kargs` template function. See more
	// examples below.
	Kargs map[string]string `json:"kargs,omitempty"`

	// V1 specifies an action to be taken by an ePoxy client.
	V1 *V1 `json:"v1,omitempty"`
}

// V1 specifies an action for an ePoxy client to execute. V1 configurations
// could be generated by the ePoxy server, another service, or written by an
// operator. V1 supports two primitive actions:
//
//   1. Load another nextboot configuration from a `Chain` URL.
//   2. Execute `Commands`, using the given environment.
//
// When the Chain URL is present, Commands and related fields are ignored.
//
// When the Chain URL is empty, then kernel parameters, config variables,
// files to download, and environment, are evaluated before executing
// Commands. The values of several fields are evaluated as templates. The
// templates are evaluated in dependency order.
//
//   1. Kargs are loaded statically, typically from /proc/cmdline.
//   2. Vars may reference values in Kargs.
//   3. Files may reference values in Vars or Kargs.
//   4. Env may reference values in Files, Vars, or Kargs.
//   5. Commands may reference values in Files, Vars, or Kargs.
//
// Commands may access environment variables programatically once run, but
// environment values are not accessible to templates.
type V1 struct {

	// Chain is a URL to a new nextboot configuration. The contents of
	// that configuration may contain a new Chain URL, but should typically
	// refer to a config with Commands.
	Chain string `json:"chain,omitempty"`

	// Vars contains key/value pairs. Every string value is evaluated as
	// a template. Every template value may only reference kernel parameters
	// using the "kargs" template function. For example, if there was originally
	// a kernel parameter "net.arg1=test", then a variable template like:
	//
	//     "this is a {{kargs `net.arg1`}}"
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

	// Files is a map of names to source specs. Every source spec must contain a
	// URL using the key name "url". The source spec may contain a sha256
	// checksum of the file using the key name "sha256". After downloading the
	// URL to a local file, the checksum is used to verify the file contents.
	//
	// The local filename is added to the source spec using the key "name" so
	// that templates may access file names through the ".files" namespace.
	//
	// Each URL is evaluated as a template. File URLs may
	// reference kernel parameters using the "kargs" template function, and
	// variable values using the ".vars" namespace.
	//
	// For compatibility with template lookup, Files keys must not contain ".".
	//
	// For example, after download, the local file name of a source spec like:
	//
	// "files": {
	//    "initram" : {
	//       "url" : "https://storage.com/coreos_initram.cpio.gz",
	//       "sha256" : "37c0e81be3a24752fcc2bc51c20e8dae897417dfaabbdce3a8b1efc8a2d310c6"
	//    }
	// }
	//
	// can be referenced in templates as:
	//
	//   {{.files.initram.name}}
	//
	// Files may be empty.
	Files map[string]map[string]string `json:"files,omitempty"`

	// Env is a map of environment variable names to values. These values are
	// added to the environment when running Commands. Values are evaluated as
	// a template, allowing substitution of values using "kargs" template
	// function, as well as the ".vars" and ".files" namespaces.
	//
	// Env may be empty. Unless overridden, a default environment will include:
	//   PATH=/usr/bin:/bin
	//   USER=root
	//   HOME=/
	Env map[string]string `json:"env,omitempty"`

	// Commands is a list of commands to execute, using Env. Every command is
	// evaluated as a template, allowing substitution of values from the
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
	// for adding in-line comments to JSON files.
	//
	// Shell-style quotation is supported for command arguments. So, the
	// following would be split into three elements:
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
