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
	"context"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/datastore"

	"github.com/m-lab/epoxy/storage"
	"github.com/m-lab/go/rtx"
	"github.com/m-lab/go/siteinfo"

	"github.com/spf13/cobra"
)

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Syncs ePoxy Host records in Datastore with siteinfo",
	Long: `
USAGE:

    Syncs all active hosts in siteinfo with the Datastore records for a
    given project. The outcome should be that there are no sites in siteinfo for
    which Datastore records do not exist. NOTE: sync does not remove Datastore
    records for retired sites, but merely adds missing ones.

EXAMPLE:

    epoxy_admin sync --project mlab-sandbox
`,
	Run: runSync,
}

func runSync(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	siteinfo := siteinfo.New(fProject, "v2", &http.Client{})
	machines, err := siteinfo.Machines()
	rtx.Must(err, "Failed to get siteinfo.Machines()")

	// Setup Datastore client.
	client, err := datastore.NewClient(ctx, fProject)
	rtx.Must(err, "Failed to create new datastore client")

	// Get all Datastore entities for the given project.
	ds := storage.NewDatastoreConfig(client)
	entities, err := ds.List()
	rtx.Must(err, "Failed to get Datastore entities")

	for _, machine := range machines {
		if machine.Project != fProject {
			continue
		}
		if isHostnameInDatastore(machine.Hostname, entities) {
			continue
		}
		cfHostname = machine.Hostname
		cfAddress = machine.IPv4

		fmt.Printf("Adding host to Datastore: %s\n", machine.Hostname)
		runCreate(cmd, args)
	}
}

// isHostnameInDatastore looks for a given hostname in a slice of storage.Hosts
// and returns true if it is found, else false.
func isHostnameInDatastore(hostname string, entities []*storage.Host) bool {
	for _, entity := range entities {
		if hostname == entity.Name {
			return true
		}
	}
	return false
}

func init() {
	rootCmd.AddCommand(syncCmd)

	syncCmd.Flags().StringVar(&sfSiteinfo, "siteinfo",
		"https://siteinfo."+fProject+".measurementlab.net/v2/sites/projects.json",
		"Absolute URL to siteinfo /v2/projects.json file.")
}
