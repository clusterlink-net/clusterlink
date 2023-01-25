/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"
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
		serviceId, _ := cmd.Flags().GetString("Id")
		servicetype, _ := cmd.Flags().GetString("type")
		state.UpdateState()
		if serviceId == "" {
			getAllServicesReq(servicetype)
		} else {
			getServiceReq(serviceId, servicetype)
		}
	},
}

func init() {
	rootCmd.AddCommand(getServiceCmd)
	getServiceCmd.Flags().String("Id", "", "service id field")
	getServiceCmd.Flags().String("type", "remote", "service type : remote/local")

}

func getAllServicesReq(servicetype string) {
	mbgIP := state.GetMbgIP()
	var address string
	if servicetype == "local" {
		address = state.GetAddrStart() + mbgIP + "/service/"
	} else {
		address = state.GetAddrStart() + mbgIP + "/remoteservice/"
	}
	resp := httpAux.HttpGet(address, state.GetHttpClient())

	sArr := make(map[string]protocol.ServiceRequest)
	if err := json.Unmarshal(resp, &sArr); err != nil {
		log.Fatal("getAllServicesReq Error :", err)
	}
	log.Infof("MBG services:")
	for _, s := range sArr {
		state.AddService(s.Id, s.Ip, s.Description)
		log.Infof("Service: %s IP: %s MBGID: %s Description: %s", s.Id, s.Ip, s.MbgID, s.Description)
	}

}

func getServiceReq(serviceId, servicetype string) {
	mbgIP := state.GetMbgIP()
	var address string
	if servicetype == "local" {
		address = state.GetAddrStart() + mbgIP + "/service/" + serviceId
	} else {
		address = state.GetAddrStart() + mbgIP + "/remoteservice/" + serviceId
	}

	//Send request
	resp := httpAux.HttpGet(address, state.GetHttpClient())

	var s protocol.ServiceRequest
	if err := json.Unmarshal(resp, &s); err != nil {
		log.Fatal("getServiceReq Error :", err)
	}
	state.AddService(s.Id, s.Ip, s.Description)
	log.Infof(`Response message from MBG getting service: %s with IP: %s MBGID %s Description %s`, s.Id, s.Ip, s.MbgID, s.Description)
}
