/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/cluster/state"
	handler "github.ibm.com/mbg-agent/pkg/protocol/http/cluster"
)

// connectCmd represents the connect command
var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "connect flow connection to the closest MBG",
	Long:  `connect flow connection to the closest MBG`,
	Run: func(cmd *cobra.Command, args []string) {
		svcId, _ := cmd.Flags().GetString("serviceId")
		svcIdDest, _ := cmd.Flags().GetString("serviceIdDest")
		//svcPolicy, _ := cmd.Flags().GetString("policy")

		state.UpdateState()

		if svcId == "" || svcIdDest == "" {
			fmt.Println("Error: please insert all flag arguments for connect command")
			os.Exit(1)
		}
		svc := state.GetService(svcId)
		destSvc := state.GetService(svcIdDest)
		name := state.GetId() + " egress: " + svcIdDest
		srcIp := svc.Service.Ip
		destIp := destSvc.Service.Ip
		log.Infof("[Cluster %v] Using %v:%v to connect IP-%v", state.GetId(), "TCP", destIp, destSvc.Service.Ip)
		handler.ConnectClient(svc.Service.Id, destSvc.Service.Id, srcIp, destIp, name)

	},
}

func init() {
	rootCmd.AddCommand(connectCmd)
	connectCmd.Flags().String("serviceId", "", "Service Id that the cluster is listen")
	connectCmd.Flags().String("serviceIdDest", "", "Destination service id the cluster is connecting")
	connectCmd.Flags().String("policy", "Forward", "Connection policy")
}
