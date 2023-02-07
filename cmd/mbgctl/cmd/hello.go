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
	address := state.GetAddrStart() + mbgIP + "/hello/" + peerID

	//send hello
	j := []byte{}
	resp := httpAux.HttpPost(address, j, state.GetHttpClient())
	fmt.Printf("Response message hello to MBG peer(%s) :  %s\n", peerID, string(resp))
}

func hello2AllReq(mbgIP string) {
	address := state.GetAddrStart() + mbgIP + "/hello/"

	//send hello
	j := []byte{}
	resp := httpAux.HttpPost(address, j, state.GetHttpClient())
	fmt.Printf("Hello Response message from all MBG peers: %s\n", string(resp))
}
