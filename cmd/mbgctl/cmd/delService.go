/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/mbgctl/state"
	"github.ibm.com/mbg-agent/pkg/protocol"
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
		serviceMbg, _ := cmd.Flags().GetString("mbg")

		state.UpdateState()
		state.DelService(serviceId)
		mbgIP := state.GetMbgIP()
		if serviceType == "local" {
			delLocalServiceReq(serviceId, mbgIP)
		} else {
			delRemoteServiceReq(serviceId, serviceMbg, mbgIP)
		}

	},
}

func init() {
	rootCmd.AddCommand(delServiceCmd)
	delServiceCmd.Flags().String("id", "", "service id field")
	delServiceCmd.Flags().String("type", "local", "Choose which type of service to delete remote/local")
	delServiceCmd.Flags().String("mbg", "", "service mbg field for remote service")
}

func delLocalServiceReq(serviceId, mbgIP string) {
	address := state.GetAddrStart() + mbgIP + "/service/" + serviceId
	//send
	resp := httpAux.HttpDelete(address, nil, state.GetHttpClient())
	fmt.Printf("Response message for deleting service [%s]:%s \n", serviceId, string(resp))
}

func delRemoteServiceReq(serviceId, serviceMbg, mbgIP string) {
	address := state.GetAddrStart() + mbgIP + "/remoteservice/" + serviceId
	j, err := json.Marshal(protocol.ServiceRequest{Id: serviceId, MbgID: serviceMbg})
	if err != nil {
		fmt.Printf("Unable to marshal json: %v", err)
	}

	//send
	resp := httpAux.HttpDelete(address, j, state.GetHttpClient())
	fmt.Printf("Response message for deleting service [%s]:%s \n", serviceId, string(resp))
}
