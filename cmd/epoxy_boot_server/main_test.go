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
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/m-lab/epoxy/storage"
	"github.com/prometheus/prometheus/util/promlint"
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

// fakeDatastoreClient implements the datastoreClient interface for testing.
// Every operation should be successful.
type fakeDatastoreClient struct {
	host *storage.Host
}

func (f *fakeDatastoreClient) Get(ctx context.Context, key *datastore.Key, dst interface{}) error {
	return fmt.Errorf("this fake does not support Get()")
}
func (f *fakeDatastoreClient) Put(ctx context.Context, key *datastore.Key, src interface{}) (*datastore.Key, error) {
	return nil, fmt.Errorf("this fake does not support Put()")
}
func (f *fakeDatastoreClient) GetAll(ctx context.Context, q *datastore.Query, dst interface{}) ([]*datastore.Key, error) {
	// Extract the pointer to a list of *storage.Host, and append f.host to the list.
	hosts, _ := dst.(*[]*storage.Host)
	*hosts = append(*hosts, f.host)
	return nil, nil
}

func Test_setupMetricsHandler(t *testing.T) {
	dsCfg := &storage.DatastoreConfig{
		Client: &fakeDatastoreClient{
			host: &storage.Host{
				Name:                "mlab1.iad1t.measurement-lab.org",
				IPv4Addr:            "165.117.240.9",
				LastSuccess:         time.Now(),
				LastSessionCreation: time.Now(),
			},
		},
	}
	mux := setupMetricsHandler(dsCfg)
	srv := httptest.NewServer(mux)

	metricReader, err := http.Get(srv.URL + "/metrics")
	if err != nil || metricReader == nil {
		t.Errorf("Could not GET metrics: %v", err)
	}
	metricBytes, err := ioutil.ReadAll(metricReader.Body)
	if err != nil {
		t.Errorf("Could not read metrics: %v", err)
	}
	metricsLinter := promlint.New(bytes.NewBuffer(metricBytes))
	problems, err := metricsLinter.Lint()
	if err != nil {
		t.Errorf("Could not lint metrics: %v", err)
	}
	for _, p := range problems {
		t.Errorf("Bad metric %v: %v", p.Metric, p.Text)
	}
}

func Test_main(t *testing.T) {
	projectID = "mlab-testing"
	publicHostname = "fake.public.hostname.com"
	bindAddress = "localhost"
	bindPort = "8800"
	os.Setenv("GAE_SERVICE", "fake") // Simulate deployment in AE.
	go main()
	time.Sleep(1 * time.Second)
	// TODO: replace test with actual datastore records.
	//       "/v1/boot/mlab1.foo01.measurement-lab.org/stage1.ixpe"
	health, err := http.Get("http://localhost:8800/_ah/health")
	if err != nil || health == nil {
		t.Errorf("Could not GET health: %v", err)
	}
	bytes, err := ioutil.ReadAll(health.Body)
	if err != nil {
		t.Errorf("Could not read health: %v", err)
	}
	if string(bytes) != "ok" {
		t.Errorf("Could not read health: got %q, want 'ok'", string(bytes))
	}
	cancelCtx()
}
