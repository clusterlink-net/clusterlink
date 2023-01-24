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
var getPeerCmd = &cobra.Command{
	Use:   "getPeer",
	Short: "get service list from the MBG",
	Long:  `get service list from the MBG`,
	Run: func(cmd *cobra.Command, args []string) {
		peerId, _ := cmd.Flags().GetString("serviceId")
		state.UpdateState()
		if peerId == "" {
			getAllPeersReq()
		} else {
			getPeerReq(peerId)
		}

	},
}

func init() {
	rootCmd.AddCommand(getPeerCmd)
	getPeerCmd.Flags().String("perrId", "", "Peer id field")

}

func getAllPeersReq() {
	mbgIP := state.GetMbgIP()
	var address string
	address = state.GetAddrStart() + mbgIP + "/peer/"
	resp := httpAux.HttpGet(address, state.GetHttpClient())

	pArr := make(map[string]protocol.PeerRequest)
	if err := json.Unmarshal(resp, &pArr); err != nil {
		log.Fatal("getAllServicesReq Error :", err)
	}
	log.Infof("MBG peers:")
	for _, p := range pArr {
		log.Infof("MBG ID: %v IP: %v cPort %v", p.Id, p.Ip, p.Cport)
	}

}

func getPeerReq(peerId string) {
	mbgIP := state.GetMbgIP()
	var address string

	address = state.GetAddrStart() + mbgIP + "/peer/" + peerId

	//Send request
	resp := httpAux.HttpGet(address, state.GetHttpClient())

	var p protocol.PeerRequest
	if err := json.Unmarshal(resp, &p); err != nil {
		log.Fatal("getPeerReq Error :", err)
	}
	log.Infof("MBG peer details: ID: %v IP: %v cPort %v", p.Id, p.Ip, p.Cport)
}
