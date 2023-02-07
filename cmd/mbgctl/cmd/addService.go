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
var addServiceCmd = &cobra.Command{
	Use:   "addService",
	Short: "Add local service to the MBG",
	Long:  `Add local service to the MBG and save it also in the state of the mbgctl`,
	Run: func(cmd *cobra.Command, args []string) {
		serviceId, _ := cmd.Flags().GetString("id")
		serviceIp, _ := cmd.Flags().GetString("ip")
		description, _ := cmd.Flags().GetString("description")

		state.UpdateState()
		state.AddService(serviceId, serviceIp, description)
		addServiceReq(serviceId)

	},
}

func init() {
	rootCmd.AddCommand(addServiceCmd)
	addServiceCmd.Flags().String("id", "", "service id field")
	addServiceCmd.Flags().String("ip", "", "service ip to connect")
	addServiceCmd.Flags().String("description", "", "Service description to connect")
}

func addServiceReq(serviceId string) {
	s := state.GetService(serviceId)
	mbgIP := state.GetMbgIP()
	svcExp := s.Service

	address := state.GetAddrStart() + mbgIP + "/service"
	j, err := json.Marshal(protocol.ServiceRequest{Id: svcExp.Id, Ip: svcExp.Ip, Description: svcExp.Description})
	if err != nil {
		fmt.Printf("Unable to marshal json: %v", err)
	}

	//send
	resp := httpAux.HttpPost(address, j, state.GetHttpClient())
	fmt.Printf("Response message for adding service [%s]:  %s\n", svcExp.Id, string(resp))
}
