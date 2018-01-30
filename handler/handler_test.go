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
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/m-lab/epoxy/storage"
)

// fakeConfig is a minimal Config implementation that emulates Host storage with a
// private field.
type fakeConfig struct {
	host *storage.Host
}

// Save copies the host parameter to the fakeConfig.
func (f fakeConfig) Save(host *storage.Host) error {
	*f.host = *host
	return nil
}

// Save returns a copy of the fakeConfig host.
func (f fakeConfig) Load(name string) (*storage.Host, error) {
	h := &storage.Host{}
	*h = *f.host
	return h, nil
}

// TestGenerateStage1IPXE performs an integration test with an httptest server and a
// fakeConfig providing Host storage.
func TestGenerateStage1IPXE(t *testing.T) {
	// Setup fake server.
	h := &storage.Host{
		Name:                "mlab1.iad1t.measurement-lab.org",
		IPAddress:           "165.117.240.9",
		Stage1to2ScriptName: "https://storage.googleapis.com/epoxy-boot-server/stage1to2/stage1to2.ipxe",
	}
	env := &Env{fakeConfig{h}, "example.com:4321"}
	router := mux.NewRouter()
	router.Methods("POST").
		Path("/v1/boot/{hostname}/stage1.ipxe").
		Handler(Handler{env, GenerateStage1IPXE})
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
		{"stage1to2_url", "storage.googleapis.com", "epoxy-boot-server/stage1to2/stage1to2.ipxe"},
		{"nextstage_url", "example.com:4321", h.CurrentSessionIDs.NextStageID},
		{"beginstage_url", "example.com:4321", h.CurrentSessionIDs.BeginStageID},
		{"endstage_url", "example.com:4321", h.CurrentSessionIDs.EndStageID},
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
