package nextboot

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/kr/pretty"
	"github.com/renstrom/dedent"
)

func init() {
	// Disable log output.
	log.SetFlags(0)
	log.SetOutput(ioutil.Discard)
}

func TestConfig_ParseCmdline(t *testing.T) {
	type input struct {
		cmdline string
	}
	type expected struct {
		Kargs map[string]string
	}
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		// All tests are expected to succeed.
		{
			name:     "single word",
			input:    "key",
			expected: map[string]string{"key": ""},
		},
		{
			name:     "single value",
			input:    "key=val",
			expected: map[string]string{"key": "val"},
		},
		{
			name:     "single value with extra whitespace",
			input:    "key=val  \n",
			expected: map[string]string{"key": "val"},
		},
		{
			name:     "multi-value",
			input:    "key=val key2=val2",
			expected: map[string]string{"key": "val", "key2": "val2"},
		},
		{
			name:     "multi-value with multiple equal signs",
			input:    "key=val key2=val2=val3",
			expected: map[string]string{"key": "val", "key2": "val2=val3"},
		},
		{
			name:  "multi-value with URL with parameters",
			input: "key=val url=http://thing.com?a=b&c=d",
			expected: map[string]string{
				"key": "val",
				"url": "http://thing.com?a=b&c=d",
			},
		},
		{
			name:     "multi-value with special values in key",
			input:    "key=val ide-core.nodma=0.1",
			expected: map[string]string{"key": "val", "ide-core.nodma": "0.1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{}
			c.ParseCmdline(tt.input)

			if diff := pretty.Diff(c.Kargs, tt.expected); len(diff) != 0 {
				t.Errorf("Config.ParseCmdline() got = %v, want %v\nDiff: %#v",
					c.Kargs, tt.expected, diff)
			}
		})
	}
}

func TestConfig_String(t *testing.T) {
	type fields struct {
		Kargs map[string]string
		V1    *V1
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "empty",
			fields: fields{
				Kargs: map[string]string{},
				V1:    &V1{},
			},
			// Be careful with white space. Indent with tabs, and spaces in the json.
			want: dedent.Dedent(`
				{
				    "v1": {}
				}`),
		},
		{
			name: "full",
			fields: fields{
				Kargs: map[string]string{"key": "val"},
				V1: &V1{
					Chain: "http://foo.com/post",
					Vars: map[string]interface{}{
						"key": "var",
					},
					Files: map[string]map[string]string{
						"vmlinuz": map[string]string{
							"url": "http://foo.com/download",
						},
					},
					Env: map[string]string{
						"a": "b",
					},
					Commands: []interface{}{
						"/bin/true",
					},
				},
			},
			want: dedent.Dedent(`
				{
				    "kargs": {
				        "key": "val"
				    },
				    "v1": {
				        "chain": "http://foo.com/post",
				        "vars": {
				            "key": "var"
				        },
				        "files": {
				            "vmlinuz": {
				                "url": "http://foo.com/download"
				            }
				        },
				        "env": {
				            "a": "b"
				        },
				        "commands": [
				            "/bin/true"
				        ]
				    }
				}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				Kargs: tt.fields.Kargs,
				V1:    tt.fields.V1,
			}
			// want[1:] strips a leading \n which Dedent does not strip.
			if got := c.String(); got != tt.want[1:] {
				t.Errorf("Config.String():")
				t.Errorf("got :\n%#v", got)
				t.Errorf("want:\n%#v", tt.want[1:])
			}
		})
	}
}

func TestConfig_Report(t *testing.T) {
	expectedValues := url.Values{
		"message": {"success"},
	}
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Parse the form data sent from client.
			err := r.ParseForm()
			if err != nil {
				t.Fatal(err)
			}
			// Verify that the expected keys are present.
			for k := range expectedValues {
				if k != "debug.config" && r.PostForm.Get(k) != expectedValues.Get(k) {
					t.Fatalf("Report Handler: got %v; want %v",
						r.PostForm.Get("message"), expectedValues.Get("message"))
				}
			}
			// Verify that the "debug.config" value is present.
			if r.PostForm.Get("debug.config") == "" {
				t.Fatalf("Report Handler: missing 'debug.config' form value")
			}
			w.WriteHeader(http.StatusNoContent)
		}))
	defer ts.Close()
	type args struct {
		report string
		values url.Values
	}
	tests := []struct {
		name    string
		kargs   map[string]string
		args    args
		wantErr bool
	}{
		{
			name: "working",
			kargs: map[string]string{
				// This key must match the args report name.
				"epoxy.report": ts.URL,
			},
			args: args{
				report: "epoxy.report",
				values: expectedValues,
			},
			wantErr: false,
		},
		{
			name: "broken-url",
			kargs: map[string]string{
				// Deliberately construct an invalid URL.
				"epoxy.report": ":this-is-not-a-url",
			},
			args: args{
				report: "epoxy.report",
				values: url.Values{},
			},
			wantErr: true,
		},
		{
			name: "bad-action-key",
			kargs: map[string]string{
				"epoxy.report": ts.URL,
			},
			args: args{
				// Deliberately use the wrong key value in kargs.
				report: "epoxy.wrongkey",
				values: url.Values{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				Kargs: tt.kargs,
			}
			if err := c.Report(tt.args.report, tt.args.values); (err != nil) != tt.wantErr {
				t.Errorf("Config.Report() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_Run(t *testing.T) {
	tests := []struct {
		name       string
		action     string
		statusPost int
		statusGet  int
		wantErr    bool
	}{
		{
			name:       "successful-post-chain-and-get-commands",
			action:     "epoxy.stage2",
			statusPost: http.StatusOK,
			statusGet:  http.StatusOK,
			wantErr:    false,
		},
		{
			name:    "bad-action-key",
			action:  "epoxy.wrongkey",
			wantErr: true,
		},
		{
			name:       "bad-post-http-respose-status",
			action:     "epoxy.stage2",
			statusPost: http.StatusNotFound,
			wantErr:    true,
		},
		{
			name:       "bad-get-http-reponse-status",
			action:     "epoxy.stage2",
			statusPost: http.StatusOK,
			statusGet:  http.StatusNotFound,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup two local test servers to simulate an epoxy client Run. The epoxy
			// client first POSTs a request (typically to the ePoxy server) to receive a
			// Chain reference to a second server (typically on GCS). The epoxy client
			// then GETs that URL, which will typically have commands to execute.
			tsGet := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.statusGet)
					// Declare a minimal config with one command.
					c := &Config{V1: &V1{Commands: []interface{}{"/bin/true okay"}}}
					fmt.Fprint(w, c.String())
				}))
			tsPost := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.statusPost)
					// Declare a minimal config with a Chain reference to tsGet.
					c := &Config{V1: &V1{Chain: tsGet.URL}}
					fmt.Fprint(w, c.String())
				}))
			c := &Config{
				// Start off initializing the stage2 action url to the tsPost test server.
				Kargs: map[string]string{"epoxy.stage2": tsPost.URL},
			}
			if err := c.Run(tt.action, false); (err != nil) != tt.wantErr {
				t.Errorf("Config.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
			tsPost.Close()
			tsGet.Close()
		})
	}
}

func TestConfig_evaluateVars(t *testing.T) {
	tests := []struct {
		name     string
		kargs    map[string]string
		v1       *V1
		expValue string
		wantErr  bool
	}{
		{
			name:  "successfully-evaluate-vars",
			kargs: map[string]string{"kargkey": "world"},
			v1: &V1{
				Vars: map[string]interface{}{
					"varkey": "hello, {{kargs `kargkey`}}",
				},
			},
			expValue: "hello, world",
			wantErr:  false,
		},
		{
			name:  "successfully-evaluate-vars-from-array",
			kargs: map[string]string{"kargkey": "world"},
			v1: &V1{
				Vars: map[string]interface{}{
					"varkey": []interface{}{"hello,", "{{kargs `kargkey`}}"},
				},
			},
			expValue: "hello, world",
			wantErr:  false,
		},
		{
			name: "bad-vars-type",
			v1: &V1{
				Vars: map[string]interface{}{
					"varkey": 10,
				},
			},
			wantErr: true,
		},
		{
			name: "bad-vars-template",
			v1: &V1{
				Vars: map[string]interface{}{
					// No quotes around `key`.
					"varkey": "hello, {{kargs kargkey}}",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				Kargs: tt.kargs,
				V1:    tt.v1,
			}
			if err := c.evaluateVars(); (err != nil) != tt.wantErr {
				t.Errorf("Config.evaluateVars() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_evaluateEnv(t *testing.T) {
	tests := []struct {
		name     string
		kargs    map[string]string
		v1       *V1
		expValue string
		wantErr  bool
	}{
		{
			name:  "success-env-template-uses-kargs-and-vars",
			kargs: map[string]string{"kargkey": "world"},
			v1: &V1{
				Vars: map[string]interface{}{
					"varkey": "world",
				},
				Env: map[string]string{
					"envkey": "hello, {{kargs `kargkey`}}; hello, {{.vars.varkey}}",
				},
			},
			expValue: "hello, world; hello, world",
			wantErr:  false,
		},
		{
			name:  "error-env-template",
			kargs: map[string]string{},
			v1: &V1{
				Env: map[string]string{
					// Attempt to use a kargs without quoting.
					"envkey": "{{kargs unquoted_key}}",
				},
			},
			// The value does not change.
			expValue: "{{kargs unquoted_key}}",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				Kargs: tt.kargs,
				V1:    tt.v1,
			}
			err := c.evaluateEnv()
			if (err != nil) != tt.wantErr || c.V1.Env["envkey"] != tt.expValue {
				t.Errorf("Config.evaluateEnv() error = %v, wantErr %v", err, tt.wantErr)
				t.Errorf("Config.evaluateEnv() got = %q, want %q", c.V1.Env["envkey"], tt.expValue)
			}
		})
	}
}

func TestConfig_evaluateCommands(t *testing.T) {
	tests := []struct {
		name     string
		kargs    map[string]string
		v1       *V1
		expValue []interface{}
		wantErr  bool
	}{
		{
			name: "success-template-replace-vars",
			v1: &V1{
				Vars: map[string]interface{}{
					"varkey": "varvalue",
				},
				Commands: []interface{}{
					"/bin/true {{.vars.varkey}}",
				},
			},
			expValue: []interface{}{
				[]interface{}{"/bin/true", "varvalue"},
			},
			wantErr: false,
		},
		{
			name: "success-commands-as-separate-args",
			v1: &V1{
				Vars: map[string]interface{}{
					"varkey": "varvalue",
				},
				Commands: []interface{}{
					[]interface{}{"/bin/true", "{{.vars.varkey}}"},
				},
			},
			expValue: []interface{}{
				[]interface{}{"/bin/true", "varvalue"},
			},
			wantErr: false,
		},
		{
			name: "error-incomplete-quote-in-command",
			v1: &V1{
				Commands: []interface{}{
					"/bin/true 'single quote is incomplete",
				},
			},
			expValue: []interface{}{
				// Unchanged.
				"/bin/true 'single quote is incomplete",
			},
			wantErr: true,
		},
		{
			name: "error-bad-template-in-string-command",
			v1: &V1{
				Commands: []interface{}{
					"/bin/true {{kargs missingquotes}}",
				},
			},
			expValue: []interface{}{
				// Unchanged.
				"/bin/true {{kargs missingquotes}}",
			},
			wantErr: true,
		},
		{
			name: "error-bad-template-in-array-command",
			v1: &V1{
				Commands: []interface{}{
					[]interface{}{"/bin/true", "{{kargs missingquotes}}"},
				},
			},
			expValue: []interface{}{
				// Unchanged.
				[]interface{}{"/bin/true", "{{kargs missingquotes}}"},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				Kargs: tt.kargs,
				V1:    tt.v1,
			}
			err := c.evaluateCommands()
			diff := pretty.Diff(c.V1.Commands, tt.expValue)
			if (err != nil) != tt.wantErr || len(diff) != 0 {
				t.Errorf("Config.evaluateCommands() error = %v, wantErr %v\nDiff: %#v",
					err, tt.wantErr, diff)
			}
		})
	}
}

func TestConfig_runCommands(t *testing.T) {
	tests := []struct {
		name    string
		v1      *V1
		wantErr bool
	}{
		{
			name: "success-with-comments",
			v1: &V1{
				Commands: []interface{}{
					"# This is a comment!",
					"/bin/true",
					"# So is this!",
				},
			},
			wantErr: false,
		},
		{
			name: "error-command-fails",
			v1: &V1{
				Commands: []interface{}{
					"/bin/false",
				},
			},
			wantErr: true,
		},
		{
			name: "error-bad-environment",
			v1: &V1{
				Env: map[string]string{
					"PATH": "/badpath",
				},
				Commands: []interface{}{
					"echo this should not work",
				},
			},
			wantErr: true,
		},
		{
			name: "success-echo-after-PATH-reset",
			v1: &V1{
				Commands: []interface{}{
					"true this *should* work",
				},
			},
			wantErr: false,
		},
		{
			name: "success-weird-variable-set-in-env",
			v1: &V1{
				Env: map[string]string{
					"SET_ONLY_DURING_COMMAND": "set",
				},
				Commands: []interface{}{
					"bash -c 'test -n \"$SET_ONLY_DURING_COMMAND\"'",
				},
			},
			wantErr: false,
		},
		{
			name: "success-weird-variable--unset-from-restored-env",
			v1: &V1{
				// $SET_ONLY_DURING_COMMAND should have been deleted.
				Commands: []interface{}{
					"bash -c 'test -z \"$SET_ONLY_DURING_COMMAND\"'",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				V1: tt.v1,
			}
			if err := c.runCommands(); (err != nil) != tt.wantErr {
				t.Errorf("Config.runCommands() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
