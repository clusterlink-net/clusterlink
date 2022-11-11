/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
)

// updateMbgCmd represents the updateMbg command
var updateMbgCmd = &cobra.Command{
	Use:   "updateMbg",
	Short: "Update the list of neighbor MBGs",
	Long:  `Update the list of neighbor MBGs`,
	Run: func(cmd *cobra.Command, args []string) {
		ip, _ := cmd.Flags().GetString("ip")
		id, _ := cmd.Flags().GetString("id")
		state.UpdateState()
		state.UpdateMbgArr(id, ip)
	},
}

func init() {
	rootCmd.AddCommand(updateMbgCmd)
	updateMbgCmd.Flags().String("id", "", "Multi-cloud Border Gateway id")
	updateMbgCmd.Flags().String("ip", "", "Multi-cloud Border Gateway ip")
}
