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
var getServiceCmd = &cobra.Command{
	Use:   "getService",
	Short: "get service list from the MBG",
	Long:  `get service list from the MBG`,
	Run: func(cmd *cobra.Command, args []string) {
		serviceId, _ := cmd.Flags().GetString("id")
		serviceType, _ := cmd.Flags().GetString("type")
		state.UpdateState()
		mbgIP := state.GetMbgIP()
		if serviceId == "" {
			if serviceType == "local" {
				getAllLocalServicesReq(mbgIP)
			} else {
				getAllRemoteServicesReq(mbgIP)
			}
		} else {
			if serviceType == "local" {
				getLocalServiceReq(serviceId, mbgIP)
			} else {
				getRemoteServiceReq(serviceId, mbgIP)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(getServiceCmd)
	getServiceCmd.Flags().String("id", "", "service id field")
	getServiceCmd.Flags().String("type", "remote", "service type : remote/local")

}

func getAllLocalServicesReq(mbgIP string) {
	address := state.GetAddrStart() + mbgIP + "/service/"
	resp := httpAux.HttpGet(address, state.GetHttpClient())

	sArr := make(map[string]protocol.ServiceRequest)
	if err := json.Unmarshal(resp, &sArr); err != nil {
		fmt.Printf("Unable to unmarshal response :%v", err)
	}
	fmt.Printf("Local services:\n")
	i := 1
	for _, s := range sArr {
		state.AddService(s.Id, s.Ip, s.Description)
		fmt.Printf("%d) Service ID: %s IP: %s Description: %s\n", i, s.Id, s.Ip, s.Description)
		i++
	}
}
func getAllRemoteServicesReq(mbgIP string) {
	address := state.GetAddrStart() + mbgIP + "/remoteservice/"
	resp := httpAux.HttpGet(address, state.GetHttpClient())

	sArr := make(map[string][]protocol.ServiceRequest)
	if err := json.Unmarshal(resp, &sArr); err != nil {
		fmt.Printf("Unable to unmarshal response :%v", err)
	}
	fmt.Printf("Remote Services:\n")
	i := 1
	for _, sA := range sArr {
		for _, s := range sA {
			state.AddService(s.Id, s.Ip, s.Description)
			fmt.Printf("%d) Service ID: %s IP: %s MBGID: %s Description: %s\n", i, s.Id, s.Ip, s.MbgID, s.Description)
			i++
		}
	}
}

func getLocalServiceReq(serviceId, mbgIP string) {
	address := state.GetAddrStart() + mbgIP + "/service/" + serviceId
	resp := httpAux.HttpGet(address, state.GetHttpClient())

	var s protocol.ServiceRequest
	if err := json.Unmarshal(resp, &s); err != nil {
		fmt.Printf("Unable to unmarshal response :%v", err)
	}
	state.AddService(s.Id, s.Ip, s.Description)
	fmt.Printf("Local service ID: %s with IP: %s Description %s \n", s.Id, s.Ip, s.Description)
}

func getRemoteServiceReq(serviceId, mbgIP string) {
	address := state.GetAddrStart() + mbgIP + "/remoteservice/" + serviceId
	resp := httpAux.HttpGet(address, state.GetHttpClient())

	var sArr []protocol.ServiceRequest
	if err := json.Unmarshal(resp, &sArr); err != nil {
		fmt.Printf("Unable to unmarshal response :%v", err)
	}
	for _, s := range sArr {
		state.AddService(s.Id, s.Ip, s.Description)
		fmt.Printf("Remote service ID: %s with IP: %s MBGID %s Description %s \n", s.Id, s.Ip, s.MbgID, s.Description)
	}
}
