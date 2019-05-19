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

package storage

import (
	"log"
	"testing"
	"time"

	"github.com/m-lab/epoxy/datastorex"
)

func TestHostString(t *testing.T) {
	hostExpected := `{
    "Name": "mlab1.iad1t.measurement-lab.org",
    "IPv4Addr": "165.117.240.9",
    "Boot": {
        "stage1.ipxe": "https://storage.googleapis.com/epoxy-boot-server/coreos/stage1to2.ipxe",
        "stage2": "https://storage.googleapis.com/epoxy-boot-server/coreos/stage2to3.json",
        "stage3": "https://storage.googleapis.com/epoxy-boot-server/coreos/stage3setup.json"
    },
    "Update": {
        "stage1.ipxe": "https://storage.googleapis.com/epoxy-boot-server/centos6/install.json",
        "stage2": "https://storage.googleapis.com/epoxy-boot-server/centos6/boot.json",
        "stage3": ""
    },
    "UpdateEnabled": false,
    "Extensions": null,
    "CurrentSessionIDs": {
        "Stage2ID": "01234",
        "Stage3ID": "56789",
        "ReportID": "13579",
        "ExtensionID": ""
    },
    "LastSessionCreation": "2016-01-02T15:04:00Z",
    "LastReport": "0001-01-01T00:00:00Z",
    "LastSuccess": "0001-01-01T00:00:00Z",
    "CollectedInformation": {
        "buildarch": "i386",
        "chip": "ConnectX-3",
        "ip": "192.168.0.2",
        "mac": "12:34:56:78:90:ab",
        "platform": "pcbios",
        "product": "",
        "serial": "abcdefg",
        "uuid": "abcd-efgh-ijkl",
        "version": "3.4.1234"
    }
}`
	lastCreated, err := time.Parse("Jan 2, 2006 at 3:04pm (GMT)", "Jan 2, 2016 at 3:04pm (GMT)")
	if err != nil {
		t.Fatal(err)
	}
	h := Host{
		Name:     "mlab1.iad1t.measurement-lab.org",
		IPv4Addr: "165.117.240.9",
		Boot: datastorex.Map{
			"stage1.ipxe": "https://storage.googleapis.com/epoxy-boot-server/coreos/stage1to2.ipxe",
			"stage2":      "https://storage.googleapis.com/epoxy-boot-server/coreos/stage2to3.json",
			"stage3":      "https://storage.googleapis.com/epoxy-boot-server/coreos/stage3setup.json",
		},
		Update: datastorex.Map{
			"stage1.ipxe": "https://storage.googleapis.com/epoxy-boot-server/centos6/install.json",
			"stage2":      "https://storage.googleapis.com/epoxy-boot-server/centos6/boot.json",
			"stage3":      "",
		},
		CurrentSessionIDs: SessionIDs{
			Stage2ID: "01234",
			Stage3ID: "56789",
			ReportID: "13579",
		},
		LastSessionCreation: lastCreated,
		CollectedInformation: map[string]string{
			"platform":  "pcbios",
			"buildarch": "i386",
			"serial":    "abcdefg",
			"uuid":      "abcd-efgh-ijkl",
			"product":   "",
			"chip":      "ConnectX-3",
			"mac":       "12:34:56:78:90:ab",
			"ip":        "192.168.0.2",
			"version":   "3.4.1234",
		},
	}
	s := h.String()

	if s != hostExpected {
		log.Fatalf("Host record does not match: got '%s'; want '%s'\n", s, hostExpected)
	}
}

// TestHostGenerateSessionIDs uses a synthetic randRead to generate known IDs and
// verifies that a host CurrentSessionIDs contains these IDs.
func TestHostGenerateSessionIDs(t *testing.T) {
	// Assign a synthetic randRead function to generate known session IDs.
	randRead = func(b []byte) (n int, err error) {
		for i := 0; i < len(b); i++ {
			b[i] = 1
		}
		return len(b), nil
	}
	lastCreated, _ := time.Parse("Jan 2, 2006 at 3:04pm (GMT)", "Jan 2, 2016 at 3:04pm (GMT)")
	// Assign a synthetic time function to return a known time.
	timeNow = func() time.Time {
		return lastCreated
	}
	h := &Host{}

	expectedID := "AQEBAQEBAQEBAQEBAQEBAQEBAQE"
	h.GenerateSessionIDs()
	if h.CurrentSessionIDs.Stage2ID != expectedID {
		t.Fatalf("Failed to generate Stage2ID: got %q; want %q",
			h.CurrentSessionIDs.Stage2ID, expectedID)
	}
	if h.CurrentSessionIDs.Stage3ID != expectedID {
		t.Fatalf("Failed to generate Stage3ID: got %q; want %q",
			h.CurrentSessionIDs.Stage3ID, expectedID)
	}
	if h.CurrentSessionIDs.ReportID != expectedID {
		t.Fatalf("Failed to generate ReportID: got %q; want %q",
			h.CurrentSessionIDs.ReportID, expectedID)
	}
	expectedTime := "2016-01-02 15:04:00 +0000 UTC"
	if h.LastSessionCreation.String() != expectedTime {
		t.Fatalf("Failed to update LastSessionCreation: got %q; want %q",
			h.LastSessionCreation.String(), expectedTime)
	}
}
