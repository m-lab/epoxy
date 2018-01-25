package nextboot

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/kr/pretty"
	"github.com/renstrom/dedent"
)

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
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("some", "header")
			w.WriteHeader(http.StatusNoContent)
			err := r.ParseForm()
			if err != nil {
				t.Fatal(err)
			}
			pretty.Print(r.PostForm)
		}))
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
		// TODO: Add test cases.
		{
			"basic",
			map[string]string{
				"epoxy.report": ts.URL,
			},
			args{
				"epoxy.report",
				url.Values{},
			},
			false,
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
