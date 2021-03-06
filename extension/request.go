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

// Package extension defines the Extension API used between the ePoxy
// server and extension services.
package extension

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"time"
)

// Request contains information about a booting machine
type Request struct {

	// V1 contains information to send to an extension service.
	V1 *V1 `json:"v1,omitempty"`
}

// V1 contains information about a booting machine. The ePoxy server guarantees
// that a booting machine is registered and all requests have used valid session IDs.
type V1 struct {
	// Hostname is the FQDN for the booting machine.
	Hostname string `json:"hostname"`

	// IPv4Address is the IPv4 address the booting machine.
	IPv4Address string `json:"ipv4_address"`

	// IPv6Address is the IPv6 address the booting machine.
	IPv6Address string `json:"ipv6_address"`

	// LastBoot is the most recent time when the booting machine reached stage1.
	LastBoot time.Time `json:"last_boot"`

	// The raw query string from the request to ePoxy. Extensions may use this
	// to extract arbitrary data sent by the client.
	RawQuery string `json:"raw_query"`
}

// Encode marshals a Request to JSON.
func (req *Request) Encode() string {
	// Errors only occur for non-UTF8 characters in strings or unmarshalable types (which we don't have).
	b, _ := json.MarshalIndent(req, "", "    ")
	return string(b)
}

// Decode unmarshals a Request from a JSON message.
func (req *Request) Decode(msg io.Reader) error {
	raw, err := ioutil.ReadAll(msg)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, req)
}
