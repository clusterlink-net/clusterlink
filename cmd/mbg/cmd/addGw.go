/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
)

// addGwCmd represents the addGw command
var addGwCmd = &cobra.Command{
	Use:   "addGw",
	Short: "add local GW IP",
	Long:  `add local GW IP.`,
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetString("id")
		ip, _ := cmd.Flags().GetString("ip")
		state.UpdateState()
		log.Println("add local Gateway")
		state.SetLocalGw(id, ip)

	},
}

func init() {
	rootCmd.AddCommand(addGwCmd)
	addGwCmd.Flags().String("id", "", "Local gateway Id")
	addGwCmd.Flags().String("ip", "", "Local gateway IP")

}
