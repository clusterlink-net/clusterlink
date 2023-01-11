/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
)

// addMbgctlCmd represents the addMbgctl command
var addMbgctlCmd = &cobra.Command{
	Use:   "addMbgctl",
	Short: "Add mbgctl information of the mbgctl that can access to the  MBG",
	Long:  `Add mbgctl information of the mbgctl that can access to the  MBG.`,
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetString("id")
		ip, _ := cmd.Flags().GetString("ip")
		state.UpdateState()
		log.Println("add mbgctl")
		state.SetMbgctl(id, ip)

	},
}

func init() {
	rootCmd.AddCommand(addMbgctlCmd)
	addMbgctlCmd.Flags().String("id", "", "set mbgctl(mbg control) id")
	addMbgctlCmd.Flags().String("ip", "", "set mbgctl(mbg control) ip")

}
