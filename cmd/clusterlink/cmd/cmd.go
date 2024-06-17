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

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/clusterlink-net/clusterlink/cmd/clusterlink/cmd/create"
	deletion "github.com/clusterlink-net/clusterlink/cmd/clusterlink/cmd/delete"
	"github.com/clusterlink-net/clusterlink/cmd/clusterlink/cmd/deploy"
)

// NewCLADMCommand returns a cobra.Command to run the clusterlink command.
func NewCLADMCommand() *cobra.Command {
	cmds := &cobra.Command{
		Use:          "clusterlink",
		Short:        "clusterlink: bootstrap a clink fabric",
		Long:         `clusterlink: bootstrap a clink fabric`,
		SilenceUsage: true,
	}

	cmds.AddCommand(create.NewCmdCreate())
	cmds.AddCommand(deploy.NewCmdDeploy())
	cmds.AddCommand(deletion.NewCmdDelete())

	return cmds
}
