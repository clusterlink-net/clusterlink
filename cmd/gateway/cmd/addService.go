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
		servicePolicy, _ := cmd.Flags().GetString("servicePolicy")
		state.UpdateState()

		defer state.SaveState()
		state.UpdateService(serviceId, serviceIp, serviceDomain, servicePolicy)
	},
}

func init() {
	rootCmd.AddCommand(addServiceCmd)
	addServiceCmd.Flags().String("serviceId", "", "service id field")
	addServiceCmd.Flags().String("serviceIp", "", "service ip to connect")
	addServiceCmd.Flags().String("serviceDomain", "", "service domain : inner/remote")
	addServiceCmd.Flags().String("servicePolicy", "", "service policy : Forward, Tcp split and etc. ")

}
