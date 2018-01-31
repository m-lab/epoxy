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
	NextStageID  string // Needed for requesting the nextstage.json target.
	BeginStageID string // Needed for requesting the begingstage target.
	EndStageID   string // Needed for requesting the endstage target.
}

// A Host represents the configuration of a server managed by ePoxy.
type Host struct {
	// Name is the FQDN of the host.
	Name string
	// IPAddress is the IP address the booting machine will use to connect to the API.
	IPAddress string
	// Stage1to2ScriptName is the absolute URL to an iPXE script for booting stage1 to stage2.
	Stage1to2ScriptName string
	// NextStageEnabled controls whether ePoxy returns the NextStageScriptName (true)
	// or DefaultScriptName (false).
	NextStageEnabled bool
	// NextStageScriptName is the absolute URL of a JSON next stage configuration.
	NextStageScriptName string
	// DefaultScriptName is the absolute URL of a JSON default configuration.
	DefaultScriptName string
	// LastSessionIDs are the most recently generated session ids for a booting machine.
	CurrentSessionIDs SessionIDs
	// LastSessionCreation is the time when CurrentSessionIDs was generated.
	LastSessionCreation time.Time
	// Information reported by the host.
	CollectedInformation CollectedInformation
}

// String serializes a Host record. All string type Host fields should be UTF8.
func (h *Host) String() string {
	// Errors only occur for non-UTF8 characters in strings.
	b, _ := json.MarshalIndent(h, "", "    ")
	return string(b)
}

// GenerateSessionIDs creates new random session IDs for the host's CurrentSessionIDs.
// On success, the host LastSessionCreation is updated to the current time.
func (h *Host) GenerateSessionIDs() error {
	var err error
	h.CurrentSessionIDs.NextStageID, err = generateSessionID()
	if err != nil {
		return err
	}
	h.CurrentSessionIDs.BeginStageID, err = generateSessionID()
	if err != nil {
		return err
	}
	h.CurrentSessionIDs.EndStageID, err = generateSessionID()
	if err != nil {
		return err
	}
	h.LastSessionCreation = timeNow()
	return nil
}

// randomSessionByteCount is the number of bytes used to generate random session IDs.
const randomSessionByteCount = 20

// generateSessionId creates a random session ID.
func generateSessionID() (string, error) {
	b := make([]byte, randomSessionByteCount)
	_, err := randRead(b)
	if err != nil {
		// Only possible if randRead fails to read len(b) bytes.
		return "", err
	}
	// RawURLEncoding does not pad encoded string with "=".
	return base64.RawURLEncoding.EncodeToString(b), nil
}
