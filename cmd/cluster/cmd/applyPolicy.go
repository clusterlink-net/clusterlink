/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/cluster/state"
)

// applyPolicyCmd represents the applyPolicy command
var applyPolicyCmd = &cobra.Command{
	Use:   "applyPolicy",
	Short: "An applyPolicy command send the MBG the policy for dedicated service",
	Long:  `An applyPolicy command send the MBG the policy for dedicated service.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("applyPolicy called")
		policy, _ := cmd.Flags().GetString("policy")
		destServiceName, _ := cmd.Flags().GetString("destServiceName")
		myServiceName, _ := cmd.Flags().GetString("myServiceName")
		sendPolicy(policy, destServiceName, myServiceName)
	},
}

func init() {
	rootCmd.AddCommand(applyPolicyCmd)
	applyPolicyCmd.Flags().String("policy", "", "Policy setting for service inside MBG")
	applyPolicyCmd.Flags().String("destServiceName", "", "listen host:port of server side MBG")
	applyPolicyCmd.Flags().String("myServiceName", "", "Service name that need to connect")
}

func sendPolicy(policy, destServiceName, myServiceName string) {
	mbgIP := state.GetMbgIP()
	fmt.Println("send policy", policy, "for service", myServiceName, "to MBG", mbgIP)

}
