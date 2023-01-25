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
var addPolicyEngineCmd = &cobra.Command{
	Use:   "addPolicyEngine",
	Short: "add the policy engine",
	Long:  `add the policy engine`,
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
	rootCmd.AddCommand(addPolicyEngineCmd)
	addPolicyEngineCmd.Flags().String("name", "", "Policy Name")
	addPolicyEngineCmd.Flags().String("desc", "", "Short description of policy")
	addPolicyEngineCmd.Flags().String("target", "", "Target endpoint(e.g.ip:port) to reach the policy agent")
	addPolicyEngineCmd.Flags().Bool("start", true, "Start the policy dispatcher (true/false)")
}
