package cmd

import (
	"github.com/spf13/cobra"
	api "github.ibm.com/mbg-agent/pkg/api"
)

// addPolicyCmd represents the addPolicy command
var addPolicyEngineCmd = &cobra.Command{
	Use:   "addPolicyEngine",
	Short: "add the policy engine",
	Long:  `add the policy engine`,
	Run: func(cmd *cobra.Command, args []string) {
		mId, _ := cmd.Flags().GetString("myid")
		target, _ := cmd.Flags().GetString("target")
		start, _ := cmd.Flags().GetBool("start")
		zeroTrust, _ := cmd.Flags().GetBool("zeroTrust")
		m := api.Mbg{Id: mId}
		m.AddPolicyEngine(target, start, zeroTrust)
	},
}

func init() {
	rootCmd.AddCommand(addPolicyEngineCmd)
	addPolicyEngineCmd.Flags().String("target", "", "Target endpoint(e.g.ip:port) to reach the policy agent")
	addPolicyEngineCmd.Flags().Bool("start", true, "Start the policy dispatcher (true/false)")
	addPolicyEngineCmd.Flags().Bool("zeroTrust", false, "deny(true)/allow(false) by default all incoming traffic")

}
