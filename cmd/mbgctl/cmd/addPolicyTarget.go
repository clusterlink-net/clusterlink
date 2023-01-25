/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/mbgctl/state"
)

// addPolicyCmd represents the addPolicy command
var addPolicyCmd = &cobra.Command{
	Use:   "addPolicy",
	Short: "add the list of Policies that the MBG supports",
	Long:  `add the list of Policies that the MBG supports`,
	Run: func(cmd *cobra.Command, args []string) {
		target, _ := cmd.Flags().GetString("target")

		state.UpdateState()
		state.AssignPolicyDispatcher("http://" + target + "/policy")
	},
}

func init() {
	rootCmd.AddCommand(addPolicyCmd)
	addPolicyCmd.Flags().String("target", "", "Target endpoint(e.g.ip:port) to reach the policy agent")
}
