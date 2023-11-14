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

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/clusterlink-net/clusterlink/cmd/gwctl/config"
	cmdutil "github.com/clusterlink-net/clusterlink/cmd/util"
	"github.com/clusterlink-net/clusterlink/pkg/api"
)

// peerCreateOptions is the command line options for 'create peer' or 'update peer'.
type peerOptions struct {
	myID string
	name string
	host string
	port uint16
}

// PeerCreateCmd - create a peer command.
func PeerCreateCmd() *cobra.Command {
	o := peerOptions{}
	cmd := &cobra.Command{
		Use:   "peer",
		Short: "Create a peer",
		Long:  "Create a peer",
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run(false)
		},
	}

	o.addFlags(cmd.Flags())
	cmdutil.MarkFlagsRequired(cmd, []string{"name", "host", "port"})

	return cmd
}

// PeerUpdateCmd - update a peer command.
func PeerUpdateCmd() *cobra.Command {
	o := peerOptions{}
	cmd := &cobra.Command{
		Use:   "peer",
		Short: "Update a peer",
		Long:  "Update a peer",
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run(true)
		},
	}

	o.addFlags(cmd.Flags())
	cmdutil.MarkFlagsRequired(cmd, []string{"name", "host", "port"})

	return cmd
}

// addFlags registers flags for the CLI.
func (o *peerOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.myID, "myid", "", "gwctl ID")
	fs.StringVar(&o.name, "name", "", "Peer name")
	fs.StringVar(&o.host, "host", "", "Peer endpoint hostname (IP/DNS)")
	fs.Uint16Var(&o.port, "port", 0, "Peer endpoint port")
}

// run performs the execution of the 'create peer' or 'update peer' subcommand.
func (o *peerOptions) run(isUpdate bool) error {
	g, err := config.GetClientFromID(o.myID)
	if err != nil {
		return err
	}

	peerOperation := g.Peers.Create
	if isUpdate {
		peerOperation = g.Peers.Update
	}

	err = peerOperation(&api.Peer{
		Name: o.name,
		Spec: api.PeerSpec{
			Gateways: []api.Endpoint{{
				Host: o.host,
				Port: o.port,
			}},
		},
	})
	if err != nil {
		return err
	}

	return nil
}

// peerDeleteOptions is the command line options for 'delete peer'
type peerDeleteOptions struct {
	myID string
	name string
}

// PeerDeleteCmd - delete peer command.
func PeerDeleteCmd() *cobra.Command {
	o := peerDeleteOptions{}
	cmd := &cobra.Command{
		Use:   "peer",
		Short: "Delete a peer",
		Long:  `Delete a peer`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run()
		},
	}

	o.addFlags(cmd.Flags())
	cmdutil.MarkFlagsRequired(cmd, []string{"name"})
	return cmd
}

// addFlags registers flags for the CLI.
func (o *peerDeleteOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.myID, "myid", "", "gwctl ID")
	fs.StringVar(&o.name, "name", "", "Peer name")
}

// run performs the execution of the 'delete peer' subcommand
func (o *peerDeleteOptions) run() error {
	g, err := config.GetClientFromID(o.myID)
	if err != nil {
		return err
	}

	err = g.Peers.Delete(o.name)
	if err != nil {
		return err
	}

	return nil
}

// peerGetOptions is the command line options for 'get peer'
type peerGetOptions struct {
	myID string
	name string
}

// PeerGetCmd - get peer command.
func PeerGetCmd() *cobra.Command {
	o := peerGetOptions{}
	cmd := &cobra.Command{
		Use:   "peer",
		Short: "get a peer",
		Long:  `get a peer`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run()
		},
	}

	o.addFlags(cmd.Flags())

	return cmd
}

// addFlags registers flags for the CLI.I.I.
func (o *peerGetOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.myID, "myid", "", "gwctl ID")
	fs.StringVar(&o.name, "name", "", "Peer name. If empty gets all peers")
}

// run performs the execution of the 'get peer' subcommand
func (o *peerGetOptions) run() error {
	g, err := config.GetClientFromID(o.myID)
	if err != nil {
		return err
	}

	if o.name == "" {
		pArr, err := g.Peers.List()
		if err != nil {
			return err
		}
		fmt.Printf("Peers:\n")
		for i, p := range *pArr.(*[]api.Peer) {
			fmt.Printf("%d. Peer: %v\n", i+1, p)
		}
	} else {
		peer, err := g.Peers.Get(o.name)
		if err != nil {
			return err
		}
		fmt.Printf("Peer: %+v\n", peer.(*api.Peer))
	}

	return nil
}
