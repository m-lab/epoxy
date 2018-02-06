// Copyright 2016 ePoxy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//////////////////////////////////////////////////////////////////////////////

package handler

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/m-lab/epoxy/nextboot"
	"github.com/m-lab/epoxy/storage"
)

// fakeConfig is a minimal Config implementation that emulates Host storage with a
// private field.
type fakeConfig struct {
	host       *storage.Host
	failOnLoad bool
	failOnSave bool
}

// Save copies the host parameter to the fakeConfig.
func (f fakeConfig) Save(host *storage.Host) error {
	if f.failOnSave {
		return errors.New("Failed to save: " + host.Name)
	}
	*f.host = *host
	return nil
}

// Save returns a copy of the fakeConfig host.
func (f fakeConfig) Load(name string) (*storage.Host, error) {
	if f.failOnLoad {
		return nil, errors.New("Failed to load: " + name)
	}
	h := &storage.Host{}
	*h = *f.host
	return h, nil
}

// TestGenerateStage1IPXE performs an integration test with an httptest server and a
// fakeConfig providing Host storage.
func TestGenerateStage1IPXE(t *testing.T) {
	// Setup fake server.
	h := &storage.Host{
		Name:     "mlab1.iad1t.measurement-lab.org",
		IPv4Addr: "165.117.240.9",
		Boot: storage.Sequence{
			Stage1ChainURL: "https://storage.googleapis.com/epoxy-boot-server/stage1to2/stage1to2.ipxe",
		},
	}
	env := &Env{fakeConfig{host: h}, "example.com:4321"}
	router := mux.NewRouter()
	router.Methods("POST").
		Path("/v1/boot/{hostname}/stage1.ipxe").
		HandlerFunc(env.GenerateStage1IPXE)
	ts := httptest.NewServer(router)
	defer ts.Close()

	// Run client request.
	vals := url.Values{}
	u := ts.URL + "/v1/boot/mlab1.iad1t.measurement-lab.org/stage1.ipxe"

	resp, err := http.PostForm(u, vals)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("wrong status code: got %d want %d", resp.StatusCode, http.StatusOK)
	}

	// Read and parse response.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	script := string(body)
	if !strings.HasPrefix(script, "#!ipxe") {
		lines := strings.SplitN(script, "\n", 2)
		t.Errorf("Wrong iPXE script prefix: got %q want '#!ipxe'", lines[0])
	}
	// Parse the script response to verify generated URLs.
	urls := make(map[string]*url.URL)
	lines := strings.Split(script, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 3 {
			url, err := url.Parse(fields[2])
			if err != nil {
				t.Errorf("Failed to parse URL for %s: %q", fields[1], fields[2])
			}
			urls[fields[1]] = url
		}
	}

	// Define table of expected values.
	var urlChecks = []struct {
		name        string
		host        string
		partialPath string
	}{
		{"stage1chain_url", "storage.googleapis.com", "epoxy-boot-server/stage1to2/stage1to2.ipxe"},
		{"stage2_url", "example.com:4321", h.CurrentSessionIDs.Stage2ID},
		{"stage3_url", "example.com:4321", h.CurrentSessionIDs.Stage3ID},
		{"report_url", "example.com:4321", h.CurrentSessionIDs.ReportID},
	}
	// Assert that all expected values are found.
	for _, u := range urlChecks {
		if _, ok := urls[u.name]; !ok {
			t.Errorf("Missing variable in script: want %q\n", u.name)
		}
		url := urls[u.name]
		if u.host != url.Host {
			t.Errorf("Wrong host for variable %q; got %q, want %q\n", u.name, url.Host, u.host)
		}
		if !strings.Contains(url.Path, u.partialPath) {
			t.Errorf("Missing portion of URL path for variable %q; got %q, want %q\n",
				u.name, url.Path, u.partialPath)
		}
	}
}

func TestEnv_GenerateStage1IPXE(t *testing.T) {
	h := &storage.Host{
		Name:     "mlab1.iad1t.measurement-lab.org",
		IPv4Addr: "165.117.240.9",
		Boot: storage.Sequence{
			Stage1ChainURL: "https://storage.googleapis.com/epoxy-boot-server/stage1to2/stage1to2.ipxe",
		},
	}
	tests := []struct {
		name   string
		vars   map[string]string
		req    *http.Request
		config fakeConfig
		status int
	}{
		{
			name:   "okay",
			vars:   map[string]string{"hostname": h.Name},
			config: fakeConfig{host: h, failOnLoad: false, failOnSave: false},
			status: http.StatusOK,
		},
		{
			name:   "fail-on-load",
			vars:   map[string]string{"hostname": h.Name},
			config: fakeConfig{host: h, failOnLoad: true, failOnSave: false},
			status: http.StatusNotFound,
		},
		{
			name:   "fail-on-save",
			config: fakeConfig{host: h, failOnLoad: false, failOnSave: true},
			status: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := map[string]string{"hostname": h.Name}
			req := httptest.NewRequest("POST", "/v1/boot/"+h.Name+"/stage1.ipxe", nil)
			rec := httptest.NewRecorder()

			env := &Env{tt.config, "example.com:4321"}
			req = mux.SetURLVars(req, vars)
			env.GenerateStage1IPXE(rec, req)

			if rec.Code != tt.status {
				t.Errorf("GenerateStage1IPXE() wrong HTTP status: got %v; want %v", rec.Code, tt.status)
			}
		})
	}
}

func TestEnv_GenerateJSONConfig(t *testing.T) {
	h := &storage.Host{
		Name:     "mlab1.iad1t.measurement-lab.org",
		IPv4Addr: "165.117.240.9",
		Boot: storage.Sequence{
			Stage2ChainURL: "https://storage.googleapis.com/epoxy-boot-server/stage2/stage2.ipxe",
		},
		CurrentSessionIDs: storage.SessionIDs{
			Stage2ID: "12345",
		},
	}
	tests := []struct {
		name     string
		config   fakeConfig
		status   int
		expected string
	}{
		{
			name:     "okay",
			config:   fakeConfig{host: h, failOnLoad: false, failOnSave: false},
			status:   http.StatusOK,
			expected: (&nextboot.Config{V1: &nextboot.V1{Chain: h.Boot.Stage2ChainURL}}).String(),
		},
		{
			name:     "fail-on-load",
			config:   fakeConfig{host: h, failOnLoad: true, failOnSave: false},
			status:   http.StatusNotFound,
			expected: "Failed to load: mlab1.iad1t.measurement-lab.org\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := map[string]string{"hostname": h.Name, "sessionId": h.CurrentSessionIDs.Stage2ID}
			path := "/v1/boot/mlab1.iad1t.measurement-lab.org/12345/stage2"
			req := httptest.NewRequest("POST", path, nil)
			rec := httptest.NewRecorder()

			env := &Env{tt.config, "server.com:4321"}
			req = mux.SetURLVars(req, vars)
			env.GenerateJSONConfig(rec, req)

			if rec.Code != tt.status {
				t.Errorf("GenerateJSONConfig() wrong HTTP status: got %v; want %v", rec.Code, tt.status)
			}
			if tt.expected != rec.Body.String() {
				t.Errorf("GenerateJSONConfig() wrong response: got %v\n; want %v\n", rec.Body.String(), tt.expected)
			}
		})
	}
}

func TestEnv_ReceiveReport(t *testing.T) {
	h := &storage.Host{
		Name:     "mlab1.iad1t.measurement-lab.org",
		IPv4Addr: "165.117.240.9",
		CurrentSessionIDs: storage.SessionIDs{
			ReportID: "12345",
		},
	}
	tests := []struct {
		name   string
		status int
	}{
		{
			name:   "place-holder",
			status: http.StatusNoContent,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := map[string]string{"hostname": h.Name, "sessionId": h.CurrentSessionIDs.Stage2ID}
			path := "/v1/boot/mlab1.iad1t.measurement-lab.org/12345/report"
			req := httptest.NewRequest("POST", path, nil)
			rec := httptest.NewRecorder()

			env := &Env{fakeConfig{host: h, failOnLoad: false, failOnSave: false}, "server.com:4321"}
			req = mux.SetURLVars(req, vars)
			env.ReceiveReport(rec, req)

			if rec.Code != tt.status {
				t.Errorf("GenerateJSONConfig() wrong HTTP status: got %v; want %v", rec.Code, tt.status)
			}

		})
	}
}
