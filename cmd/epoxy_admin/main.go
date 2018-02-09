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
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/datastore"
	"github.com/kr/pretty"
	"github.com/m-lab/epoxy/storage"
	"golang.org/x/net/context"
)

const usage = `USAGE:
**Only use for testing.**

EXAMPLE:
    epoxy_admin --project mlab-sandbox \
        --hostname mlab3.iad1t.measurement-lab.org \
        --address 165.117.240.35 \
        --stage1 https://storage.googleapis.com/epoxy-sandbox/stage1/stage1.ipxe
        --stage2 https://storage.googleapis.com/epoxy-sandbox/stage2/stage2.ipxe
        --stage3 https://storage.googleapis.com/epoxy-sandbox/stage3/stage3.ipxe
`

var (
	fProject  string
	fHostname string
	fAddress  string
	fStage1   string
	fStage2   string
	fStage3   string
)

func init() {
	// Add an alternate usage message.
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, usage)
		flag.PrintDefaults()
	}
	flag.StringVar(&fProject, "project", "mlab-sandbox", "GCP project ID.")
	flag.StringVar(&fHostname, "hostname", "mlab3.iad1t.measurement-lab.org", "Hostname of new record.")
	flag.StringVar(&fAddress, "address", "165.117.240.35", "IP address of hostname.")
	flag.StringVar(&fStage1, "stage1",
		"https://storage.googleapis.com/epoxy-sandbox/stage1/stage1.ipxe",
		"Absolute URL to a stage1.ipxe script.")
	flag.StringVar(&fStage2, "stage2",
		"https://storage.googleapis.com/epoxy-sandbox/stage2/stage2.json",
		"Absolute URL to a stage2.json config.")
	flag.StringVar(&fStage3, "stage3",
		"https://storage.googleapis.com/epoxy-sandbox/stage3/stage3.json",
		"Absolute URL to a stage2.json config.")
}

func main() {
	flag.Parse()

	// Setup Datastore client.
	ctx := context.Background()
	client, err := datastore.NewClient(ctx, fProject)
	if err != nil {
		log.Fatalf("Failed to create new datastore client: %s", err)
	}

	// Save the host record to Datstore.
	ds := storage.NewDatastoreConfig(client)
	h := &storage.Host{
		Name:     fHostname,
		IPv4Addr: fAddress,
		Boot: storage.Sequence{
			Stage1ChainURL: fStage1,
			Stage2ChainURL: fStage2,
			Stage3ChainURL: fStage3,
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
