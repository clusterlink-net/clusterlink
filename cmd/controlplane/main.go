package main

import (
	"os"

	"github.ibm.com/mbg-agent/cmd/controlplane/subcommand"

	"github.com/spf13/cobra"
)

func main() {
	// rootCmd represents the base command when called without any subcommands
	var rootCmd = &cobra.Command{
		Use:   "mbg",
		Short: "MBG Root",
		Long:  `MBG Root`,
	}

	rootCmd.AddCommand(subcommand.StartCmd())
	rootCmd.AddCommand(getCmd())

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// getCmd contains all the get commands of the CLI.
func getCmd() *cobra.Command {
	var getCmd = &cobra.Command{
		Use:   "get",
		Short: "Get",
		Long:  `Get`,
	}
	// Get Log
	getCmd.AddCommand(subcommand.LogGetCmd)
	// Get mbg state
	getCmd.AddCommand(subcommand.StateGetCmd)
	return getCmd
}
