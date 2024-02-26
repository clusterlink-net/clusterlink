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

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/clusterlink-net/clusterlink/cmd/cl-adm/cmd/create"
	"github.com/clusterlink-net/clusterlink/cmd/cl-adm/cmd/deploy"
)

// NewCLADMCommand returns a cobra.Command to run the cl-adm command.
func NewCLADMCommand() *cobra.Command {
	cmds := &cobra.Command{
		Use:          "cl-adm",
		Short:        "cl-adm: bootstrap a clink fabric",
		Long:         `cl-adm: bootstrap a clink fabric`,
		SilenceUsage: true,
	}

	cmds.AddCommand(create.NewCmdCreate())
	cmds.AddCommand(deploy.NewCmdDeploy())

	return cmds
}
