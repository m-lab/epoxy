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

// Package template provides tools for formatting iPXE scripts and JSON configs
// for ePoxy clients.
package template

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/m-lab/epoxy/storage"
)

// stage2IpxeTemplate is a template for executing the stage2 iPXE script.
const stage2IpxeTemplate = `#!ipxe

set stage2_url {{ .Stage2ScriptName }}
set nextstage_url {{ .NextStageURL }}
set beginstage_url {{ .BeginStageURL }}
set endstage_url {{ .EndStageURL }}

chain ${stage2_url}
`

// FormatStage2IPXEScript generates a stage2 iPXE boot script using values from Host.
func FormatStage2IPXEScript(h *storage.Host, serverAddr string) (script string, err error) {
	var b bytes.Buffer

	t, err := template.New("stage2").Parse(stage2IpxeTemplate)
	if err != nil {
		return "", err
	}

	// Prepare a map
	vals := make(map[string]string)
	vals["Stage2ScriptName"] = h.Stage2ScriptName
	vals["NextStageURL"] = fmt.Sprintf("https://%s/v1/boot/%s/%s/nextstage.json",
		serverAddr, h.Name, h.CurrentSessionIDs.NextStageID)
	vals["BeginStageURL"] = fmt.Sprintf("https://%s/v1/boot/%s/%s/beginstage",
		serverAddr, h.Name, h.CurrentSessionIDs.BeginStageID)
	vals["EndStageURL"] = fmt.Sprintf("https://%s/v1/boot/%s/%s/endstage",
		serverAddr, h.Name, h.CurrentSessionIDs.EndStageID)

	err = t.Execute(&b, vals)
	if err != nil {
		return "", err
	}

	return b.String(), nil
}
