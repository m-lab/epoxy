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

// A minimal client for adding Host records to Datastore for testing. This
// command is ONLY for testing. Host record management by direct access to
// Datastore will not be supported by ePoxy.
package main

import (
	"flag"
	"log"

	"cloud.google.com/go/datastore"
	"github.com/kylelemons/godebug/pretty"
	"github.com/m-lab/epoxy/storage"
	"golang.org/x/net/context"
)

var (
	project  = flag.String("project", "mlab-sandbox", "GCP project ID.")
	hostname = flag.String("hostname", "mlab3.iad1t.measurement-lab.org", "Hostname of new record.")
	address  = flag.String("address", "165.117.240.35", "IP address of hostname.")
	stage1   = flag.String("stage1",
		"https://storage.googleapis.com/epoxy-sandbox/stage1/stage1.ipxe",
		"Absolute URL to a stage1.ipxe script.")
	stage2 = flag.String("stage2",
		"https://storage.googleapis.com/epoxy-sandbox/stage2/stage2.json",
		"Absolute URL to a stage2.json config.")
	stage3 = flag.String("stage3",
		"https://storage.googleapis.com/epoxy-sandbox/stage3/stage3.json",
		"Absolute URL to a stage2.json config.")
)

const usage = `USAGE:
**Only use for testing.**
EXAMPLE:
    create_sample_data --project mlab-sandbox \
        --hostname mlab3.iad1t.measurement-lab.org \
        --address 165.117.240.35 \
        --stage1 https://storage.googleapis.com/epoxy-sandbox/stage1/stage1.ipxe
`

func main() {
	flag.Parse()

	// Print usage unconditionally.
	log.Println(usage)

	// Setup Datastore client.
	ctx := context.Background()
	client, err := datastore.NewClient(ctx, *project)
	if err != nil {
		log.Fatalf("Failed to create new datastore client: %s", err)
	}

	// Save the host record to Datstore.
	ds := storage.NewDatastoreConfig(client)
	h := &storage.Host{
		Name:     *hostname,
		IPv4Addr: *address,
		Boot: storage.Sequence{
			Stage1ChainURL: *stage1,
			Stage2ChainURL: *stage2,
			Stage3ChainURL: *stage3,
		},
	}
	if err = ds.Save(h); err != nil {
		log.Fatalf("%s", err)
	}

	// Retrieve the host record from Datastore to exercise the full save & load path.
	h2, err := ds.Load(h.Name)
	if err != nil {
		log.Fatalf("%s", err)
	}
	pretty.Print(h2)
}
