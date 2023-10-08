package main

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/clusterlink-net/clusterlink/cmd/gwctl/subcommand"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "gwctl",
		Short: "gwctl is a CLI that sends a control message (REST API) to the gateway",
		Long: `gwctl CLI is part of the multi-cloud network project,
		that allow sending control messages (REST API) to publish, connect and update policies for services`,
	}
	// Add all commands
	rootCmd.AddCommand(subcommand.InitCmd()) // init command of Gwctl
	rootCmd.AddCommand(createCmd())
	rootCmd.AddCommand(getCmd())
	rootCmd.AddCommand(deleteCmd())
	rootCmd.AddCommand(subcommand.ConfigCmd())

	logrus.SetLevel(logrus.WarnLevel)

	// Execute runs the cobra command of the gwctl
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// createCmd contains all the create commands of the CLI.
func createCmd() *cobra.Command {
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create",
		Long:  `Create`,
	}
	// Add all create commands
	createCmd.AddCommand(subcommand.PeerCreateCmd())
	createCmd.AddCommand(subcommand.ExportCreateCmd())
	createCmd.AddCommand(subcommand.ImportCreateCmd())
	createCmd.AddCommand(subcommand.BindingCreateCmd())
	createCmd.AddCommand(subcommand.PolicyCreateCmd())
	return createCmd
}

// deleteCmd contains all the delete commands of the CLI.
func deleteCmd() *cobra.Command {
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete",
		Long:  `Delete`,
	}
	// Add all delete commands
	deleteCmd.AddCommand(subcommand.PeerDeleteCmd())
	deleteCmd.AddCommand(subcommand.ExportDeleteCmd())
	deleteCmd.AddCommand(subcommand.ImportDeleteCmd())
	deleteCmd.AddCommand(subcommand.BindingDeleteCmd())
	deleteCmd.AddCommand(subcommand.PolicyDeleteCmd())
	return deleteCmd
}

// getCmd contains all the get commands of the CLI.
func getCmd() *cobra.Command {
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get",
		Long:  `Get`,
	}
	// Add all get commands
	getCmd.AddCommand(subcommand.StateGetCmd())
	getCmd.AddCommand(subcommand.PeerGetCmd())
	getCmd.AddCommand(subcommand.ExportGetCmd())
	getCmd.AddCommand(subcommand.ImportGetCmd())
	getCmd.AddCommand(subcommand.BindingGetCmd())
	getCmd.AddCommand(subcommand.PolicyGetCmd())
	getCmd.AddCommand(subcommand.MetricsGetCmd())
	getCmd.AddCommand(subcommand.AllGetCmd())
	return getCmd
}
