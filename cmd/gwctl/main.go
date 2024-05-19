// Copyright (c) The ClusterLink Authors.
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

// Copyright (c) 2022 The ClusterLink Authors.
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

// Copyright (C) The ClusterLink Authors.
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

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/clusterlink-net/clusterlink/cmd/gwctl/subcommand"
	"github.com/clusterlink-net/clusterlink/pkg/versioninfo"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "gwctl",
		Short: "gwctl is a CLI that sends a control message (REST API) to the gateway",
		Long: `gwctl CLI is part of the multi-cloud network project,
		that allow sending control messages (REST API) to publish, connect and update policies for services`,
		Version: versioninfo.Short(),
	}

	// Add all commands
	rootCmd.AddCommand(subcommand.InitCmd()) // init command of Gwctl
	rootCmd.AddCommand(createCmd())
	rootCmd.AddCommand(getCmd())
	rootCmd.AddCommand(deleteCmd())
	rootCmd.AddCommand(updateCmd())
	rootCmd.AddCommand(subcommand.ConfigCmd())

	logrus.SetLevel(logrus.WarnLevel)

	// Execute runs the cobra command of the gwctl
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// createCmd contains all the create commands of the CLI.
func createCmd() *cobra.Command {
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create",
		Long:  `Create`,
	}
	// Add all create commands
	createCmd.AddCommand(subcommand.PeerCreateCmd())
	createCmd.AddCommand(subcommand.ExportCreateCmd())
	createCmd.AddCommand(subcommand.ImportCreateCmd())
	createCmd.AddCommand(subcommand.PolicyCreateCmd())
	return createCmd
}

// deleteCmd contains all the delete commands of the CLI.
func deleteCmd() *cobra.Command {
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete",
		Long:  `Delete`,
	}
	// Add all delete commands
	deleteCmd.AddCommand(subcommand.PeerDeleteCmd())
	deleteCmd.AddCommand(subcommand.ExportDeleteCmd())
	deleteCmd.AddCommand(subcommand.ImportDeleteCmd())
	deleteCmd.AddCommand(subcommand.PolicyDeleteCmd())
	return deleteCmd
}

// updateCmd contains all the update commands of the CLI.
func updateCmd() *cobra.Command {
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update",
		Long:  `Update`,
	}
	// Add all update commands
	updateCmd.AddCommand(subcommand.PeerUpdateCmd())
	updateCmd.AddCommand(subcommand.ExportUpdateCmd())
	updateCmd.AddCommand(subcommand.ImportUpdateCmd())
	updateCmd.AddCommand(subcommand.PolicyUpdateCmd())
	return updateCmd
}

// getCmd contains all the get commands of the CLI.
func getCmd() *cobra.Command {
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get",
		Long:  `Get`,
	}
	// Add all get commands
	getCmd.AddCommand(subcommand.StateGetCmd())
	getCmd.AddCommand(subcommand.PeerGetCmd())
	getCmd.AddCommand(subcommand.ExportGetCmd())
	getCmd.AddCommand(subcommand.ImportGetCmd())
	getCmd.AddCommand(subcommand.PolicyGetCmd())
	getCmd.AddCommand(subcommand.MetricsGetCmd())
	getCmd.AddCommand(subcommand.AllGetCmd())
	return getCmd
}
