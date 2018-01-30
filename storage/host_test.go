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
)

func TestHostString(t *testing.T) {
	hostExpected := `{
    "Name": "mlab1.iad1t.measurement-lab.org",
    "IPAddress": "165.117.240.9",
    "Stage1to2ScriptName": "https://storage.googleapis.com/epoxy-boot-server/stage1to2/stage1to2.ipxe",
    "NextStageEnabled": false,
    "NextStageScriptName": "https://storage.googleapis.com/epoxy-boot-server/centos6/install.json",
    "DefaultScriptName": "https://storage.googleapis.com/epoxy-boot-server/centos6/boot.json",
    "CurrentSessionIDs": {
        "NextStageID": "01234",
        "BeginStageID": "56789",
        "EndStageID": "13579"
    },
    "LastSessionCreation": "2016-01-02T15:04:00Z",
    "CollectedInformation": {
        "Platform": "",
        "BuildArch": "",
        "Serial": "",
        "Asset": "",
        "UUID": "",
        "Manufacturer": "",
        "Product": "",
        "Chip": "",
        "MAC": "",
        "IP": "",
        "Version": "",
        "PublicSSHHostKey": ""
    }
}`

	lastCreated, err := time.Parse("Jan 2, 2006 at 3:04pm (GMT)", "Jan 2, 2016 at 3:04pm (GMT)")
	if err != nil {
		t.Fatal(err)
	}
	h := Host{
		Name:                "mlab1.iad1t.measurement-lab.org",
		IPAddress:           "165.117.240.9",
		Stage1to2ScriptName: "https://storage.googleapis.com/epoxy-boot-server/stage1to2/stage1to2.ipxe",
		NextStageScriptName: "https://storage.googleapis.com/epoxy-boot-server/centos6/install.json",
		DefaultScriptName:   "https://storage.googleapis.com/epoxy-boot-server/centos6/boot.json",
		CurrentSessionIDs: SessionIDs{
			NextStageID:  "01234",
			BeginStageID: "56789",
			EndStageID:   "13579",
		},
		LastSessionCreation: lastCreated,
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
	lastCreated, err := time.Parse("Jan 2, 2006 at 3:04pm (GMT)", "Jan 2, 2016 at 3:04pm (GMT)")
	// Assign a synthetic time function to return a known time.
	timeNow = func() time.Time {
		return lastCreated
	}
	h := &Host{}

	expectedID := "AQEBAQEBAQEBAQEBAQEBAQEBAQE"
	err = h.GenerateSessionIDs()
	if err != nil {
		t.Fatalf("Failed to generate session IDs: %s", err)
	}
	if h.CurrentSessionIDs.NextStageID != expectedID {
		t.Fatalf("Failed to generate NextStageID: got %q; want %q",
			h.CurrentSessionIDs.NextStageID, expectedID)
	}
	if h.CurrentSessionIDs.BeginStageID != expectedID {
		t.Fatalf("Failed to generate BeginStageID: got %q; want %q",
			h.CurrentSessionIDs.BeginStageID, expectedID)
	}
	if h.CurrentSessionIDs.EndStageID != expectedID {
		t.Fatalf("Failed to generate EndStageID: got %q; want %q",
			h.CurrentSessionIDs.EndStageID, expectedID)
	}
	expectedTime := "2016-01-02 15:04:00 +0000 UTC"
	if h.LastSessionCreation.String() != expectedTime {
		t.Fatalf("Failed to update LastSessionCreation: got %q; want %q",
			h.LastSessionCreation.String(), expectedTime)
	}
}
