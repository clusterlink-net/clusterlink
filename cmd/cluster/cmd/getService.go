/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/cluster/state"
	"github.ibm.com/mbg-agent/pkg/protocol"

	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
)

// updateCmd represents the update command
var getServiceCmd = &cobra.Command{
	Use:   "getService",
	Short: "get service list from the MBG",
	Long:  `get service list from the MBG`,
	Run: func(cmd *cobra.Command, args []string) {
		serviceId, _ := cmd.Flags().GetString("serviceId")
		state.UpdateState()
		if serviceId == "" {
			getAllServicesReq()
		} else {
			getServiceReq(serviceId)
		}
	},
}

func init() {
	rootCmd.AddCommand(getServiceCmd)
	getServiceCmd.Flags().String("serviceId", "", "service id field")

}

func getAllServicesReq() {
	mbgIP := state.GetMbgIP()
	address := "http://" + mbgIP + "/service/"

	resp := httpAux.HttpGet(address)

	sArr := make(map[string]protocol.ServiceRequest)
	if err := json.Unmarshal(resp, &sArr); err != nil {
		log.Fatal("getAllServicesReq Error :", err)
	}
	for _, s := range sArr {
		state.AddService(s.Id, s.Ip, s.Domain)
		log.Infof(`Response message from MBG getting service: %s with ip: %s`, s.Id, s.Ip)
	}

}

func getServiceReq(serviceId string) {
	mbgIP := state.GetMbgIP()
	address := "http://" + mbgIP + "/service/" + serviceId
	//Send request
	resp := httpAux.HttpGet(address)

	var s protocol.ServiceRequest
	if err := json.Unmarshal(resp, &s); err != nil {
		log.Fatal("getServiceReq Error :", err)
	}
	state.AddService(s.Id, s.Ip, s.Domain)
	log.Infof(`Response message from MBG getting service: %s with ip: %s`, s.Id, s.Ip)
}
