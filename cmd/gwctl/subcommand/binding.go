package subcommand

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.ibm.com/mbg-agent/cmd/gwctl/config"
	cmdutil "github.ibm.com/mbg-agent/cmd/util"
	"github.ibm.com/mbg-agent/pkg/api"
)

// bindingCreateOptions is the command line options for 'create binding'
type bindingCreateOptions struct {
	myID     string
	importID string
	peer     string
}

// BindingCreateCmd - create a binding command.
func BindingCreateCmd() *cobra.Command {
	o := bindingCreateOptions{}
	cmd := &cobra.Command{
		Use:   "binding",
		Short: "Create binding for a imported service to remote peer",
		Long:  `Create binding for a imported service to remote peer`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run()
		},
	}

	o.addFlags(cmd.Flags())
	cmdutil.MarkFlagsRequired(cmd, []string{"import", "peer"})

	return cmd
}

// addFlags registers flags for the CLI.
func (o *bindingCreateOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.myID, "myid", "", "gwctl ID")
	fs.StringVar(&o.importID, "import", "", "Imported service name to bind")
	fs.StringVar(&o.peer, "peer", "", "Remote peer to import the service from")
}

// run performs the execution of 'create binding' subcommand
func (o *bindingCreateOptions) run() error {
	g, err := config.GetClientFromID(o.myID)
	if err != nil {
		return err
	}

	err = g.Bindings.Create(&api.Binding{
		Spec: api.BindingSpec{
			Import: o.importID,
			Peer:   o.peer},
	})
	if err != nil {
		return err
	}

	fmt.Printf("Binding created successfully\n")
	return nil
}

// bindingDeleteOptions is the command line options for 'delete binding'
type bindingDeleteOptions struct {
	myID     string
	importID string
	peer     string
}

// BindingDeleteCmd - Delete a binding service command.
func BindingDeleteCmd() *cobra.Command {
	o := bindingDeleteOptions{}
	cmd := &cobra.Command{
		Use:   "binding",
		Short: "Deletes binding of a imported service to a remote peer",
		Long:  `Deletes binding of a imported service to a remote peer`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run()
		},
	}

	o.addFlags(cmd.Flags())
	cmdutil.MarkFlagsRequired(cmd, []string{"import"})

	return cmd
}

// addFlags registers flags for the CLI.
func (o *bindingDeleteOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.myID, "myid", "", "gwctl ID")
	fs.StringVar(&o.importID, "import", "", "Imported service name to unbind")
	fs.StringVar(&o.peer, "peer", "", "Remote peer to stop importing from")
}

// run performs the execution of the 'delete binding' subcommand
func (o *bindingDeleteOptions) run() error {
	g, err := config.GetClientFromID(o.myID)
	if err != nil {
		return err
	}

	err = g.Bindings.Delete(&api.Binding{
		Spec: api.BindingSpec{
			Import: o.importID,
			Peer:   o.peer},
	})
	if err != nil {
		return err
	}

	fmt.Printf("Binding was deleted successfully\n")
	return nil
}

// bindingGetOptions is the command line options for 'delete binding'
type bindingGetOptions struct {
	myID     string
	importID string
}

// BindingGetCmd - get a binding of imported service command.
func BindingGetCmd() *cobra.Command {
	o := bindingGetOptions{}
	cmd := &cobra.Command{
		Use:   "binding",
		Short: "Get the peer list corresponding to an import",
		Long:  `Get the peer list corresponding to an import`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run()
		},
	}

	o.addFlags(cmd.Flags())
	cmdutil.MarkFlagsRequired(cmd, []string{"import"})

	return cmd
}

// addFlags registers flags for the CLI.
func (o *bindingGetOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.myID, "myid", "", "gwctl ID")
	fs.StringVar(&o.importID, "import", "", "Imported service name to bind")
}

// run performs the execution of the 'get binding' subcommand
func (o *bindingGetOptions) run() error {
	g, err := config.GetClientFromID(o.myID)
	if err != nil {
		return err
	}

	bArr, err := g.Bindings.Get(o.importID)
	if err != nil {
		return err
	}

	fmt.Printf("Binding of the imported service %s:\n", o.importID)
	for i, b := range *bArr.(*[]api.Binding) {
		fmt.Printf("%d. Peer : %s \n", i+1, b.Spec.Peer)
	}
	return nil
}
