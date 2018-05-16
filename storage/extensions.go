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

// ExtentionOperation maps an operation name (used in URLs) to an extension service URL.
type ExtentionOperation struct {
	// Name is the operation name. This will appear in URLs to the ePoxy server.
	// Name should only use characters [a-zA-Z0-9].
	Name string

	// URL references a service that implements the extension operation. During
	// machine boot, when a machine requests the ePoxy extension URL, the ePoxy
	// server will, in turn, issue a request to this URL, sending an
	// extension.WebhookRequest message. The extension service response is
	// returned to the client verbatim.
	URL string
}

var (
	// TODO: save/retrieve extension configuration in/from datastore.
	// This static map is for preliminary testing.
	Extensions map[string]string = map[string]string{
		"k8stoken": "http://k8s-platform-master.mlab-sandbox.measurementlab.net:8000",
	}
)