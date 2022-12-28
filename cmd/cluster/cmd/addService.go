/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/cluster/state"
	handler "github.ibm.com/mbg-agent/pkg/protocol/http/cluster"
)

// updateCmd represents the update command
var addServiceCmd = &cobra.Command{
	Use:   "addService",
	Short: "Add service to local cluster and send it to the MBG",
	Long:  `Add service to local cluster and send it to the MBG`,
	Run: func(cmd *cobra.Command, args []string) {
		serviceId, _ := cmd.Flags().GetString("serviceId")
		serviceIp, _ := cmd.Flags().GetString("serviceIp")
		serviceDomain, _ := cmd.Flags().GetString("serviceDomain")
		state.UpdateState()
		state.AddService(serviceId, serviceIp, serviceDomain)
		handler.AddServiceReq(serviceId)

	},
}

func init() {
	rootCmd.AddCommand(addServiceCmd)
	addServiceCmd.Flags().String("serviceId", "", "service id field")
	addServiceCmd.Flags().String("serviceIp", "", "service ip to connect")
	addServiceCmd.Flags().String("serviceDomain", "Internal", "service domain : Internal/Remote")
}
