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

	"github.com/m-lab/epoxy/nextboot"
	"github.com/m-lab/epoxy/storage"
)

// stage1IpxeTemplate is a template for executing the stage1 iPXE script.
const stage1IpxeTemplate = `#!ipxe

set stage1chain_url {{ .Stage1ChainURL }}
set stage2_url {{ .Stage2URL }}
set stage3_url {{ .Stage3URL }}
set report_url {{ .ReportURL }}
{{- range $key, $value := .Extensions }}
set {{ $key }}_url {{ $value }}
{{- end }}

chain ${stage1chain_url}
`

// FormatStage1IPXEScript generates a stage1 iPXE boot script using values from Host.
func FormatStage1IPXEScript(h *storage.Host, serverAddr string) string {
	var b bytes.Buffer

	// Chose the current boot sequence from host.
	s := h.CurrentSequence()

	// Prepare a map for evaluating template.
	vals := make(map[string]interface{}, 5)
	vals["Stage1ChainURL"] = s.NextURL("stage1")
	vals["Stage2URL"] = fmt.Sprintf("https://%s/v1/boot/%s/%s/stage2",
		serverAddr, h.Name, h.CurrentSessionIDs.Stage2ID)
	vals["Stage3URL"] = fmt.Sprintf("https://%s/v1/boot/%s/%s/stage3",
		serverAddr, h.Name, h.CurrentSessionIDs.Stage3ID)
	vals["ReportURL"] = fmt.Sprintf("https://%s/v1/boot/%s/%s/report",
		serverAddr, h.Name, h.CurrentSessionIDs.ReportID)

	// Construct an extension URL for all extensions this host supports.
	extensionURLs := make(map[string]string, len(h.Extensions))
	// TODO: verify that extensions actually exist. e.g. do not generate invalid urls.
	for _, operation := range h.Extensions {
		extensionURLs[operation] = fmt.Sprintf("https://%s/v1/boot/%s/%s/extension/%s",
			serverAddr, h.Name, h.CurrentSessionIDs.ExtensionID, operation)
	}
	vals["Extensions"] = extensionURLs

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

// CreateStage1Action generates a stage1 epoxy-client action using values from Host.
func CreateStage1Action(h *storage.Host, serverAddr string) string {
	// Chose the current boot sequence from host.
	s := h.CurrentSequence()

	c := nextboot.Config{
		// clients receiving this configuration must support merging local and given Kargs.
		Kargs: map[string]string{
			"epoxy.stage2": fmt.Sprintf("https://%s/v1/boot/%s/%s/stage2", serverAddr, h.Name, h.CurrentSessionIDs.Stage2ID),
			"epoxy.stage3": fmt.Sprintf("https://%s/v1/boot/%s/%s/stage3", serverAddr, h.Name, h.CurrentSessionIDs.Stage3ID),
			"epoxy.report": fmt.Sprintf("https://%s/v1/boot/%s/%s/report", serverAddr, h.Name, h.CurrentSessionIDs.ReportID),
		},
		V1: &nextboot.V1{
			Chain: s.NextURL("stage1"),
		},
	}

	// Construct an extension URL for all extensions this host supports.
	// TODO: verify that extensions actually exist. e.g. do not generate invalid urls.
	for _, operation := range h.Extensions {
		c.Kargs["epoxy."+operation] = fmt.Sprintf("https://%s/v1/boot/%s/%s/extension/%s", serverAddr, h.Name, h.CurrentSessionIDs.ExtensionID, operation)
	}

	return c.String()
}

// FormatStage2JSONConfig generates a stage2 JSON configuration for an epoxy client.
func FormatJSONConfig(h *storage.Host, stage string) string {
	// Chose the current boot sequence from host.
	s := h.CurrentSequence()
	c := nextboot.Config{
		V1: &nextboot.V1{
			Chain: s.NextURL(stage),
		},
	}
	return c.String()
}
