// Copyright 2021 ePoxy Authors
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

package command

import (
	"fmt"
	"net"
	"testing"

	"github.com/m-lab/epoxy/storage"
)

func TestSync_getV4Address(t *testing.T) {
	tests := []struct {
		name     string
		ips      []net.IP
		err      error
		expected string
		wanterr  bool
	}{
		{
			name: "one-ipv4",
			ips: []net.IP{
				net.ParseIP("2005:170:1100:6d::25"),
				net.ParseIP("10.0.0.15"),
			},
			err:      nil,
			expected: "10.0.0.15",
			wanterr:  false,
		},
		{
			name: "two-ipv4",
			ips: []net.IP{
				net.ParseIP("192.168.5.96"),
				net.ParseIP("10.0.0.15"),
			},
			err:      nil,
			expected: "192.168.5.96",
			wanterr:  false,
		},
		{
			name: "no-ipv4",
			ips: []net.IP{
				net.ParseIP("2005:1700:1100:6d::25"),
				net.ParseIP("2903:f50b:4100::11"),
				net.ParseIP("2022:2050:0:29::101"),
			},
			err:      nil,
			expected: "",
			wanterr:  true,
		},
		{
			name:     "no-ips",
			ips:      []net.IP{},
			err:      fmt.Errorf("Host not found"),
			expected: "",
			wanterr:  true,
		},
	}

	for _, tt := range tests {
		lookupIP = func(host string) ([]net.IP, error) {
			return tt.ips, tt.err
		}
		v4, err := getV4Address("fake-host")
		if tt.wanterr {
			if err == nil {
				t.Errorf("getV4Address(): expected an error, but got: %v", err)
			}
		}
		if v4 != tt.expected {
			t.Errorf("getV4Address(): expected IPv4 %s, but got %s", tt.expected, v4)
		}
	}

}

func TestSync_isHostnameinDatastore(t *testing.T) {
	entities := []*storage.Host{
		{
			Name: "mlab1-xyz0t.mlab-sandbox.measurement-lab.org",
		},
		{
			Name: "mlab3-lol05.mlab-staging.measurement-lab.org",
		},
		{
			Name: "mlab2-abc01.mlab-sandbox.measurement-lab.org",
		},
		{
			Name: "mlab4-usa02.mlab-staging.measurement-lab.org",
		},
		{
			Name: "mlab1-xyz01.mlab-sandbox.measurement-lab.org",
		},
	}

	tests := []struct {
		name     string
		hostname string
		found    bool
	}{
		{
			name:     "found-hostname",
			hostname: "mlab4-usa02.mlab-staging.measurement-lab.org",
			found:    true,
		},
		{
			name:     "not-found-hostname",
			hostname: "mlab9-ddd01.mlab-staging.measurement-lab.org",
			found:    false,
		},
		{
			name:     "no-hostname",
			hostname: "",
			found:    false,
		},
	}

	for _, tt := range tests {
		found := isHostnameInDatastore(tt.hostname, entities)
		if found != tt.found {
			t.Errorf("isHostnameInDatastore(): wanted %v, got %v", tt.found, found)
		}
	}
}
