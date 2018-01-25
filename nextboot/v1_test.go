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
						"/bin/echo ok",
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
				            "/bin/echo ok"
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
	chainfmt := `{"v1": {"chain": "%s"}}`
	cmd := `{"v1": {"commands": ["/bin/echo okay"]}}`
	tests := []struct {
		name       string
		action     string
		statusPost int
		statusGet  int
		wantErr    bool
	}{
		{
			name:       "successful-post-chain-then-get-commands",
			action:     "epoxy.stage2",
			statusPost: http.StatusOK,
			statusGet:  http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "bad-action-key",
			action:     "epoxy.wrongkey",
			statusPost: http.StatusNotFound,
			wantErr:    true,
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
			ts2 := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.statusGet)
					w.Write([]byte(cmd))
				}))
			ts1 := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.statusPost)
					w.Write([]byte(fmt.Sprintf(chainfmt, ts2.URL)))
				}))
			c := &Config{
				Kargs: map[string]string{"epoxy.stage2": ts1.URL},
			}
			if err := c.Run(tt.action, false); (err != nil) != tt.wantErr {
				t.Errorf("Config.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
			ts1.Close()
			ts2.Close()
		})
	}
}
