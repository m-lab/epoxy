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
	"log"
	"regexp"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/m-lab/epoxy/storage"
	"github.com/m-lab/go/rtx"
	"github.com/spf13/cobra"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Updates all ePoxy Host records matching --hostname pattern",
	Long: `
USAGE:
    **ONLY FOR TESTING**

    Updates Host records matching the regex pattern in the --hostname flag.

EXAMPLE:

    # Set the "update enabled" flag on the Host record.
    epoxy_admin update --project mlab-sandbox \
        --hostname mlab3.iad1t.measurement-lab.org \
        --update

    # Set the "update enabled" flag on all mlab4 Host records.
    epoxy_admin update --project mlab-sandbox \
        --hostname 'mlab4.*' \
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
	hosts, err := ds.List()
	rtx.Must(err, "Failed to list host records")

	// Compile given regex.
	r, err := regexp.Compile(ufHostname)
	rtx.Must(err, "Failed to compile given hostname pattern: %q", ufHostname)

	for _, h := range hosts {
		if !r.MatchString(h.Name) {
			continue
		}
		log.Printf("Updating: %s", h.Name)

		handleUpdate(h)

		// Save the host record.
		err = ds.Save(h)
		rtx.Must(err, "Failed to save new host record")

		// Retrieve the host record from Datastore to exercise the full save & load path.
		h, err = ds.Load(h.Name)
		rtx.Must(err, "Failed to save new host record")
		fmt.Println(h.String())
	}
	return
}

func handleUpdate(h *storage.Host) {
	h.UpdateEnabled = ufUpdate
	// TODO: support multiple extensions.
	if ufExtension != "" {
		h.Extensions = []string{ufExtension}
	}
	if ufAddress != "" {
		h.IPv4Addr = ufAddress
	}
	h.Boot.Stage1ChainURL = updateURL(fmtURL(ufBootStage1), h.Boot.Stage1ChainURL)
	h.Boot.Stage2ChainURL = updateURL(fmtURL(ufBootStage2), h.Boot.Stage2ChainURL)
	h.Boot.Stage3ChainURL = updateURL(fmtURL(ufBootStage3), h.Boot.Stage3ChainURL)
	h.Update.Stage1ChainURL = updateURL(fmtURL(ufUpdateStage1), h.Update.Stage1ChainURL)
	h.Update.Stage2ChainURL = updateURL(fmtURL(ufUpdateStage2), h.Update.Stage2ChainURL)
	h.Update.Stage3ChainURL = updateURL(fmtURL(ufUpdateStage3), h.Update.Stage3ChainURL)
}

func init() {
	rootCmd.AddCommand(updateCmd)

	// Required local flags.
	updateCmd.Flags().StringVar(&ufHostname, "hostname", "",
		"Hostname of new record.")
	updateCmd.MarkFlagRequired("hostname")

	// Local flags which will only run when "update" is called directly.
	updateCmd.Flags().StringVar(&ufAddress, "address", "",
		"IP address of hostname.")
	updateCmd.Flags().StringVar(&ufExtension, "extension", "",
		"Name of an extension to enable for host.")
	updateCmd.Flags().BoolVar(&ufUpdate, "update", false,
		"Set Host.UpdateEnabled to true for an existing Host.")
	updateCmd.Flags().StringVar(&ufBootStage1, "boot-stage1", "",
		"Absolute URL to an action definition to run during stage1 to stage2 boot.")
	updateCmd.Flags().StringVar(&ufBootStage2, "boot-stage2", "",
		"Absolute URL to an action definition to run during stage2 to stage3 boot.")
	updateCmd.Flags().StringVar(&ufBootStage3, "boot-stage3", "",
		"Absolute URL to an action definition to run after running stage3 boot.")
	updateCmd.Flags().StringVar(&ufUpdateStage1, "update-stage1", "",
		"Absolute URL to an action definition to run during stage1 to stage2 update.")
	updateCmd.Flags().StringVar(&ufUpdateStage2, "update-stage2", "",
		"Absolute URL to an action definition to run during stage2 to stage3 update.")
	updateCmd.Flags().StringVar(&ufUpdateStage3, "update-stage3", "",
		"Absolute URL to an action definition to run after running stage3 update.")
}
