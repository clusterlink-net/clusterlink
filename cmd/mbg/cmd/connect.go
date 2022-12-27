/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
	"github.ibm.com/mbg-agent/pkg/mbgDataplane"
)

// connectCmd represents the connect command
var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Create flow connection to the MBG/cluster -for Debug only",
	Long:  `Create flow connection to the MBG/cluster -for Debug only`,
	Run: func(cmd *cobra.Command, args []string) {
		svcId, _ := cmd.Flags().GetString("serviceId")
		svcIdDest, _ := cmd.Flags().GetString("serviceIdDest")
		localPort, _ := cmd.Flags().GetString("localPort")
		policy, _ := cmd.Flags().GetString("policy")

		state.UpdateState()

		if svcId == "" || svcIdDest == "" {
			log.Println("Error: please insert all flag arguments for connect command")
			os.Exit(1)
		}
		var destIp string
		if state.IsServiceLocal(svcIdDest) {
			destSvc := state.GetLocalService(svcIdDest)
			destIp = destSvc.Service.Ip
		} else { //For Remote service
			destSvc := state.GetRemoteService(svcIdDest)
			destIp = destSvc.Service.Ip
		}

		log.Printf("Connect service %v to service %v \n", svcId, svcIdDest)
		connID := svcId + ":" + svcIdDest
		mbgDataplane.ConnectService(localPort, destIp, policy, connID, nil, nil)

	},
}

func init() {
	rootCmd.AddCommand(connectCmd)
	connectCmd.Flags().String("serviceId", "", "Source service id for connecting ")
	connectCmd.Flags().String("serviceIdDest", "", "Destination service id for connecting")
	connectCmd.Flags().String("policy", "Forward", "Policy connection")
	connectCmd.Flags().String("localPort", "", "Local for open listen server")

}
