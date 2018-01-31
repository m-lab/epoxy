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

package template

import (
	"strings"
	"testing"

	"github.com/kylelemons/godebug/pretty"
	"github.com/m-lab/epoxy/storage"
)

const expectedStage1Script = `#!ipxe

set stage1to2_url https://example.com/path/stage1to2/stage1to2.ipxe
set nextstage_url https://boot-api-mlab-sandbox.appspot.com/v1/boot/mlab1.iad1t.measurement-lab.org/01234/nextstage.json
set beginstage_url https://boot-api-mlab-sandbox.appspot.com/v1/boot/mlab1.iad1t.measurement-lab.org/56789/beginstage
set endstage_url https://boot-api-mlab-sandbox.appspot.com/v1/boot/mlab1.iad1t.measurement-lab.org/86420/endstage

chain ${stage1to2_url}
`

// TestFormatStage1IPXEScript formats a stage1 iPXE script for a sample Host record.
// The result is checked for a valid header and verbatim against the expected content.
func TestFormatStage1IPXEScript(t *testing.T) {
	h := &storage.Host{
		Name:                "mlab1.iad1t.measurement-lab.org",
		IPAddress:           "165.117.240.9",
		Stage1to2ScriptName: "https://example.com/path/stage1to2/stage1to2.ipxe",
		CurrentSessionIDs: storage.SessionIDs{
			NextStageID:  "01234",
			BeginStageID: "56789",
			EndStageID:   "86420",
		},
	}

	script, err := FormatStage1IPXEScript(h, "boot-api-mlab-sandbox.appspot.com")
	if err != nil {
		t.Fatalf("Failed to create stage1 ipxe script: %s", err)
	}
	// Verify the correct script header.
	if !strings.HasPrefix(script, "#!ipxe") {
		lines := strings.SplitN(script, "\n", 2)
		t.Errorf("Wrong iPXE script prefix: got %q want '#!ipxe'", lines[0])
	}
	expectedLines := strings.Split(expectedStage1Script, "\n")
	actualLines := strings.Split(script, "\n")
	if diff := pretty.Compare(expectedLines, actualLines); diff != "" {
		t.Errorf("Wrong iPXE script: diff (-want +got):\n%s", diff)
	}
}
