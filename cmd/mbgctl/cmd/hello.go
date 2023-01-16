/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/mbgctl/state"
	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
)

// helloCmd represents the hello command
var helloCmd = &cobra.Command{
	Use:   "hello",
	Short: "Hello command send hello message to all MBGs in thr MBG neighbor list",
	Long:  `Hello command send hello message to all MBGs in thr MBG neighbor list.`,
	Run: func(cmd *cobra.Command, args []string) {
		mbgPeerId, _ := cmd.Flags().GetString("mbgId")
		state.UpdateState()

		myMbgIP := state.GetMbgIP()
		if mbgPeerId == "" {
			hello2AllReq(myMbgIP)
		} else {
			helloReq(myMbgIP, mbgPeerId)
		}
	},
}

func init() {
	rootCmd.AddCommand(helloCmd)
	helloCmd.Flags().String("mbgId", "", "Send hello to specific MBG peer")

}

func helloReq(mbgIP, peerID string) {
	log.Printf("Start hello from to MBG peer %v", peerID)

	address := state.GetAddrStart() + mbgIP + "/hello/" + peerID

	//send hello
	j := []byte{}
	resp := httpAux.HttpPost(address, j, state.GetHttpClient())
	log.Infof(`Response message hello to MBG peer(%s) :  %s`, peerID, string(resp))
}

func hello2AllReq(mbgIP string) {
	log.Printf("Start hello to all MBG peers")

	address := state.GetAddrStart() + mbgIP + "/hello/"

	//send hello
	j := []byte{}
	resp := httpAux.HttpPost(address, j, state.GetHttpClient())
	log.Infof(`Response message hello to all MBG peers: %s`, string(resp))
}
