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
	"time"

	"github.com/gorilla/mux"
	"github.com/m-lab/epoxy/extension"
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
	env := &Env{fakeConfig{host: h}, "example.com:4321", true}
	router := mux.NewRouter()
	router.Methods("POST").
		Path("/v1/boot/{hostname}/stage1.ipxe").
		HandlerFunc(env.GenerateStage1IPXE)
	ts := httptest.NewServer(router)
	defer ts.Close()

	// Run client request.
	vals := url.Values{}
	path := ts.URL + "/v1/boot/mlab1.iad1t.measurement-lab.org/stage1.ipxe"

	req, err := http.NewRequest("POST", path, strings.NewReader(vals.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Forwarded-For", h.IPv4Addr)
	resp, err := http.DefaultClient.Do(req)
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
		from   string
		status int
	}{
		{
			name:   "okay",
			vars:   map[string]string{"hostname": h.Name},
			config: fakeConfig{host: h, failOnLoad: false, failOnSave: false},
			from:   h.IPv4Addr,
			status: http.StatusOK,
		},
		{
			name:   "fail-on-load",
			vars:   map[string]string{"hostname": h.Name},
			config: fakeConfig{host: h, failOnLoad: true, failOnSave: false},
			from:   h.IPv4Addr,
			status: http.StatusNotFound,
		},
		{
			name:   "fail-on-save",
			config: fakeConfig{host: h, failOnLoad: false, failOnSave: true},
			from:   h.IPv4Addr,
			status: http.StatusInternalServerError,
		},
		{
			name:   "fail-from-wrong-ip",
			config: fakeConfig{host: h},
			from:   "192.168.0.1",
			status: http.StatusForbidden,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := map[string]string{"hostname": h.Name}
			req := httptest.NewRequest("POST", "/v1/boot/"+h.Name+"/stage1.ipxe", nil)
			req.Header.Set("X-Forwarded-For", tt.from)
			rec := httptest.NewRecorder()

			env := &Env{tt.config, "example.com:4321", true}
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
		from     string
		expected string
	}{
		{
			name:     "okay",
			config:   fakeConfig{host: h, failOnLoad: false, failOnSave: false},
			status:   http.StatusOK,
			from:     h.IPv4Addr,
			expected: (&nextboot.Config{V1: &nextboot.V1{Chain: h.Boot.Stage2ChainURL}}).String(),
		},
		{
			name:     "fail-on-load",
			config:   fakeConfig{host: h, failOnLoad: true, failOnSave: false},
			status:   http.StatusNotFound,
			from:     h.IPv4Addr,
			expected: "Failed to load: mlab1.iad1t.measurement-lab.org\n",
		},
		{
			name:     "fail-from-wrong-ip",
			config:   fakeConfig{host: h},
			status:   http.StatusForbidden,
			from:     "192.168.0.1",
			expected: "Caller cannot access host\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := map[string]string{"hostname": h.Name, "sessionID": h.CurrentSessionIDs.Stage2ID}
			path := "/v1/boot/mlab1.iad1t.measurement-lab.org/12345/stage2"
			req := httptest.NewRequest("POST", path, nil)
			req.Header.Set("X-Forwarded-For", tt.from)
			rec := httptest.NewRecorder()

			env := &Env{tt.config, "server.com:4321", true}
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
		name            string
		sessionID       string
		from            string
		expectedStatus  int
		expectedEnabled bool
		form            url.Values
	}{
		{
			name:            "disable-update-enabled-on-success",
			sessionID:       "12345",
			from:            h.IPv4Addr,
			expectedStatus:  http.StatusNoContent,
			expectedEnabled: false,
			form: url.Values{
				"message": []string{"success"},
			},
		},
		{
			name:            "preserve-update-enabled-on-failure",
			sessionID:       "12345",
			from:            h.IPv4Addr,
			expectedStatus:  http.StatusNoContent,
			expectedEnabled: true,
			form: url.Values{
				"message": []string{"error: something failed"},
			},
		},
		{
			name:            "bad-session-returns-forbidden",
			sessionID:       "mismatched-session-id",
			from:            h.IPv4Addr,
			expectedStatus:  http.StatusForbidden,
			expectedEnabled: true,
			form:            url.Values{},
		},
		{
			name:            "fail-from-wrong-ip",
			sessionID:       "12345",
			from:            "192.168.0.1",
			expectedStatus:  http.StatusForbidden,
			expectedEnabled: true,
			form:            url.Values{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := map[string]string{"hostname": h.Name, "sessionID": tt.sessionID}
			path := "/v1/boot/mlab1.iad1t.measurement-lab.org/12345/report"
			h.UpdateEnabled = true

			req := httptest.NewRequest("POST", path, strings.NewReader(tt.form.Encode()))
			// Mark the body as form content to be read by ParseForm.
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("X-Forwarded-For", tt.from)
			rec := httptest.NewRecorder()

			env := &Env{fakeConfig{host: h, failOnLoad: false, failOnSave: false}, "server.com:4321", true}
			req = mux.SetURLVars(req, vars)
			env.ReceiveReport(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("ReceiveReport() wrong HTTP status: got %v; want %v", rec.Code, tt.expectedStatus)
			}

			if h.UpdateEnabled != tt.expectedEnabled {
				t.Errorf("ReceiveReport() failed to change UpdateEnabled: got %t; want %t",
					h.UpdateEnabled, tt.expectedEnabled)
			}
		})
	}
}

func TestEnv_HandleExtension(t *testing.T) {
	// Generic Host record for all tests.
	h := &storage.Host{
		Name:     "mlab1.iad1t.measurement-lab.org",
		IPv4Addr: "165.117.240.9",
		CurrentSessionIDs: storage.SessionIDs{
			ExtensionID: "12345",
		},
		LastSessionCreation: time.Date(2018, 5, 1, 0, 0, 0, 0, time.UTC),
	}
	// The request that should be received by the extension server.
	expectedRequest := &extension.Request{
		V1: &extension.V1{
			Hostname:    h.Name,
			IPv4Address: h.IPv4Addr,
			LastBoot:    h.LastSessionCreation,
		},
	}
	tests := []struct {
		name            string
		sessionID       string
		operation       string
		failOnLoad      bool
		urlPrefix       string
		from            string
		expectedStatus  int
		expectedResult  string
		expectedRequest *extension.Request
	}{
		{
			name:            "successful-request",
			sessionID:       "12345",
			operation:       "foobar",
			from:            h.IPv4Addr,
			expectedStatus:  http.StatusOK,
			expectedResult:  "okay",
			expectedRequest: expectedRequest,
		},
		{
			name:            "failure-backend-returns-notfound",
			sessionID:       "12345",
			operation:       "foobar",
			from:            h.IPv4Addr,
			expectedStatus:  http.StatusNotFound,
			expectedResult:  "not found",
			expectedRequest: expectedRequest,
		},
		{
			name:           "failure-failonload",
			failOnLoad:     true,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "failure-bad-sessionid",
			sessionID:      "54321",
			from:           h.IPv4Addr,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "failure-from-wrong-ip",
			sessionID:      "54321",
			from:           "192.168.0.1",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "failure-zerolength-operation",
			sessionID:      "12345",
			operation:      "",
			from:           h.IPv4Addr,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "failure-unknown-operation",
			sessionID:      "12345",
			operation:      "unknown",
			from:           h.IPv4Addr,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "failure-parsing-extension-url",
			sessionID:      "12345",
			operation:      "foobar",
			from:           h.IPv4Addr,
			urlPrefix:      ":", // with this character, the backend URL will fail to parse.
			expectedStatus: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Pseudo variables for mux.Vars.
			vars := map[string]string{
				"hostname":  h.Name,
				"sessionID": tt.sessionID,
				"operation": tt.operation,
			}
			extURL := "/v1/boot/mlab1.iad1t.measurement-lab.org/12345/extension/foobar"

			req := httptest.NewRequest("POST", extURL, nil)
			req.Header.Set("X-Forwarded-For", tt.from)
			rec := httptest.NewRecorder()
			env := &Env{fakeConfig{host: h, failOnLoad: tt.failOnLoad}, "server.com:4321", true}
			req = mux.SetURLVars(req, vars)
			// Setup a fake extension server to handle the Request.
			ts := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					ext := &extension.Request{}
					err := ext.Decode(r.Body)
					if err != nil {
						// Decode failed, bad request.
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					// Decode was successful, so make sure it's what we expect.
					if !tt.expectedRequest.V1.LastBoot.Equal(ext.V1.LastBoot) ||
						tt.expectedRequest.V1.Hostname != ext.V1.Hostname ||
						tt.expectedRequest.V1.IPv4Address != ext.V1.IPv4Address ||
						tt.expectedRequest.V1.IPv6Address != ext.V1.IPv6Address {
						t.Errorf("HandleExtension() malformed request: got %#v, want %#v",
							ext.V1, tt.expectedRequest.V1)
					}
					// Unconditionally report the test-defined status.
					w.WriteHeader(tt.expectedStatus)
					w.Write([]byte(tt.expectedResult))
				}))
			defer ts.Close()
			// TODO: this modifies a global variable, which may have side-effects.
			// This will be eliminated once Extensions are read from datastore.
			storage.Extensions["foobar"] = tt.urlPrefix + ts.URL

			// Run the extension handler.
			env.HandleExtension(rec, req)

			if tt.expectedStatus != rec.Code {
				t.Errorf("HandleExtension() wrong HTTP status: got %v; want %v",
					rec.Code, tt.expectedStatus)
			}
			if tt.expectedResult != "" && tt.expectedResult != rec.Body.String() {
				t.Errorf("HandleExtension() wrong result forwarded: got %v\n; want %v\n",
					rec.Body.String(), tt.expectedResult)
			}
		})
	}
}
