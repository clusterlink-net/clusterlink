/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
)

// addClusterCmd represents the addCluster command
var addClusterCmd = &cobra.Command{
	Use:   "addCluster",
	Short: "Add local CLuster Ip to MBG",
	Long:  `Add local CLuster Ip to MBG.`,
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetString("id")
		ip, _ := cmd.Flags().GetString("ip")
		state.UpdateState()
		log.Println("add local Cluster")
		state.SetLocalCluster(id, ip)

	},
}

func init() {
	rootCmd.AddCommand(addClusterCmd)
	addClusterCmd.Flags().String("id", "", "Local cluster id")
	addClusterCmd.Flags().String("ip", "", "Local cluster ip")

}
