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
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/clusterlink-net/clusterlink/cmd/gwctl/config"
)

// ConfigCmd contains all the config commands of the CLI.
func ConfigCmd() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "config",
		Long:  `config`,
	}

	configCmd.AddCommand(currentContextCmd())
	configCmd.AddCommand(useContextCmd())
	return configCmd
}

// getContextCmd is the command line options for 'config current-context'.
type currentContextOptions struct {
}

// currentContextCmd - get the last gwctl context command to use.
func currentContextCmd() *cobra.Command {
	o := currentContextOptions{}
	cmd := &cobra.Command{

		Use:   "current-context",
		Short: "Get gwctl current context.",
		Long:  `Get gwctl current context.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run()
		},
	}

	return cmd
}

// run performs the execution of the 'config current-context' subcommand.
func (o *currentContextOptions) run() error {
	s, err := config.GetConfigFromID("")
	if err != nil {
		return err
	}

	sJSON, _ := json.MarshalIndent(s, "", " ")
	fmt.Println("gwctl current state\n", string(sJSON))
	return nil
}

// useContext is the command line options for 'config use-context'.
type useContextOptions struct {
	myID string
}

// useContextCmd - set gwctl context.
func useContextCmd() *cobra.Command {
	o := useContextOptions{}
	cmd := &cobra.Command{
		Use:   "use-context",
		Short: "use gwctl context.",
		Long:  `use gwctl context.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run()
		},
	}

	o.addFlags(cmd.Flags())

	return cmd
}

// addFlags registers flags for the CLI.
func (o *useContextOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.myID, "myid", "", "gwctl ID")
}

// run performs the execution of the 'config current-context' subcommand.
func (o *useContextOptions) run() error {
	c, err := config.GetConfigFromID(o.myID)
	if err != nil {
		return err
	}

	err = c.SetDefaultClient(o.myID)
	if err != nil {
		return err
	}

	fmt.Println("gwctl use context ", o.myID)
	return nil
}
