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

package main

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCheckHealth(t *testing.T) {
	r := httptest.NewRequest("GET", "/_ah/health", nil)
	w := httptest.NewRecorder()

	checkHealth(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("wrong HTTP status: got %d want %d", w.Code, http.StatusOK)
	}
	if w.Body.String() != "ok" {
		t.Errorf("wrong health response: got %s want 'ok'", w.Body.String())
	}
}

func TestGenerateStage2IPXE(t *testing.T) {
	r := newRouter()
	ts := httptest.NewServer(r)
	defer ts.Close()

	// TODO(soltesz): simulate the values POSTed by an iPXE client.
	vals := url.Values{}
	u := ts.URL + "/v1/boot/example.com/stage2.ipxe"

	resp, err := http.PostForm(u, vals)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("wrong status code: got %d want %d", resp.StatusCode, http.StatusOK)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(body), "#!ipxe") {
		t.Errorf("wrong script prefix: got '%s' want '#!ipxe'", body[:6])
	}
}
