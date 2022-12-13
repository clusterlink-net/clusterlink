/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/cluster/state"
	handler "github.ibm.com/mbg-agent/pkg/protocol/http/cluster"
)

// connectCmd represents the connect command
var disconnectCmd = &cobra.Command{
	Use:   "disconnect",
	Short: "disconnect existing service pair connection",
	Long:  `disconnect existing service pair connection`,
	Run: func(cmd *cobra.Command, args []string) {
		svcId, _ := cmd.Flags().GetString("serviceId")
		svcIdDest, _ := cmd.Flags().GetString("serviceIdDest")

		state.UpdateState()

		if svcId == "" || svcIdDest == "" {
			fmt.Println("Error: please insert all flag arguments for connect command")
			os.Exit(1)
		}
		// svc := state.GetService(svcId)
		// destSvc := state.GetService(svcIdDest)
		mbgIP := state.GetMbgIP()
		handler.DisconnectReq(svcId, svcIdDest, mbgIP)
		disconnectClient(svcId, svcIdDest)

	},
}

func init() {
	rootCmd.AddCommand(disconnectCmd)
	disconnectCmd.Flags().String("serviceId", "", "Service Id that the cluster is listen")
	disconnectCmd.Flags().String("serviceIdDest", "", "Destination service id the cluster is connecting")
}

func disconnectClient(svcId, svcIdDest string) {
	state.CloseOpenConnection(svcId, svcIdDest)
}
