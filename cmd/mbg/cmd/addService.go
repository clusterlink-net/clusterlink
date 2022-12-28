/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
)

// addServiceCmd represents the addService command
var addServiceCmd = &cobra.Command{
	Use:   "addService",
	Short: "Add local CLuster Ip to MBG",
	Long:  `Add local CLuster Ip to MBG.`,
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetString("id")
		ip, _ := cmd.Flags().GetString("ip")
		domain, _ := cmd.Flags().GetString("domain")
		state.UpdateState()
		log.Println("add local service")
		state.AddLocalService(id, ip, domain)

	},
}

func init() {
	rootCmd.AddCommand(addServiceCmd)
	addServiceCmd.Flags().String("id", "", "Local cluster id")
	addServiceCmd.Flags().String("ip", "", "Local cluster ip")
	addServiceCmd.Flags().String("domain", "internal", "Local cluster domain")
}
