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
	"github.ibm.com/mbg-agent/pkg/mbgDataplane"
	"github.ibm.com/mbg-agent/pkg/policyEngine"
	pb "github.ibm.com/mbg-agent/pkg/protocol/grpc"

	"google.golang.org/grpc"
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
		ConnectService(localPort, destIp, policy, connID)

	},
}

func init() {
	rootCmd.AddCommand(connectCmd)
	connectCmd.Flags().String("serviceId", "", "Source service id for connecting ")
	connectCmd.Flags().String("serviceIdDest", "", "Destination service id for connecting")
	connectCmd.Flags().String("policy", "Forward", "Policy connection")
	connectCmd.Flags().String("localPort", "", "Local for open listen server")

}

//Run server for Data connection - we have one server and client that we can add some network functions e.g: TCP-split
//By default we just forward the data
func ConnectService(svcListenPort, svcIp, policy, connName string) {

	srcIp := ":" + svcListenPort
	destIp := svcIp

	policyTarget := policyEngine.GetPolicyTarget(policy)
	if policyTarget == "" {
		// No Policy to be applied
		var forward mbgDataplane.MbgTcpForwarder

		forward.InitTcpForwarder(srcIp, destIp, connName)
		forward.RunTcpForwarder()
	} else {
		var ingress mbgDataplane.MbgTcpForwarder
		var egress mbgDataplane.MbgTcpForwarder

		ingress.InitTcpForwarder(srcIp, policyTarget, connName)
		egress.InitTcpForwarder(policyTarget, destIp, connName)
		ingress.RunTcpForwarder()
		egress.RunTcpForwarder()
	}

}

//Send control request to connect
func SendConnectReq(svcId, svcIdDest, svcPolicy, mbgIp string) (string, string, error) {
	log.Printf("Start connect Request to MBG %v for service %v", mbgIp, svcIdDest)

	conn, err := grpc.Dial(mbgIp, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Printf("Failed to connect grpc: %v", err)
		return "", "", fmt.Errorf("Connect Request Failed")
	}
	defer conn.Close()
	c := pb.NewConnectClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.ConnectCmd(ctx, &pb.ConnectRequest{Id: svcId, IdDest: svcIdDest, Policy: svcPolicy})
	if err != nil {
		log.Printf("Failed to send connect: %v", err)
		return "", "", fmt.Errorf("Connect Request Failed")
	}
	if r.GetMessage() == "Success" {
		log.Printf("Successfully Connected : Using Connection:Port - %s:%s", r.GetConnectType(), r.GetConnectDest())
		return r.GetConnectType(), r.GetConnectDest(), nil
	}
	log.Printf("[MBG %v] Failed to Connect : %s", state.GetMyId(), r.GetMessage())
	if "Connection already setup!" == r.GetMessage() {
		return r.GetConnectType(), r.GetConnectDest(), fmt.Errorf("Connection already setup!")
	} else {
		return "", "", fmt.Errorf("Connect Request Failed")
	}
}
