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
var addServiceCmd = &cobra.Command{
	Use:   "addService",
	Short: "Add local service to the MBG",
	Long:  `Add local service to the MBG and save it also in the state of the mbgctl`,
	Run: func(cmd *cobra.Command, args []string) {
		serviceId, _ := cmd.Flags().GetString("serviceId")
		serviceIp, _ := cmd.Flags().GetString("serviceIp")
		state.UpdateState()
		state.AddService(serviceId, serviceIp)
		addServiceReq(serviceId)

	},
}

func init() {
	rootCmd.AddCommand(addServiceCmd)
	addServiceCmd.Flags().String("serviceId", "", "service id field")
	addServiceCmd.Flags().String("serviceIp", "", "service ip to connect")
}

func addServiceReq(serviceId string) {
	log.Printf("Start addService %v to ", serviceId)
	s := state.GetService(serviceId)
	mbgIP := state.GetMbgIP()
	svcExp := s.Service
	log.Printf("Service %v", s)

	address := "http://" + mbgIP + "/service"
	j, err := json.Marshal(protocol.ServiceRequest{Id: svcExp.Id, Ip: svcExp.Ip})
	if err != nil {
		log.Fatal(err)
	}
	//send expose
	resp := httpAux.HttpPost(address, j)
	log.Infof(`Response message for serive %s expose :  %s`, svcExp.Id, string(resp))
}
