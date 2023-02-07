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

// exposeCmd represents the expose command
var exposeCmd = &cobra.Command{
	Use:   "expose",
	Short: "Expose command send an expose message to Multi-cloud Border Gateway",
	Long:  `Expose command send an expose message to Multi-cloud Border Gateway`,
	Run: func(cmd *cobra.Command, args []string) {
		serviceId, _ := cmd.Flags().GetString("serviceId")
		state.UpdateState()

		mbgIP := state.GetMbgIP()
		exposeReq(serviceId, mbgIP)

	},
}

func init() {
	rootCmd.AddCommand(exposeCmd)
	exposeCmd.Flags().String("serviceId", "", "Service Id for exposing")

}

func exposeReq(serviceId, mbgIP string) {
	s := state.GetService(serviceId)
	svcExp := s.Service

	address := state.GetAddrStart() + mbgIP + "/expose"
	j, err := json.Marshal(protocol.ExposeRequest{Id: svcExp.Id, Ip: svcExp.Ip, MbgID: ""})
	if err != nil {
		fmt.Errorf("Unable to marshal json %v", err)
	}
	//send expose
	resp := httpAux.HttpPost(address, j, state.GetHttpClient())
	fmt.Printf("Response message for exposing service [%s] :  %s\n", svcExp.Id, string(resp))
}
