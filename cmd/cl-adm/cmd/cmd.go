package cmd

import (
	"github.com/spf13/cobra"

	"github.com/clusterlink-net/clusterlink/cmd/cl-adm/cmd/create"
)

// NewCLADMCommand returns a cobra.Command to run the cl-adm command.
func NewCLADMCommand() *cobra.Command {
	cmds := &cobra.Command{
		Use:          "cl-adm",
		Short:        "cl-adm: bootstrap a clink fabric",
		Long:         `cl-adm: bootstrap a clink fabric`,
		SilenceUsage: true,
	}

	cmds.AddCommand(create.NewCmdCreate())

	return cmds
}
