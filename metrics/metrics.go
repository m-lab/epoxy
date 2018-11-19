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

// Package metrics contains prometheus metric definitions for the epoxy server.
package metrics

import (
	"log"

	"github.com/m-lab/epoxy/storage"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Stage1Total counts the number of host boots.
	Stage1Total = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "epoxy_stage1_total",
			Help: "Total number of boots per machine.",
		},
		// Machine name.
		[]string{"machine"},
	)

	// RequestDuration profiles request latency.
	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "epoxy_request_duration_seconds",
			Help: "A histogram of request latencies.",
			// Note: use default buckets.
		},
		[]string{"code"},
	)
)

func init() {
	prometheus.MustRegister(Stage1Total)
	prometheus.MustRegister(RequestDuration)
}

// Config provides access to Host records.
type Config interface {
	List() ([]*storage.Host, error)
}

// Collector defines a custom collector for reading metrics from datastore.
type Collector struct {
	name   string
	desc   *prometheus.Desc
	config Config
}

// NewCollector creates a new datastore collector instance. The metricName should
// be one of "epoxy_last_boot" or "epoxy_last_success".
func NewCollector(metricName string, config Config) *Collector {
	return &Collector{
		name:   metricName,
		desc:   nil,
		config: config,
	}
}

// Describe satisfies the prometheus.Collector interface. Describe is called
// immediately after registering the collector.
func (col *Collector) Describe(ch chan<- *prometheus.Desc) {
	if col.desc == nil {
		col.desc = prometheus.NewDesc(col.name, "The last timestamp for "+col.name, []string{"machine"}, nil)
	}
	ch <- col.desc
}

// Collect satisfies the prometheus.Collector interface. Collect reports values
// from hosts datastore.
func (col *Collector) Collect(ch chan<- prometheus.Metric) {
	hosts, err := col.config.List()
	if err != nil {
		log.Println("Failed to list hosts", err)
		return
	}
	for i := range hosts {
		var ts float64
		if hosts[i].LastSessionCreation.IsZero() || hosts[i].LastSuccess.IsZero() {
			// Skip reporting metrics for hosts that have never booted.
			continue
		}
		switch col.name {
		case "epoxy_last_boot":
			ts = float64(hosts[i].LastSessionCreation.UnixNano()) / 1e9
		case "epoxy_last_success":
			ts = float64(hosts[i].LastSuccess.UnixNano()) / 1e9
		default:
			log.Println("Unknown collector name:", col.name)
			return
		}
		ch <- prometheus.MustNewConstMetric(
			col.desc, prometheus.GaugeValue, ts, hosts[i].Name)
	}
}
