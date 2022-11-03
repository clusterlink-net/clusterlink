/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/gateway/state"
	"github.ibm.com/mbg-agent/pkg/client"
)

// connectCmd represents the connect command
var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "connect flow connection to the closest MBG",
	Long:  `connect flow connection to the closest MBG`,
	Run: func(cmd *cobra.Command, args []string) {
		svcName, _ := cmd.Flags().GetString("svcName")
		svcId, _ := cmd.Flags().GetString("svcId")
		svcNameDest, _ := cmd.Flags().GetString("svcNameDest")
		svcIdDest, _ := cmd.Flags().GetString("svcIdDest")
		state.UpdateState()

		if svcName == "" || svcId == "" || svcNameDest == "" || svcIdDest == "" {
			fmt.Println("Error: please insert all flag arguments for connect command")
			os.Exit(1)
		}
		svc := state.GetService(svcName, svcId)
		destSvc := state.GetService(svcNameDest, svcIdDest)

		connectClient(svc.Service.Ip, destSvc.Service.Ip)

	},
}

func init() {
	rootCmd.AddCommand(connectCmd)
	connectCmd.Flags().String("svcName", "", "Service name that the gateway is listen")
	connectCmd.Flags().String("svcId", "", "Service Id that the gateway is listen")
	connectCmd.Flags().String("svcNameDest", "", "Destination service name the gateway is connecting")
	connectCmd.Flags().String("svcIdDest", "", "Destination service id the gateway is connecting")

}

func connectClient(source, dest string) {
	var c client.MbgClient
	//TBD add validity check for the source and dest  IP
	c.InitClient(source, dest)
	c.RunClient()
}
