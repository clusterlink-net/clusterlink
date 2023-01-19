/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
	"github.ibm.com/mbg-agent/pkg/policyEngine"
)

// addPolicyCmd represents the addPolicy command
var addPolicyCmd = &cobra.Command{
	Use:   "addPolicy",
	Short: "add the list of Policies that the MBG supports",
	Long:  `add the list of Policies that the MBG supports`,
	Run: func(cmd *cobra.Command, args []string) {
		target, _ := cmd.Flags().GetString("target")
		start, _ := cmd.Flags().GetBool("start")

		state.UpdateState()
		state.GetEventManager().AssignPolicyDispatcher("http://" + target + "/policy")
		state.SaveState()
		if start {
			policyEngine.StartPolicyDispatcher(state.GetChiRouter(), target)
		}
	},
}

func init() {
	rootCmd.AddCommand(addPolicyCmd)
	addPolicyCmd.Flags().String("name", "", "Policy Name")
	addPolicyCmd.Flags().String("desc", "", "Short description of policy")
	addPolicyCmd.Flags().String("target", "", "Target endpoint(e.g.ip:port) to reach the policy agent")
	addPolicyCmd.Flags().Bool("start", true, "Start the policy dispatcher (true/false)")
}
