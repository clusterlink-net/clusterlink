package create

import (
	"github.com/spf13/cobra"
)

// NewCmdCreate returns a cobra.Command to run the create command.
func NewCmdCreate() *cobra.Command {
	cmds := &cobra.Command{
		Use: "create",
	}

	cmds.AddCommand(NewCmdCreateFabric())
	cmds.AddCommand(NewCmdCreatePeer())

	return cmds
}
