/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/mbgctl/state"
	"github.ibm.com/mbg-agent/pkg/protocol"
	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
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
		disconnectReq(svcId, svcIdDest, mbgIP)
		disconnectClient(svcId, svcIdDest)

	},
}

func init() {
	rootCmd.AddCommand(disconnectCmd)
	disconnectCmd.Flags().String("serviceId", "", "Service Id of the connection")
	disconnectCmd.Flags().String("serviceIdDest", "", "Destination service of the connection")
}

func disconnectClient(svcId, svcIdDest string) {
	state.CloseOpenConnection(svcId, svcIdDest)
}

func disconnectReq(svcId, svcIdDest, mbgIP string) {
	log.Printf("Start disconnect Request to MBG %v for service %v:%v", mbgIP, svcId, svcIdDest)
	address := "http://" + mbgIP + "/connect"

	j, err := json.Marshal(protocol.DisconnectRequest{Id: svcId, IdDest: svcIdDest})
	if err != nil {
		log.Fatal(err)
	}
	//send expose
	resp := httpAux.HttpDelete(address, j)
	log.Infof(`Service %s disconnect for message: %s`, svcId, string(resp))
}
