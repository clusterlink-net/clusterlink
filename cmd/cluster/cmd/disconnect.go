/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/cluster/state"
	pb "github.ibm.com/mbg-agent/pkg/protocol/grpc"
	"google.golang.org/grpc"
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
		SendDisconnectReq(svcId, svcIdDest, mbgIP)
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

func SendDisconnectReq(svcId, svcIdDest, mbgIP string) {
	log.Printf("Start disconnect Request to MBG %v for service %v:%v", mbgIP, svcId, svcIdDest)

	conn, err := grpc.Dial(mbgIP, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewDisconnectClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.DisconnectCmd(ctx, &pb.DisconnectRequest{Id: svcId, IdDest: svcIdDest})
	if err != nil {
		log.Fatalf("could not create user: %v", err)
	}
	log.Printf(`Response Connect message:  %s`, r.GetMessage())
}
