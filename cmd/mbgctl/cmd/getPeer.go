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
var getPeerCmd = &cobra.Command{
	Use:   "getPeer",
	Short: "get service list from the MBG",
	Long:  `get service list from the MBG`,
	Run: func(cmd *cobra.Command, args []string) {
		peerId, _ := cmd.Flags().GetString("id")
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
	getPeerCmd.Flags().String("id", "", "Peer id field")

}

func getAllPeersReq() {
	mbgIP := state.GetMbgIP()
	var address string
	address = state.GetAddrStart() + mbgIP + "/peer/"
	resp := httpAux.HttpGet(address, state.GetHttpClient())

	pArr := make(map[string]protocol.PeerRequest)
	if err := json.Unmarshal(resp, &pArr); err != nil {
		fmt.Printf("Unable to unmarshal response :%v", err)
	}
	fmt.Println("MBG peers:")
	i := 1
	for _, p := range pArr {
		fmt.Printf("%d) MBG ID: %v IP: %v:%v\n", i, p.Id, p.Ip, p.Cport)
		i++
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
		fmt.Printf("Unable to unmarshal json :%v", err)
	}
	fmt.Printf("MBG peer details: ID: %s IP: %v:%v\n", p.Id, p.Ip, p.Cport)
}
