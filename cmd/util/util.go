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

package util

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// MarkFlagsRequired mark the flags that required for the command line.
func MarkFlagsRequired(cmd *cobra.Command, flags []string) {
	for _, f := range flags {
		if err := cmd.MarkFlagRequired(f); err != nil {
			fmt.Printf("Error marking required flag '%s': %v\n", f, err)
			os.Exit(1)
		}
	}
}
