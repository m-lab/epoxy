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

package metrics

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/m-lab/epoxy/storage"
	"github.com/m-lab/go/prometheusx/promtest"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// fakeConfig emulates the storage.Config interface for unit tests.
type fakeConfig struct {
	host *storage.Host
}

// List returns a copy of the fakeConfig host.
func (f fakeConfig) List() ([]*storage.Host, error) {
	h := make([]*storage.Host, 1)
	h[0] = f.host
	return h, nil
}

func TestNewCollector(t *testing.T) {
	tests := []struct {
		name       string
		metricName string
		want       string
	}{
		{
			name:       "success",
			metricName: "epoxy_last_boot",
			want:       `epoxy_last_boot{machine="mlab1.foo01"}`,
		},
		{
			name:       "success",
			metricName: "epoxy_last_success",
			want:       `epoxy_last_success{machine="mlab1.foo01"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := fakeConfig{
				host: &storage.Host{
					Name:                "mlab1.foo01",
					LastSessionCreation: time.Now(),
					LastSuccess:         time.Now(),
				},
			}
			prometheus.MustRegister(NewCollector(tt.metricName, cfg))
			ts := httptest.NewServer(promhttp.Handler())
			resp, err := http.Get(ts.URL)
			if err != nil {
				t.Fatalf("Metrics request failed: %v", err)
			}
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read results: %v", err)
			}
			msg := string(b)
			lines := strings.Split(msg, "\n")
			success := false
			for i := range lines {
				if strings.HasPrefix(lines[i], tt.want) {
					success = true
					break
				}
			}
			if !success {
				t.Fatalf("Failed to find: %v", tt.want)
			}
		})
	}
}

func TestMetrics(t *testing.T) {
	// Lint the normal prometheus metrics.
	Stage1Total.WithLabelValues("x")
	RequestDuration.WithLabelValues("x")
	promtest.LintMetrics(t)
}
