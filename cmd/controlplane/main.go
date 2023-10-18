// Copyright 2023 The ClusterLink Authors.
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

package main

import (
	"os"

	"github.com/clusterlink-net/clusterlink/cmd/controlplane/subcommand"

	"github.com/spf13/cobra"
)

func main() {
	// rootCmd represents the base command when called without any subcommands
	var rootCmd = &cobra.Command{
		Use:   "mbg",
		Short: "MBG Root",
		Long:  `MBG Root`,
	}

	rootCmd.AddCommand(subcommand.StartCmd())
	rootCmd.AddCommand(getCmd())

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// getCmd contains all the get commands of the CLI.
func getCmd() *cobra.Command {
	var getCmd = &cobra.Command{
		Use:   "get",
		Short: "Get",
		Long:  `Get`,
	}
	// Get Log
	getCmd.AddCommand(subcommand.LogGetCmd)
	// Get mbg state
	getCmd.AddCommand(subcommand.StateGetCmd)
	return getCmd
}
