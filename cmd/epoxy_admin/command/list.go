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
	"regexp"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/m-lab/epoxy/storage"
	"github.com/m-lab/go/rtx"
	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Lists ePoxy Host records from Datastore",
	Long: `
USAGE:

    TODO: implement list.
`,
	Run: runList,
}

func runList(cmd *cobra.Command, args []string) {
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
	r, err := regexp.Compile(lfHostname)
	rtx.Must(err, "Failed to compile given hostname pattern: %q", lfHostname)

	for _, h := range hosts {
		if !r.MatchString(h.Name) {
			continue
		}
		fmt.Printf("Listing: %s\n", h.Name)
		fmt.Println(h.String())
	}
}

func init() {
	rootCmd.AddCommand(listCmd)

	// Required local flags.
	listCmd.Flags().StringVar(&lfHostname, "hostname", "",
		"Hostname of new record.")
	listCmd.MarkFlagRequired("hostname")
}
