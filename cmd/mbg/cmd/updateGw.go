/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
)

// updateGwCmd represents the updateGw command
var updateGwCmd = &cobra.Command{
	Use:   "updateGw",
	Short: "Update local GW IP",
	Long:  `Update local GW IP.`,
	Run: func(cmd *cobra.Command, args []string) {
		ip, _ := cmd.Flags().GetString("ip")
		state.UpdateState()
		log.Println("Update local Gateway")
		state.SetLocalGw(ip)

	},
}

func init() {
	rootCmd.AddCommand(updateGwCmd)
	updateGwCmd.Flags().String("ip", "", "Update local gateway Ip")
}
