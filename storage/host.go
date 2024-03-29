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

// Package storage includes the Host record definition. Host records represent
// a managed machine and store the next stage configuration. Host records are
// saved to persistent storage.
package storage

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/m-lab/epoxy/datastorex"
)

// These variables provide indirection for the default function implementations.
// Each can be reassigned with an alternate implementation for unit tests.
var (
	randRead = rand.Read
	timeNow  = time.Now
)

// allowedCollectedInformation contains the complete set of keys that a client may
// provid to be saved in the Host record.
var allowedCollectedInformation = map[string]bool{
	"platform":            true,
	"buildarch":           true,
	"serial":              true,
	"asset":               true,
	"uuid":                true,
	"manufacturer":        true,
	"product":             true,
	"chip":                true,
	"mac":                 true,
	"ip":                  true,
	"version":             true,
	"public_ssh_host_key": true,
}

// Constant names for standard boot & update sequence maps.
const (
	Stage1IPXE = "stage1.ipxe"
	Stage1JSON = "stage1.json"
	Stage2     = "stage2"
	Stage3     = "stage3"
)

// TODO: SessionIDs structs should be map[string]string, that
// store target stage names as keys. This prevents hard-coding the target names,
// the SessionID names.

// SessionIDs contains the three session IDs generated when requesting a stage1 target.
type SessionIDs struct {
	Stage2ID string // Needed for requesting the stage2.json target.
	Stage3ID string // Needed for requesting the stage3.json target.
	ReportID string // Needed for requesting the report target.
	// TODO: support multiple extensions.
	ExtensionID string // Needed for requesting the extension target.
}

// A Host represents the configuration of a server managed by ePoxy.
type Host struct {
	// Name is the FQDN of the host.
	Name string
	// IPv4Addr is the IPv4 address the booting machine will use to connect to the API.
	IPv4Addr string

	// TODO: add IPv6Addr.

	// Boot is the typical boot sequence for this Host.
	Boot datastorex.Map
	// Update is an alternate boot sequence, typically used to update the system, e.g. reinstall, reflash.
	Update datastorex.Map
	// ImagesVersion is the version of epoxy-images to use when booting the
	// machines in all stages (1-3).
	ImagesVersion string

	// UpdateEnabled controls whether ePoxy returns the Update sequence (true)
	// or Boot sequence (false) Chain URLs.
	UpdateEnabled bool

	// Extensions is an array of extension operation names enabled for this host.
	Extensions []string

	// CurrentSessionIDs are the most recently generated session ids for a booting machine.
	CurrentSessionIDs SessionIDs
	// LastSessionCreation is the time when CurrentSessionIDs was generated.
	LastSessionCreation time.Time
	// LastReport is the time of the most recent report for this host.
	LastReport time.Time
	// LastSuccess is the time of the most recent successful report from this host.
	LastSuccess time.Time
	// CollectedInformation reported by the host. CollectedInformation must be non-nil.
	CollectedInformation datastorex.Map
}

// String serializes a Host record. All string type Host fields should be UTF8.
func (h *Host) String() string {
	// Errors only occur for non-UTF8 characters in strings or unmarshalable types (which we don't have).
	b, _ := json.MarshalIndent(h, "", "    ")
	return string(b)
}

// GenerateSessionIDs creates new random session IDs for the host's CurrentSessionIDs.
// On success, the host LastSessionCreation is updated to the current time.
func (h *Host) GenerateSessionIDs() {
	h.CurrentSessionIDs.Stage2ID = generateSessionID()
	h.CurrentSessionIDs.Stage3ID = generateSessionID()
	h.CurrentSessionIDs.ReportID = generateSessionID()
	h.CurrentSessionIDs.ExtensionID = generateSessionID()
	h.LastSessionCreation = timeNow()
}

// CurrentSequence returns the currently enabled boot sequence.
func (h *Host) CurrentSequence() datastorex.Map {
	if h.UpdateEnabled {
		return h.Update
	}
	return h.Boot
}

// AddInformation adds values to the Host's CollectedInformation. Only key
// names in CollectedInformationWhitelist will be added.
func (h *Host) AddInformation(values url.Values) {
	for key, values := range values {
		value := strings.TrimSpace(strings.Join(values, " "))
		if !utf8.ValidString(value) {
			log.Printf("Skipping invalid value for: %s CollectedInformation.%s\n", h.Name, key)
			continue
		}
		if allowedCollectedInformation[key] && value != "" {
			h.CollectedInformation[key] = value
		}
	}
}

// randomSessionByteCount is the number of bytes used to generate random session IDs.
const randomSessionByteCount = 20

// generateSessionId creates a random session ID.
func generateSessionID() string {
	b := make([]byte, randomSessionByteCount)
	_, err := randRead(b)
	if err != nil {
		// Only possible if randRead fails to read len(b) bytes.
		panic(err)
	}
	// RawURLEncoding does not pad encoded string with "=".
	return base64.RawURLEncoding.EncodeToString(b)
}
