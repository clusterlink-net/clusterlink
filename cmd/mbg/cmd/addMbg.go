/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
)

// addMbgCmd represents the addMbg command
var addMbgCmd = &cobra.Command{
	Use:   "addMbg",
	Short: "add the list of neighbor MBGs",
	Long:  `add the list of neighbor MBGs`,
	Run: func(cmd *cobra.Command, args []string) {
		ip, _ := cmd.Flags().GetString("ip")
		id, _ := cmd.Flags().GetString("id")
		cport, _ := cmd.Flags().GetString("cport")
		state.UpdateState()
		state.AddMbgNbr(id, ip, cport)
	},
}

func init() {
	rootCmd.AddCommand(addMbgCmd)

	addMbgCmd.Flags().String("id", "", "MBG neighbor id")
	addMbgCmd.Flags().String("ip", "", "MBG neighbor ip")
	addMbgCmd.Flags().String("cport", "", "MBG neighbor control port")
}
