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
//
// TODO:
//   * Create distinct subcommands, e.g. create, update, delete.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/datastore"
	"github.com/kr/pretty"
	"github.com/m-lab/epoxy/storage"
)

const usage = `USAGE:
**ONLY USE FOR TESTING**

EXAMPLE:
    # Use the default boot and update stage URLs:
    epoxy_admin --project mlab-sandbox \
        --hostname mlab3.iad1t.measurement-lab.org \
        --address 165.117.240.35
`

var (
	fProject      string
	fHostname     string
	fAddress      string
	fExtension    string
	fUpdate       bool
	fBootStage1   string
	fBootStage2   string
	fBootStage3   string
	fUpdateStage1 string
	fUpdateStage2 string
	fUpdateStage3 string
)

func init() {
	// Add an alternate usage message.
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, usage)
		flag.PrintDefaults()
	}
	flag.StringVar(&fProject, "project", "mlab-sandbox", "GCP project ID.")
	flag.StringVar(&fHostname, "hostname", "", "Hostname of new record.")
	flag.StringVar(&fAddress, "address", "", "IP address of hostname.")
	flag.StringVar(&fExtension, "extension", "allocate_k8s_token", "Name of an extension to enable for host.")
	flag.BoolVar(&fUpdate, "update", false,
		"Set Host.UpdateEnabled to true for an existing Host. Do not specify when creating a new Host.")
	flag.StringVar(&fBootStage1, "boot-stage1",
		"https://storage.googleapis.com/epoxy-%s/stage3_coreos/stage1to2.ipxe",
		"Absolute URL to an action definition to run during stage1 to boot stage2.")
	flag.StringVar(&fBootStage2, "boot-stage2",
		"https://storage.googleapis.com/epoxy-%s/stage3_coreos/stage2to3.json",
		"Absolute URL to an action definition to run during stage2 to boot stage3.")
	flag.StringVar(&fBootStage3, "boot-stage3",
		"https://storage.googleapis.com/epoxy-%s/stage3_coreos/stage3post.json",
		"Absolute URL to an action definition to run after booting stage3.")
	flag.StringVar(&fUpdateStage1, "update-stage1",
		"https://storage.googleapis.com/epoxy-%s/stage3_mlxupdate/stage1to2.ipxe",
		"Absolute URL to an action definition to run during stage1 to boot stage2.")
	flag.StringVar(&fUpdateStage2, "update-stage2",
		"https://storage.googleapis.com/epoxy-%s/stage3_mlxupdate/stage2to3.json",
		"Absolute URL to an action definition to run during stage2 to boot stage3.")
	flag.StringVar(&fUpdateStage3, "update-stage3",
		"https://storage.googleapis.com/epoxy-%s/stage3_mlxupdate/stage3post.json",
		"Absolute URL to an action definition to run after booting stage3.")
}

func main() {
	flag.Parse()

	var err error
	// Setup Datastore client.
	ctx := context.Background()
	client, err := datastore.NewClient(ctx, fProject)
	if err != nil {
		log.Fatalf("Failed to create new datastore client: %s", err)
	}

	// Save the host record to Datstore.
	ds := storage.NewDatastoreConfig(client)
	var h *storage.Host

	if fUpdate {
		// Retrieve the host record from Datastore before updating it.
		h, err = ds.Load(fHostname)
		if err != nil {
			log.Fatalf("%s", err)
		}
		h.UpdateEnabled = true

	} else {

		// Create a new host record.
		h = &storage.Host{
			Name:          fHostname,
			IPv4Addr:      fAddress,
			UpdateEnabled: false,
			Extensions:    []string{fExtension},
			Boot: storage.Sequence{
				Stage1ChainURL: fmt.Sprintf(fBootStage1, fProject),
				Stage2ChainURL: fmt.Sprintf(fBootStage2, fProject),
				Stage3ChainURL: fmt.Sprintf(fBootStage3, fProject),
			},
			Update: storage.Sequence{
				Stage1ChainURL: fmt.Sprintf(fUpdateStage1, fProject),
				Stage2ChainURL: fmt.Sprintf(fUpdateStage2, fProject),
				Stage3ChainURL: fmt.Sprintf(fUpdateStage3, fProject),
			},
		}
	}
	if err = ds.Save(h); err != nil {
		log.Fatalf("%s", err)
	}

	// Retrieve the host record from Datastore to exercise the full save & load path.
	h2, err := ds.Load(h.Name)
	if err != nil {
		log.Fatalf("%s", err)
	}
	pretty.Print(h2.String())
}
