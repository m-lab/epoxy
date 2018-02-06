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
	"time"
)

// These variables provide indirection for the default function implementations.
// Each can be reassigned with an alternate implementation for unit tests.
var (
	randRead = rand.Read
	timeNow  = time.Now
)

// CollectedInformation stores information received directly from iPXE clients.
type CollectedInformation struct {
	Platform         string
	BuildArch        string
	Serial           string
	Asset            string
	UUID             string
	Manufacturer     string
	Product          string
	Chip             string
	MAC              string
	IP               string
	Version          string
	PublicSSHHostKey string
}

// SessionIDs contains the three session IDs generated when requesting a stage1 target.
type SessionIDs struct {
	Stage2ID string // Needed for requesting the nextstage.json target.
	Stage3ID string // Needed for requesting the begingstage target.
	ReportID string // Needed for requesting the endstage target.
}

// Sequence ...
type Sequence struct {
	// Stage1ChainURL is the absolute URL to an iPXE script for booting stage1 to stage2.
	Stage1ChainURL string // boot stage2 image
	// Stage2ChainURL is the absolute URL to a JSON config for booting stage2 to stage3.
	Stage2ChainURL string // boot stage3 update image, or coreos
	// Stage3ChainURL is the absolute URL to a JSON config for running commands in stage3.
	Stage3ChainURL string // flashrom, or join global k8s cluster.
}

func (s Sequence) NextURL(stage string) string {
	switch stage {
	case "stage1":
		return s.Stage1ChainURL
	case "stage2":
		return s.Stage2ChainURL
	case "stage3":
		return s.Stage3ChainURL
	default:
		// TODO: support a default error url.
		return ""
	}
}

// A Host represents the configuration of a server managed by ePoxy.
type Host struct {
	// Name is the FQDN of the host.
	Name string
	// IPv4Addr is the IPv4 address the booting machine will use to connect to the API.
	IPv4Addr string

	Boot   Sequence
	Update Sequence

	// UpdateEnabled controls whether ePoxy returns the Update Sequence (true)
	// or Boot Sequence (false) Chain URLs.
	UpdateEnabled bool

	// CurrentSessionIDs are the most recently generated session ids for a booting machine.
	CurrentSessionIDs SessionIDs
	// LastSessionCreation is the time when CurrentSessionIDs was generated.
	LastSessionCreation time.Time
	// CollectedInformation reported by the host.
	CollectedInformation CollectedInformation
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
	h.LastSessionCreation = timeNow()
}

// CurrentSequence returns the currently enabled boot sequence.
func (h *Host) CurrentSequence() Sequence {
	if h.UpdateEnabled {
		return h.Update
	}
	return h.Boot
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
