/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/pkg/policyEngine"
)

// addPolicyCmd represents the addPolicy command
var addPolicyCmd = &cobra.Command{
	Use:   "addPolicy",
	Short: "add the list of Policies that the MBG supports",
	Long:  `add the list of Policies that the MBG supports`,
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		desc, _ := cmd.Flags().GetString("desc")
		target, _ := cmd.Flags().GetString("target")
		ptype, _ := cmd.Flags().GetString("type")

		policyType, err := strconv.Atoi(ptype)
		if err == nil {
			policyEngine.AddPolicy(name, desc, target, policyType)
		} else {
			log.Errorf("Invalid Policy Type")
		}
	},
}

func init() {
	rootCmd.AddCommand(addPolicyCmd)

	addPolicyCmd.Flags().String("name", "", "Policy Name")
	addPolicyCmd.Flags().String("desc", "", "Short description of policy")
	addPolicyCmd.Flags().String("target", "", "Target endpoint(e.g.ip:port) to reach the policy agent")
	addPolicyCmd.Flags().String("type", "", "Policy Type")
}
