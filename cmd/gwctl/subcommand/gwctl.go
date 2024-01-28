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
	"github.com/clusterlink-net/clusterlink/pkg/util"
	"github.com/clusterlink-net/clusterlink/pkg/util/rest"
)

// initOptions is the command line options for 'init'.
type initOptions struct {
	id             string
	gwIP           string
	gwPort         uint16
	certCa         string
	cert           string
	key            string
	dataplane      string
	policyEngineIP string
}

// InitCmd represents the init command.
func InitCmd() *cobra.Command {
	o := initOptions{}
	cmd := &cobra.Command{
		Use:   "init",
		Short: "A start command set all parameter state of gwctl (gw control)",
		Long:  `A start command set all parameter state of gwctl (gw control)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run()
		},
	}

	o.addFlags(cmd.Flags())
	cmdutil.MarkFlagsRequired(cmd, []string{"id", "gwIP", "gwPort", "certca", "cert", "key"})
	return cmd
}

// addFlags registers flags for the CLI.
func (o *initOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.id, "id", "", "gwctl ID")
	fs.StringVar(&o.gwIP, "gwIP", "", "IP address of the gateway (that the gwctl is connected)")
	fs.Uint16Var(&o.gwPort, "gwPort", 0, "Port of the gateway (that the gwctl is connected)")
	fs.StringVar(&o.certCa, "certca", "", "Path to the Root Certificate Auth File (.pem)")
	fs.StringVar(&o.cert, "cert", "", "Path to the Certificate File (.pem)")
	fs.StringVar(&o.key, "key", "", "Path to the Key File (.pem)")
	fs.StringVar(&o.dataplane, "dataplane", "mtls", "tcp/mtls based dataplane proxies")
	fs.StringVar(&o.policyEngineIP, "policyEngineIP", "",
		"IP address of the policy engine, if empty will use the same value as gwIP")
}

// run performs the execution of the 'init' subcommand.
func (o *initOptions) run() error {
	if _, err := util.ParseTLSFiles(o.certCa, o.cert, o.key); err != nil {
		return err
	}

	_, err := config.NewClientConfig(&config.ClientConfig{
		ID:             o.id,
		GwIP:           o.gwIP,
		GwPort:         o.gwPort,
		CaFile:         o.certCa,
		CertFile:       o.cert,
		KeyFile:        o.key,
		Dataplane:      o.dataplane,
		PolicyEngineIP: o.policyEngineIP,
	})

	return err
}

// stateGetOptions is the command line options for 'get state'.
type stateGetOptions struct {
	myID string
}

// StateGetCmd  - get gwctl parameters.
func StateGetCmd() *cobra.Command {
	o := stateGetOptions{}
	cmd := &cobra.Command{
		Use:   "state",
		Short: "Get gwctl information",
		Long:  `Get gwctl information`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run()
		},
	}
	o.addFlags(cmd.Flags())

	return cmd
}

// addFlags registers flags for the CLI.
func (o *stateGetOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.myID, "myid", "", "gwctl ID")
}

// run performs the execution of the 'get state' subcommand.
func (o *stateGetOptions) run() error {
	d, err := config.GetConfigFromID(o.myID)
	if err != nil {
		return err
	}

	sJSON, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	fmt.Println(string(sJSON))
	return nil
}

type allGetOptions struct {
	myID string
}

// AllGetCmd - get all gateways objects.
func AllGetCmd() *cobra.Command {
	o := allGetOptions{}
	cmd := &cobra.Command{
		Use:   "all",
		Short: "Get all information of the gateway (peers, exports, imports, bindings policies)",
		Long:  "Get all information of the gateway (peers, exports, imports, bindings policies)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run()
		},
	}
	o.addFlags(cmd.Flags())

	return cmd
}

// addFlags registers flags for the CLI.
func (o *allGetOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.myID, "myid", "", "gwctl ID")
}

// run performs the execution of the 'get all' subcommand.
func (o *allGetOptions) run() error {
	g, err := config.GetClientFromID(o.myID)
	if err != nil {
		return err
	}

	objects := map[string]*rest.Client{
		"Peers":    g.Peers,
		"Exports":  g.Exports,
		"Imports":  g.Imports,
		"Bindings": g.Bindings,
	}

	for name, o := range objects {
		fmt.Printf("%s:\n", name)
		d, err := o.List()
		if err != nil {
			return fmt.Errorf("error: %w", err)
		}
		sJSON, err := json.Marshal(d)
		if err != nil {
			return fmt.Errorf("error: %w", err)
		}
		fmt.Println(string(sJSON))
	}

	// Policy
	p := policyGetOptions{myID: o.myID}
	return p.run()
}
