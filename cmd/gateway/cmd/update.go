/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/gateway/state"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Add service to service list that expose to gateway",
	Long:  `Add service to service list that expose to gateway.`,
	Run: func(cmd *cobra.Command, args []string) {
		serviceName, _ := cmd.Flags().GetString("serviceName")
		serviceId, _ := cmd.Flags().GetString("serviceId")
		serviceIp, _ := cmd.Flags().GetString("serviceIp")
		serviceDomain, _ := cmd.Flags().GetString("serviceDomain")
		servicePolicy, _ := cmd.Flags().GetString("servicePolicy")
		state.UpdateState()

		defer state.SaveState()
		state.UpdateService(serviceName, serviceId, serviceIp, serviceDomain, servicePolicy)
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().String("serviceName", "", "service name field")
	updateCmd.Flags().String("serviceId", "", "service id field")
	updateCmd.Flags().String("serviceIp", "", "service ip to connect")
	updateCmd.Flags().String("serviceDomain", "", "service domain : inner/remote")
	updateCmd.Flags().String("servicePolicy", "", "service policy : Forward, Tcp split and etc. ")

}
