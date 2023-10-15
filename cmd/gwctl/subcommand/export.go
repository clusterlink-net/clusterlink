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
	"net"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/clusterlink-net/clusterlink/cmd/gwctl/config"
	cmdutil "github.com/clusterlink-net/clusterlink/cmd/util"
	"github.com/clusterlink-net/clusterlink/pkg/api"
)

// exportCreateOptions is the command line options for 'create export'
type exportCreateOptions struct {
	myID     string
	name     string
	host     string
	port     uint16
	external string
}

// ExportCreateCmd - Create an exported service.
func ExportCreateCmd() *cobra.Command {
	o := exportCreateOptions{}
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Create an exported service",
		Long:  `Create an exported service that can be accessed by other peers`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run()
		},
	}

	o.addFlags(cmd.Flags())
	cmdutil.MarkFlagsRequired(cmd, []string{"name", "port"})
	return cmd
}

// addFlags registers flags for the CLI.
func (o *exportCreateOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.myID, "myid", "", "gwctl ID")
	fs.StringVar(&o.name, "name", "", "Exported service name")
	fs.StringVar(&o.host, "host", "", "Exported service endpoint hostname (IP/DNS), if unspecified, uses the service name")
	fs.Uint16Var(&o.port, "port", 0, "Exported service port")
	fs.StringVar(&o.external, "external", "", "External endpoint <host>:<port, which the exported service will be connected")
}

// run performs the execution of the 'create export' subcommand
func (o *exportCreateOptions) run() error {
	var exEndpoint api.Endpoint
	g, err := config.GetClientFromID(o.myID)
	if err != nil {
		return err
	}

	if o.external != "" {
		exHost, exPort, err := net.SplitHostPort(o.external)
		if err != nil {
			return err
		}

		if exHost == "" {
			return fmt.Errorf("missing host in address")
		}

		exPortInt, err := strconv.Atoi(exPort)
		if err != nil {
			return err
		}

		exEndpoint = api.Endpoint{
			Host: exHost,
			Port: uint16(exPortInt),
		}
	}

	err = g.Exports.Create(&api.Export{
		Name: o.name,
		Spec: api.ExportSpec{
			Service: api.Endpoint{
				Host: o.host,
				Port: o.port},
			ExternalService: exEndpoint,
		},
	})
	if err != nil {
		return err
	}

	fmt.Printf("Exported service created successfully\n")
	return nil
}

// exportDeleteOptions is the command line options for 'delete export'
type exportDeleteOptions struct {
	myID string
	name string
}

// ExportDeleteCmd - delete an exported service command.
func ExportDeleteCmd() *cobra.Command {
	o := exportDeleteOptions{}
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Delete an exported service",
		Long:  `Delete an exported service`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run()
		},
	}

	o.addFlags(cmd.Flags())
	cmdutil.MarkFlagsRequired(cmd, []string{"name"})

	return cmd
}

// addFlags registers flags for the CLI.
func (o *exportDeleteOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.myID, "myid", "", "gwctl ID")
	fs.StringVar(&o.name, "name", "", "Exported service name")
}

// run performs the execution of the 'delete export' subcommand
func (o *exportDeleteOptions) run() error {
	g, err := config.GetClientFromID(o.myID)
	if err != nil {
		return err
	}

	err = g.Exports.Delete(o.name)
	if err != nil {
		return err
	}

	fmt.Println("Exported service was deleted successfully")
	return nil
}

// exportGetOptions is the command line options for 'get export'
type exportGetOptions struct {
	myID string
	name string
}

// ExportGetCmd - get an exported service command.
func ExportGetCmd() *cobra.Command {
	o := exportGetOptions{}
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Get an exported service list",
		Long:  `Get an exported service list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run()
		},
	}

	o.addFlags(cmd.Flags())

	return cmd
}

// addFlags registers flags for the CLI.
func (o *exportGetOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.myID, "myid", "", "gwctl ID")
	fs.StringVar(&o.name, "name", "", "Exported service name. If empty gets all exported services.")
}

// run performs the execution of the 'get export' subcommand
func (o *exportGetOptions) run() error {
	g, err := config.GetClientFromID(o.myID)
	if err != nil {
		return err
	}

	if o.name == "" {
		sArr, err := g.Exports.List()
		if err != nil {
			return err
		}
		fmt.Printf("Exported services:\n")
		for i, s := range *sArr.(*[]api.Export) {
			fmt.Printf("%d. Service Name: %s. Endpoint: %v\n", i+1, s.Name, s.Spec.Service)
		}
	} else {
		s, err := g.Exports.Get(o.name)
		if err != nil {
			return err
		}
		fmt.Printf("Exported service :%+v\n", s.(*api.Export))
	}

	return nil
}
