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
	cmdutil "github.com/clusterlink-net/clusterlink/cmd/util"
	"github.com/clusterlink-net/clusterlink/pkg/api"
)

// importOptions is the command line options for 'create import' or 'update import'.
type importOptions struct {
	myID string
	name string
	host string
	port uint16
}

// ImportCreateCmd - create an imported service.
func ImportCreateCmd() *cobra.Command {
	o := importOptions{}
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Create an imported service",
		Long:  `Create an imported service that can be bounded to other peers`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run(false)
		}}

	o.addFlags(cmd.Flags())
	cmdutil.MarkFlagsRequired(cmd, []string{"name", "port"})

	return cmd
}

// ImportUpdateCmd - update an imported service.
func ImportUpdateCmd() *cobra.Command {
	o := importOptions{}
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Update an imported service",
		Long:  `Update an imported service that can be bounded to other peers`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run(true)
		}}

	o.addFlags(cmd.Flags())
	cmdutil.MarkFlagsRequired(cmd, []string{"name", "port"})

	return cmd
}

// addFlags registers flags for the CLI.
func (o *importOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.myID, "myid", "", "gwctl ID")
	fs.StringVar(&o.name, "name", "", "Imported service name")
	fs.StringVar(&o.host, "host", "", "Imported service endpoint (IP/DNS), if unspecified, uses the service name")
	fs.Uint16Var(&o.port, "port", 0, "Imported service port")
}

// run performs the execution of the 'create import' or 'update import' subcommand.
func (o *importOptions) run(isUpdate bool) error {
	g, err := config.GetClientFromID(o.myID)
	if err != nil {
		return err
	}

	importOperation := g.Imports.Create
	if isUpdate {
		importOperation = g.Imports.Update
	}

	err = importOperation(&api.Import{
		Name: o.name,
		Spec: api.ImportSpec{
			Service: api.Endpoint{
				Host: o.host,
				Port: o.port},
		},
	})
	if err != nil {
		return err
	}

	return nil
}

// importDeleteOptions is the command line options for 'delete import'.
type importDeleteOptions struct {
	myID string
	name string
}

// ImportDeleteCmd - delete an imported service command.
func ImportDeleteCmd() *cobra.Command {
	o := importDeleteOptions{}
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Delete an imported service",
		Long:  `Delete an imported service`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run()
		},
	}

	o.addFlags(cmd.Flags())
	cmdutil.MarkFlagsRequired(cmd, []string{"name"})

	return cmd
}

// addFlags registers flags for the CLI.
func (o *importDeleteOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.myID, "myid", "", "gwctl ID")
	fs.StringVar(&o.name, "name", "", "Imported service name")
}

// run performs the execution of the 'delete import' subcommand.
func (o *importDeleteOptions) run() error {
	g, err := config.GetClientFromID(o.myID)
	if err != nil {
		return err
	}

	err = g.Imports.Delete(o.name)
	if err != nil {
		return err
	}

	return nil
}

// importGetOptions is the command line options for 'get import'.
type importGetOptions struct {
	myID string
	name string
}

// ImportGetCmd - get imported service command.
func ImportGetCmd() *cobra.Command {
	o := importGetOptions{}
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Get an imported service",
		Long:  `Get an imported service`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run()
		},
	}
	o.addFlags(cmd.Flags())

	return cmd
}

// addFlags registers flags for the CLI.I.
func (o *importGetOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.myID, "myid", "", "gwctl ID")
	fs.StringVar(&o.name, "name", "", "Imported service name. If empty gets all imported services.")
}

// run performs the execution of the 'get import' subcommand.
func (o *importGetOptions) run() error {
	g, err := config.GetClientFromID(o.myID)
	if err != nil {
		return err
	}

	if o.name == "" {
		sArr, err := g.Imports.List()
		if err != nil {
			return err
		}
		fmt.Printf("Imported services:\n")
		for i, s := range *sArr.(*[]api.Import) {
			fmt.Printf("%d. Imported Name: %s. Endpoint %v\n", i+1, s.Name, s.Spec.Service)
		}
	} else {
		imp, err := g.Imports.Get(o.name)
		if err != nil {
			return err
		}

		impJSON, err := json.MarshalIndent(imp, "", "  ")
		if err != nil {
			return err
		}

		fmt.Printf("%s\n", string(impJSON))
	}

	return nil
}
