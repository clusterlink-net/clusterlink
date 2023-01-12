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
	Short: "Add local service to the MBG -Use for Debug",
	Long:  `Add local service to the MBG -Use for Debug.`,
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetString("id")
		ip, _ := cmd.Flags().GetString("ip")
		state.UpdateState()
		log.Println("add local service")
		state.AddLocalService(id, ip)

	},
}

func init() {
	rootCmd.AddCommand(addServiceCmd)
	addServiceCmd.Flags().String("id", "", "Service id")
	addServiceCmd.Flags().String("ip", "", "Service ip")
}
