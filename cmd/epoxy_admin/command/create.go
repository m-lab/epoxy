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

	"cloud.google.com/go/datastore"

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
	// Setup Datastore client.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := datastore.NewClient(ctx, fProject)
	rtx.Must(err, "Failed to create new datastore client")

	// Save the host record to Datstore.
	ds := storage.NewDatastoreConfig(client)

	h := &storage.Host{
		Name:          fHostname,
		IPv4Addr:      fAddress,
		UpdateEnabled: fUpdate,
		Extensions:    []string{fExtension},
		Boot: storage.Sequence{
			Stage1ChainURL: fmtURL(fBootStage1),
			Stage2ChainURL: fmtURL(fBootStage2),
			Stage3ChainURL: fmtURL(fBootStage3),
		},
		Update: storage.Sequence{
			Stage1ChainURL: fmtURL(fUpdateStage1),
			Stage2ChainURL: fmtURL(fUpdateStage2),
			Stage3ChainURL: fmtURL(fUpdateStage3),
		},
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
	createCmd.Flags().StringVar(&fHostname, "hostname", "",
		"Hostname of new record.")
	createCmd.Flags().StringVar(&fAddress, "address", "",
		"IP address of hostname.")
	createCmd.MarkFlagRequired("hostname")
	createCmd.MarkFlagRequired("address")

	// Local flags which will only apply when "create" is called directly.
	createCmd.Flags().StringVar(&fExtension, "extension", "allocate_k8s_token",
		"Name of an extension to enable for host.")
	createCmd.Flags().BoolVar(&fUpdate, "update", false,
		"Set Host.UpdateEnabled to true for an existing Host.")
	createCmd.Flags().StringVar(&fBootStage1, "boot-stage1",
		"https://storage.googleapis.com/epoxy-%s/stage3_coreos/stage1to2.ipxe",
		"Absolute URL to an action definition to run during stage1 to boot stage2.")
	createCmd.Flags().StringVar(&fBootStage2, "boot-stage2",
		"https://storage.googleapis.com/epoxy-%s/stage3_coreos/stage2to3.json",
		"Absolute URL to an action definition to run during stage2 to boot stage3.")
	createCmd.Flags().StringVar(&fBootStage3, "boot-stage3",
		"https://storage.googleapis.com/epoxy-%s/stage3_coreos/stage3post.json",
		"Absolute URL to an action definition to run after booting stage3.")
	createCmd.Flags().StringVar(&fUpdateStage1, "update-stage1",
		"https://storage.googleapis.com/epoxy-%s/stage3_mlxupdate/stage1to2.ipxe",
		"Absolute URL to an action definition to run during stage1 to boot stage2.")
	createCmd.Flags().StringVar(&fUpdateStage2, "update-stage2",
		"https://storage.googleapis.com/epoxy-%s/stage3_mlxupdate/stage2to3.json",
		"Absolute URL to an action definition to run during stage2 to boot stage3.")
	createCmd.Flags().StringVar(&fUpdateStage3, "update-stage3",
		"https://storage.googleapis.com/epoxy-%s/stage3_mlxupdate/stage3post.json",
		"Absolute URL to an action definition to run after booting stage3.")
}
