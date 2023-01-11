/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/cluster/state"
	handler "github.ibm.com/mbg-agent/pkg/protocol/http/cluster"
)

// updateCmd represents the update command
var addPeerCmd = &cobra.Command{
	Use:   "addPeer",
	Short: "Add MBG peer to MBG",
	Long:  `Add MBG peer to MBG`,
	Run: func(cmd *cobra.Command, args []string) {
		ip, _ := cmd.Flags().GetString("ip")
		id, _ := cmd.Flags().GetString("id")
		cport, _ := cmd.Flags().GetString("cport")
		state.UpdateState()
		handler.AddPeerReq(id, ip, cport)

	},
}

func init() {
	rootCmd.AddCommand(addPeerCmd)

	addPeerCmd.Flags().String("id", "", "MBG peer id")
	addPeerCmd.Flags().String("ip", "", "MBG peer ip")
	addPeerCmd.Flags().String("cport", "", "MBG peer control port")
}
