// Copyright 2018 ePoxy Authors
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
	"time"

	"cloud.google.com/go/datastore"
	"github.com/m-lab/epoxy/storage"
	"github.com/m-lab/go/rtx"
	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Updates an existing ePoxy Host record in Datastore",
	Long: `
USAGE:
	**ONLY FOR TESTING**

    Updates an existing Host record with the given values. Updating a
    non-existant host is a failure.

EXAMPLE:

    # Set the "update enabled" flag on the Host record.
    epoxy_admin update --project mlab-sandbox \
		--hostname mlab3.iad1t.measurement-lab.org \
		--update
`,
	Run: runUpdate,
}

func updateURL(url, original string) string {
	if url != "" {
		return fmtURL(url)
	}
	return original
}

// TODO: add unit tests by masking out NewClient & NewDatstoreConfig. Consider
// promoting the fake datastore types from storage/datastore_test.go to an
// internal fake package.
func runUpdate(cmd *cobra.Command, args []string) {
	// Setup Datastore client.
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	client, err := datastore.NewClient(ctx, fProject)
	rtx.Must(err, "Failed to create new datastore client")

	// Save the host record to Datstore.
	ds := storage.NewDatastoreConfig(client)

	h, err := ds.Load(fHostname)
	rtx.Must(err, "Failed to load record for %q", fHostname)
	h.UpdateEnabled = fUpdate
	// TODO: support multiple extensions.
	if fExtension != "" {
		h.Extensions = []string{fExtension}
	}
	if fAddress != "" {
		h.IPv4Addr = fAddress
	}
	h.Boot.Stage1ChainURL = updateURL(fmtURL(fBootStage1), h.Boot.Stage1ChainURL)
	h.Boot.Stage2ChainURL = updateURL(fmtURL(fBootStage2), h.Boot.Stage2ChainURL)
	h.Boot.Stage3ChainURL = updateURL(fmtURL(fBootStage3), h.Boot.Stage3ChainURL)
	h.Update.Stage1ChainURL = updateURL(fmtURL(fUpdateStage1), h.Update.Stage1ChainURL)
	h.Update.Stage2ChainURL = updateURL(fmtURL(fUpdateStage2), h.Update.Stage2ChainURL)
	h.Update.Stage3ChainURL = updateURL(fmtURL(fUpdateStage3), h.Update.Stage3ChainURL)

	// Save the host record.
	err = ds.Save(h)
	rtx.Must(err, "Failed to save new host record")

	// Retrieve the host record from Datastore to exercise the full save & load path.
	h, err = ds.Load(h.Name)
	rtx.Must(err, "Failed to save new host record")
	fmt.Println(h.String())
	return
}

func init() {
	rootCmd.AddCommand(updateCmd)

	// Required local flags.
	updateCmd.Flags().StringVar(&fHostname, "hostname", "",
		"Hostname of new record.")
	updateCmd.MarkFlagRequired("hostname")

	// Local flags which will only run when "update" is called directly.
	updateCmd.Flags().StringVar(&fAddress, "address", "",
		"IP address of hostname.")
	updateCmd.Flags().StringVar(&fExtension, "extension", "",
		"Name of an extension to enable for host.")
	updateCmd.Flags().BoolVar(&fUpdate, "update", false,
		"Set Host.UpdateEnabled to true for an existing Host.")
	updateCmd.Flags().StringVar(&fBootStage1, "boot-stage1", "",
		"Absolute URL to an action definition to run during stage1 to stage2 boot.")
	updateCmd.Flags().StringVar(&fBootStage2, "boot-stage2", "",
		"Absolute URL to an action definition to run during stage2 to stage3 boot.")
	updateCmd.Flags().StringVar(&fBootStage3, "boot-stage3", "",
		"Absolute URL to an action definition to run after running stage3 boot.")
	updateCmd.Flags().StringVar(&fUpdateStage1, "update-stage1", "",
		"Absolute URL to an action definition to run during stage1 to stage2 update.")
	updateCmd.Flags().StringVar(&fUpdateStage2, "update-stage2", "",
		"Absolute URL to an action definition to run during stage2 to stage3 update.")
	updateCmd.Flags().StringVar(&fUpdateStage3, "update-stage3", "",
		"Absolute URL to an action definition to run after running stage3 update.")
}
