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
package extension

import (
	"io"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/renstrom/dedent"
)

func TestRequest_Encode(t *testing.T) {
	tests := []struct {
		name string
		v1   *V1
		want string
	}{
		{
			name: "encode-successful",
			v1: &V1{
				Hostname:    "mlab4.lga0t.measurement-lab.org",
				IPv4Address: "192.168.0.12",
				LastBoot:    time.Date(2018, 5, 1, 0, 0, 0, 0, time.UTC),
				// json.Marshal escapes '&' with '\u0026', which is why the
				// "want" value below has that value instead of an actual
				// ampersand: https://golang.org/pkg/encoding/json/#Marshal
				RawQuery: "p=somevalue&z=othervalue",
			},
			want: dedent.Dedent(`
        {
            "v1": {
                "hostname": "mlab4.lga0t.measurement-lab.org",
                "ipv4_address": "192.168.0.12",
                "ipv6_address": "",
                "last_boot": "2018-05-01T00:00:00Z",
                "raw_query": "p=somevalue\u0026z=othervalue"
            }
        }`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &Request{
				V1: tt.v1,
			}
			if got := req.Encode(); got != tt.want[1:] {
				t.Errorf("Request.Encode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRequest_Decode(t *testing.T) {
	tests := []struct {
		name     string
		msg      io.Reader
		wantErr  bool
		expected *V1
	}{
		{
			name: "decode-successful",
			msg: ioutil.NopCloser(strings.NewReader(dedent.Dedent(`
        {
            "v1": {
                "hostname": "mlab4.lga0t.measurement-lab.org",
                "ipv4_address": "192.168.0.12",
                "ipv6_address": "",
                "last_boot": "2018-05-01T00:00:00Z",
                "raw_query": "p=somevalue\u0026z=othervalue"
            }
        }`))),
			expected: &V1{
				Hostname:    "mlab4.lga0t.measurement-lab.org",
				IPv4Address: "192.168.0.12",
				LastBoot:    time.Date(2018, 5, 1, 0, 0, 0, 0, time.UTC),
				RawQuery:    "p=somevalue&z=othervalue",
			},
		},
		{
			name:    "decode-failure",
			msg:     ioutil.NopCloser(strings.NewReader(`{ THIS IS NOT VALID JSON " },`)),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &Request{}
			err := req.Decode(tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Request.Decode() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr &&
				(!tt.expected.LastBoot.Equal(req.V1.LastBoot) ||
					tt.expected.Hostname != req.V1.Hostname ||
					tt.expected.IPv4Address != req.V1.IPv4Address ||
					tt.expected.IPv6Address != req.V1.IPv6Address) {
				t.Errorf("Request.Decode() unexpected values: got %#v, want %#v",
					req.V1, tt.expected)
			}
		})
	}
}
