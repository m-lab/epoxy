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
	"net/url"
	"strings"
	"time"

	"cloud.google.com/go/datastore"

	"github.com/m-lab/epoxy/datastorex"
	"github.com/m-lab/epoxy/storage"
	"github.com/m-lab/go/rtx"

	"github.com/spf13/cobra"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Adds a new ePoxy Host record to Datastore",
	Long: `
USAGE:
	**ONLY FOR TESTING**

    Creates a new datastore Host record for ePoxy server. Calling "create" on
    an existing host will overwrite the original.

EXAMPLE:

    # Use the default boot and update stage URLs:
    epoxy_admin create --project mlab-sandbox \
        --hostname mlab3.iad1t.measurement-lab.org \
        --address 165.117.240.35
`,
	Run: runCreate,
}

// fmtURL formats (if needed) and parses the given string as a URL. If the
// resulting URL is invalid, fmtURL panics.
func fmtURL(urlStr string) string {
	if strings.Contains(urlStr, "%s") {
		urlStr = fmt.Sprintf(urlStr, fProject)
	}
	_, err := url.Parse(urlStr)
	rtx.Must(err, "Failed to parse URL: %s", urlStr)
	return urlStr
}

// TODO: add unit tests by masking out NewClient & NewDatstoreConfig. Consider
// promoting the fake datastore types from storage/datastore_test.go to an
// internal fake package.
func runCreate(cmd *cobra.Command, args []string) {
	fmt.Println("Project:", fProject)
	// Setup Datastore client.
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	client, err := datastore.NewClient(ctx, fProject)
	rtx.Must(err, "Failed to create new datastore client")

	// Save the host record to Datstore.
	ds := storage.NewDatastoreConfig(client)

	h := &storage.Host{
		Name:          cfHostname,
		IPv4Addr:      cfAddress,
		UpdateEnabled: cfUpdate,
		Extensions:    []string{cfExtension},
		Boot: datastorex.Map{
			storage.Stage1IPXE: fmtURL(cfBootStage1),
			storage.Stage1JSON: fmtURL(cfBootStage1JSON),
			storage.Stage2:     fmtURL(cfBootStage2),
			storage.Stage3:     fmtURL(cfBootStage3),
		},
		Update: datastorex.Map{
			storage.Stage1IPXE: fmtURL(cfUpdateStage1),
			storage.Stage1JSON: fmtURL(cfUpdateStage1JSON),
			storage.Stage2:     fmtURL(cfUpdateStage2),
			storage.Stage3:     fmtURL(cfUpdateStage3),
		},
		CollectedInformation: datastorex.Map{},
	}

	// Save the host record.
	err = ds.Save(h)
	rtx.Must(err, "Failed to save new host record")

	// Retrieve the host record from Datastore to exercise the full save & load path.
	h, err = ds.Load(h.Name)
	rtx.Must(err, "Failed to save new host record")
	fmt.Println(h.String())
}

func init() {
	rootCmd.AddCommand(createCmd)

	// Required local flags.
	createCmd.Flags().StringVar(&cfHostname, "hostname", "",
		"Hostname of new record.")
	createCmd.Flags().StringVar(&cfAddress, "address", "",
		"IP address of hostname.")
	createCmd.MarkFlagRequired("hostname")
	createCmd.MarkFlagRequired("address")

	// Local flags which will only apply when "create" is called directly.
	createCmd.Flags().StringVar(&cfExtension, "extension", "allocate_k8s_token",
		"Name of an extension to enable for host.")
	createCmd.Flags().BoolVar(&cfUpdate, "update", false,
		"Set Host.UpdateEnabled to true for an existing Host.")
	createCmd.Flags().StringVar(&cfBootStage1, "boot-stage1",
		"https://epoxy-boot-api.%s.measurementlab.net:4430/v1/storage/stage3_coreos/stage1to2.ipxe",
		"Absolute URL to an action definition to run during stage1 to stage2 boot.")
	createCmd.Flags().StringVar(&cfBootStage1JSON, "boot-stage1-json",
		"https://storage.googleapis.com/epoxy-%s/stage3_coreos/stage1to2.json",
		"Absolute URL to an action definition to run during stage1 to stage2 boot.")
	createCmd.Flags().StringVar(&cfBootStage2, "boot-stage2",
		"https://storage.googleapis.com/epoxy-%s/stage3_coreos/stage2to3.json",
		"Absolute URL to an action definition to run during stage2 to stage3 boot.")
	createCmd.Flags().StringVar(&cfBootStage3, "boot-stage3",
		"https://storage.googleapis.com/epoxy-%s/stage3_coreos/stage3post.json",
		"Absolute URL to an action definition to run after running stage3 boot.")
	createCmd.Flags().StringVar(&cfUpdateStage1, "update-stage1",
		"https://epoxy-boot-api.%s.measurementlab.net:4430/v1/storage/stage3_mlxupdate/stage1to2.ipxe",
		"Absolute URL to an action definition to run during stage1 to stage2 update.")
	createCmd.Flags().StringVar(&cfUpdateStage1JSON, "update-stage1-json",
		"https://storage.googleapis.com/epoxy-%s/stage3_mlxupdate/stage1to2.json",
		"Absolute URL to an action definition to run during stage1 to stage2 update.")
	createCmd.Flags().StringVar(&cfUpdateStage2, "update-stage2",
		"https://storage.googleapis.com/epoxy-%s/stage3_mlxupdate/stage2to3.json",
		"Absolute URL to an action definition to run during stage2 to stage3 update.")
	createCmd.Flags().StringVar(&cfUpdateStage3, "update-stage3",
		"https://storage.googleapis.com/epoxy-%s/stage3_mlxupdate/stage3post.json",
		"Absolute URL to an action definition to run after running stage3 update.")
}
