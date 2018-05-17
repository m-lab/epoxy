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

set stage1chain_url https://example.com/path/stage1to2/stage1to2.ipxe
set stage2_url https://boot-api-mlab-sandbox.appspot.com/v1/boot/mlab1.iad1t.measurement-lab.org/01234/stage2
set stage3_url https://boot-api-mlab-sandbox.appspot.com/v1/boot/mlab1.iad1t.measurement-lab.org/56789/stage3
set report_url https://boot-api-mlab-sandbox.appspot.com/v1/boot/mlab1.iad1t.measurement-lab.org/86420/report
set ext1_url https://boot-api-mlab-sandbox.appspot.com/v1/boot/mlab1.iad1t.measurement-lab.org/75319/extension/ext1
set ext2_url https://boot-api-mlab-sandbox.appspot.com/v1/boot/mlab1.iad1t.measurement-lab.org/75319/extension/ext2

chain ${stage1chain_url}
`

// TestFormatStage1IPXEScript formats a stage1 iPXE script for a sample Host record.
// The result is checked for a valid header and verbatim against the expected content.
func TestFormatStage1IPXEScript(t *testing.T) {
	h := &storage.Host{
		Name:     "mlab1.iad1t.measurement-lab.org",
		IPv4Addr: "165.117.240.9",
		Boot: storage.Sequence{
			Stage1ChainURL: "https://example.com/path/stage1to2/stage1to2.ipxe",
		},
		Extensions: []string{"ext1", "ext2"},
		CurrentSessionIDs: storage.SessionIDs{
			Stage2ID:    "01234",
			Stage3ID:    "56789",
			ReportID:    "86420",
			ExtensionID: "75319",
		},
	}

	script := FormatStage1IPXEScript(h, "boot-api-mlab-sandbox.appspot.com")
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
