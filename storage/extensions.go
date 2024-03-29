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
	"fmt"
	"os"
)

// ExtentionOperation maps an operation name (used in URLs) to an extension service URL.
type ExtentionOperation struct {
	// Name is the operation name. This will appear in URLs to the ePoxy server.
	// Name should only use characters [a-zA-Z0-9].
	Name string

	// URL references a service that implements the extension operation. During
	// machine boot, when a machine requests the ePoxy extension URL, the ePoxy
	// server will, in turn, issue a request to this URL, sending an
	// extension.Request message. The extension service response is
	// returned to the client verbatim.
	URL string
}

var (
	// Extensions is a static map of operation names to extension URLS for testing.
	// TODO: save/retrieve extension configuration in/from datastore.
	Extensions = map[string]string{
		"allocate_k8s_token":    "http://epoxy-extension-server.%s.measurementlab.net:8800/v2/allocate_k8s_token",
		"bmc_store_password":    "http://epoxy-extension-server.%s.measurementlab.net:8800/v1/bmc_store_password",
		"test_op":               "http://soltesz-epoxy-testing-instance-1.c.%s.internal:8001/operation",
	}
)

func init() {
	// TODO: Remove this logic once the allocate_k8s_token URL is stored/read from datastore.
	projectID := os.Getenv("GCLOUD_PROJECT")
	if projectID != "" {
		for key, value := range Extensions {
			Extensions[key] = fmt.Sprintf(value, projectID)
		}
	}
}
