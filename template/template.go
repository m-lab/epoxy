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
	"log"

	"github.com/m-lab/epoxy/storage"
)

// stage1IpxeTemplate is a template for executing the stage1 iPXE script.
const stage1IpxeTemplate = `#!ipxe

set stage1to2_url {{ .Stage1to2ScriptURL }}
set stage2_url {{ .Stage2URL }}
set stage3_url {{ .Stage3URL }}
set report_url {{ .ReportURL }}

chain ${stage1to2_url}
`

// FormatStage1IPXEScript generates a stage1 iPXE boot script using values from Host.
func FormatStage1IPXEScript(h *storage.Host, serverAddr string) string {
	var b bytes.Buffer

	// Prepare a map for evaluating template.
	vals := make(map[string]string)
	vals["Stage1to2ScriptURL"] = h.Stage1to2ScriptName
	vals["Stage2URL"] = fmt.Sprintf("https://%s/v1/boot/%s/%s/stage2.json",
		serverAddr, h.Name, h.CurrentSessionIDs.NextStageID)
	vals["Stage3URL"] = fmt.Sprintf("https://%s/v1/boot/%s/%s/stage3.json",
		serverAddr, h.Name, h.CurrentSessionIDs.BeginStageID)
	vals["ReportURL"] = fmt.Sprintf("https://%s/v1/boot/%s/%s/report",
		serverAddr, h.Name, h.CurrentSessionIDs.EndStageID)

	t := template.Must(template.New("stage1").Parse(stage1IpxeTemplate))
	err := t.Execute(&b, vals)
	if err != nil {
		// This error could only occur with a bad template, which should
		// be caught by unit tests.
		log.Print(err)
		// Use panic instead of log.Fatal so the server can recover.
		panic(err)
		// TODO: return a static fallback configuration via the stage1to2_url.
	}

	return b.String()
}
