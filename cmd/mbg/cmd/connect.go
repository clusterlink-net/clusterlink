/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
	"github.ibm.com/mbg-agent/pkg/client"
	pb "github.ibm.com/mbg-agent/pkg/protocol"
	"github.ibm.com/mbg-agent/pkg/server"
	service "github.ibm.com/mbg-agent/pkg/serviceMap"

	"google.golang.org/grpc"
)

// connectCmd represents the connect command
var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "connect flow connection to the closest MBG",
	Long:  `connect flow connection to the closest MBG`,
	Run: func(cmd *cobra.Command, args []string) {
		svcId, _ := cmd.Flags().GetString("serviceId")
		svcIdDest, _ := cmd.Flags().GetString("serviceIdDest")
		policy, _ := cmd.Flags().GetString("policy")

		state.UpdateState()

		if svcId == "" || svcIdDest == "" {
			log.Println("Error: please insert all flag arguments for connect command")
			os.Exit(1)
		}
		var listenPort, destIp string
		if state.IsServiceLocal(svcIdDest) {
			destSvc := state.GetLocalService(svcIdDest)
			listenPort = destSvc.LocalDataPort
			destIp = destSvc.Service.Ip
		} else { //For Remtote service
			destSvc := state.GetRemoteService(svcIdDest)
			listenPort = destSvc.LocalDataPort
			destIp = destSvc.Service.Ip
		}

		log.Println("Connect service %v to service %v ", svcId, svcIdDest)
		ConnectService(listenPort, destIp, policy)

	},
}

func init() {
	rootCmd.AddCommand(connectCmd)
	connectCmd.Flags().String("serviceId", "", "Source service id for connecting ")
	connectCmd.Flags().String("serviceIdDest", "", "Destination service id for connecting")
	connectCmd.Flags().String("policy", "", "Policy connection")

}

//Run server for Data connection - we have one server and client that we can add some network functions e.g: TCP-split
//By default we just forward the data
func ConnectService(svcListenPort, svcIp, policy string) {
	var s server.MbgServer
	var c client.MbgClient

	srcIp := ":" + svcListenPort
	destIp := svcIp

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
	s.InitServer(srcIp, serverTarget)
	c.InitClient(cListener, destIp)

	go c.RunClient()
	s.RunServer()
}

//Send control request to connect
func SendConnectReq(svcId, svcIdDest, svcPolicy, mbgIp string) {
	log.Printf("Start connect Request to MBG%v for service %v", mbgIp, svcIdDest)

	conn, err := grpc.Dial(mbgIp, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewConnectClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.ConnectCmd(ctx, &pb.ConnectRequest{Id: svcId, IdDest: svcIdDest, Policy: svcPolicy})
	if err != nil {
		log.Fatalf("could not create user: %v", err)
	}
	log.Printf(`Response Connect message:  %s`, r.GetMessage())
}
