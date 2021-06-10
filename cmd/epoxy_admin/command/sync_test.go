// Copyright 2021 ePoxy Authors
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

package command

import (
	"testing"

	"github.com/m-lab/epoxy/storage"
)

func TestSync_isHostnameinDatastore(t *testing.T) {
	entities := []*storage.Host{
		{
			Name: "mlab1-xyz0t.mlab-sandbox.measurement-lab.org",
		},
		{
			Name: "mlab3-lol05.mlab-staging.measurement-lab.org",
		},
		{
			Name: "mlab2-abc01.mlab-sandbox.measurement-lab.org",
		},
		{
			Name: "mlab4-usa02.mlab-staging.measurement-lab.org",
		},
		{
			Name: "mlab1-xyz01.mlab-sandbox.measurement-lab.org",
		},
	}

	tests := []struct {
		name     string
		hostname string
		found    bool
	}{
		{
			name:     "found-hostname",
			hostname: "mlab4-usa02.mlab-staging.measurement-lab.org",
			found:    true,
		},
		{
			name:     "not-found-hostname",
			hostname: "mlab9-ddd01.mlab-staging.measurement-lab.org",
			found:    false,
		},
		{
			name:     "no-hostname",
			hostname: "",
			found:    false,
		},
	}

	for _, tt := range tests {
		found := isHostnameInDatastore(tt.hostname, entities)
		if found != tt.found {
			t.Errorf("isHostnameInDatastore(): wanted %v, got %v", tt.found, found)
		}
	}
}
