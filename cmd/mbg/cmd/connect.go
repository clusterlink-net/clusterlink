/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
	"github.ibm.com/mbg-agent/pkg/client"
	mbgSwitch "github.ibm.com/mbg-agent/pkg/mbg-switch"
	service "github.ibm.com/mbg-agent/pkg/serviceMap"
)

// connectCmd represents the connect command
var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "connect flow connection to the closest MBG",
	Long:  `connect flow connection to the closest MBG`,
	Run: func(cmd *cobra.Command, args []string) {
		svcId, _ := cmd.Flags().GetString("svcId")
		svcIdDest, _ := cmd.Flags().GetString("svcIdDest")
		state.UpdateState()

		if svcId == "" || svcIdDest == "" {
			log.Println("Error: please insert all flag arguments for connect command")
			os.Exit(1)
		}
		svc := state.GetLocalService(svcId) //TBD- Not use th incoming service name
		destSvc := state.GetRemoteService(svcIdDest)

		myIp := state.GetMyIp()
		srcIp := myIp + ":" + svc.LocalPort
		destIp := destSvc.Service.Ip

		log.Println("Connect service %v to service %v ", svc.Service.Id, destSvc.Service.Id)
		sendConnectMsg() //TBD
		connect(srcIp, destIp, destSvc.Service.Policy)

	},
}

func init() {
	rootCmd.AddCommand(connectCmd)
	connectCmd.Flags().String("svcId", "", "Service Id that the gateway is listen")
	connectCmd.Flags().String("svcIdDest", "", "Destination service id the gateway is connecting")

}

func connect(source, dest, policy string) {
	var s mbgSwitch.MbgSwitch
	var c client.MbgClient

	cListener := ":5000"
	var serverTarget string
	if policy == "Forward" {
		serverTarget = cListener
	} else if policy == "TCP-split" {
		serverTarget = service.GetPolicyIp(policy)
	} else {
		fmt.Println(policy, "- Policy  not exist use Forward")
		serverTarget = cListener
	}
	s.InitMbgSwitch(source, serverTarget)
	c.InitClient(cListener, dest)

	go c.RunClient()
	s.RunMbgSwitch()

}

func sendConnectMsg() {
	//TBD- send connect message to all gRpc
}
