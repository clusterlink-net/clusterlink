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

package deletion

import (
	"github.com/spf13/cobra"
)

// NewCmdDelete returns a cobra.Command to run the delete command.
func NewCmdDelete() *cobra.Command {
	cmds := &cobra.Command{
		Use:   "delete",
		Short: "Delete ClusterLink resources",
		Long:  "Delete ClusterLink resources",
	}

	cmds.AddCommand(NewCmdDeletePeer())

	return cmds
}
