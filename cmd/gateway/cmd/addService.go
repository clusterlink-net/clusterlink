/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/gateway/state"
)

// updateCmd represents the update command
var addServiceCmd = &cobra.Command{
	Use:   "addService",
	Short: "Add service to service list that expose to gateway",
	Long:  `Add service to service list that expose to gateway.`,
	Run: func(cmd *cobra.Command, args []string) {
		serviceId, _ := cmd.Flags().GetString("serviceId")
		serviceIp, _ := cmd.Flags().GetString("serviceIp")
		serviceDomain, _ := cmd.Flags().GetString("serviceDomain")
		state.UpdateState()

		defer state.SaveState()
		state.AddService(serviceId, serviceIp, serviceDomain)
	},
}

func init() {
	rootCmd.AddCommand(addServiceCmd)
	addServiceCmd.Flags().String("serviceId", "", "service id field")
	addServiceCmd.Flags().String("serviceIp", "", "service ip to connect")
	addServiceCmd.Flags().String("serviceDomain", "Internal", "service domain : Internal/Remote")

}
