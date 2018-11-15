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
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Flag variables available to all subcommands.
var (
	fProject string
)

// Flag variables used only by the create & update commands. Since only one
// command is ever run at a time, it is safe to use them for both cases.
var (
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

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "epoxy_admin",
	Short: "Administer Datastore for ePoxy Server",
	Long: `
USAGE:

  epoxy_admin is a minimal client for adding ePoxy Host records to Datastore
  for testing. This command is ONLY for testing. Host record management by
  direct access to Datastore should not be supported.
`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Persistent flags, which will be global for all subcommands.
	rootCmd.PersistentFlags().StringVar(&fProject, "project", "mlab-sandbox", "GCP project ID.")
}
