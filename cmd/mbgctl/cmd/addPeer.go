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
var addPeerCmd = &cobra.Command{
	Use:   "addPeer",
	Short: "Add MBG peer to MBG",
	Long:  `Add MBG peer to MBG`,
	Run: func(cmd *cobra.Command, args []string) {
		ip, _ := cmd.Flags().GetString("ip")
		id, _ := cmd.Flags().GetString("id")
		cport, _ := cmd.Flags().GetString("cport")
		state.UpdateState()
		addPeerReq(id, ip, cport)

	},
}

func init() {
	rootCmd.AddCommand(addPeerCmd)

	addPeerCmd.Flags().String("id", "", "MBG peer id")
	addPeerCmd.Flags().String("ip", "", "MBG peer ip")
	addPeerCmd.Flags().String("cport", "", "MBG peer control port")
}

func addPeerReq(peerId, peerIp, peerCport string) {
	mbgIP := state.GetMbgIP()
	address := state.GetAddrStart() + mbgIP + "/peer/" + peerId
	j, err := json.Marshal(protocol.PeerRequest{Id: peerId, Ip: peerIp, Cport: ":" + peerCport})
	if err != nil {
		fmt.Printf("Unable to marshal json: %v", err)
	}
	//send peer req
	fmt.Printf("Send addPeerReq %v", address)
	httpAux.HttpPost(address, j, state.GetHttpClient())
}
