/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/mbgctl/state"
	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
)

// updateCmd represents the update command
var delServiceCmd = &cobra.Command{
	Use:   "delService",
	Short: "del local service to the MBG",
	Long:  `del local service to the MBG and save it also in the state of the mbgctl`,
	Run: func(cmd *cobra.Command, args []string) {
		serviceId, _ := cmd.Flags().GetString("id")
		serviceType, _ := cmd.Flags().GetString("type")

		state.UpdateState()
		state.DelService(serviceId)
		mbgIP := state.GetMbgIP()
		if serviceType == "local" {
			delLocalServiceReq(serviceId, mbgIP)
		} else {
			delLocalServiceReq(serviceId, mbgIP)
		}

	},
}

func init() {
	rootCmd.AddCommand(delServiceCmd)
	delServiceCmd.Flags().String("id", "", "service id field")
	delServiceCmd.Flags().String("type", "local", "Choose which type of service to delete remote/local")
}

func delLocalServiceReq(serviceId, mbgIP string) {
	address := state.GetAddrStart() + mbgIP + "/service/" + serviceId
	//send
	resp := httpAux.HttpDelete(address, nil, state.GetHttpClient())
	fmt.Printf("Response message for deleting service [%s]:%s \n", serviceId, string(resp))
}
