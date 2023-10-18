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

package subcommand

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// LogGetCmd prints out the controlplane log
var LogGetCmd = &cobra.Command{
	Use:   "log",
	Short: "Get mbg log file",
	Long:  `Get mbg log file`,
	Run: func(cmd *cobra.Command, args []string) {
		runCmd("cat /root/.gw/gw.log")
	},
}

// StateGetCmd prints out the controlplane state
var StateGetCmd = &cobra.Command{
	Use:   "state",
	Short: "Get mbg state",
	Long:  `Get mbg state`,
	Run: func(cmd *cobra.Command, args []string) {
		runCmd("cat /root/.gw/gwApp")
	},
}

// RunCmd executes os cmd and print the output
func runCmd(c string) {
	argSplit := strings.Split(c, " ")
	cmd := exec.Command(argSplit[0], argSplit[1:]...) //nolint:gosec // G204: Subprocess launched by package local calls only
	stdout, err := cmd.CombinedOutput()
	if err != nil {
		log.Println(err.Error())
		return
	}

	// Print the output
	fmt.Println(string(stdout))
}
