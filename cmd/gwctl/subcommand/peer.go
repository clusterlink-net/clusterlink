package subcommand

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	cmdutil "github.ibm.com/mbg-agent/cmd/util"
	"github.ibm.com/mbg-agent/pkg/admin"
	api "github.ibm.com/mbg-agent/pkg/api/admin"
)

// peerCreateOptions is the command line options for 'create peer'
type peerCreateOptions struct {
	myID string
	name string
	host string
	port uint16
}

// PeerCreateCmd - create a peer command.
func PeerCreateCmd() *cobra.Command {
	o := peerCreateOptions{}
	cmd := &cobra.Command{
		Use:   "peer",
		Short: "Create a peer",
		Long:  "Create a peer",
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run()
		},
	}

	o.addFlags(cmd.Flags())
	cmdutil.MarkFlagsRequired(cmd, []string{"name", "host", "port"})

	return cmd
}

// addFlags registers flags for the CLI.
func (o *peerCreateOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.myID, "myid", "", "gwctl ID")
	fs.StringVar(&o.name, "name", "", "Peer name")
	fs.StringVar(&o.host, "host", "", "Peer endpoint hostname (IP/DNS)")
	fs.Uint16Var(&o.port, "port", 0, "Peer endpoint port")
}

// run performs the execution of the 'create peer' subcommand
func (o *peerCreateOptions) run() error {
	g, err := admin.GetClientFromID(o.myID)
	if err != nil {
		return err
	}

	err = g.CreatePeer(
		api.Peer{
			Name: o.name,
			Spec: api.PeerSpec{
				Gateways: []api.Endpoint{{
					Host: o.host,
					Port: o.port,
				}},
			},
		},
	)
	if err != nil {
		return err
	}

	fmt.Printf("Peer was created successfully\n")
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
	g, err := admin.GetClientFromID(o.myID)
	if err != nil {
		return err
	}

	err = g.DeletePeer(api.Peer{Name: o.name})
	if err != nil {
		return err
	}

	fmt.Printf("Peer was deleted successfully\n")
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
	g, err := admin.GetClientFromID(o.myID)
	if err != nil {
		return err
	}

	if o.name == "" {
		pArr, err := g.GetPeers()
		if err != nil {
			return err
		}
		fmt.Printf("Peers:\n")
		for i, p := range pArr {
			fmt.Printf("%d. Peer: %v\n", i+1, p)
		}
	} else {
		peer, err := g.GetPeer(api.Peer{Name: o.name})
		if err != nil {
			return err
		}
		fmt.Printf("Peer: %+v\n", peer)
	}

	return nil
}
