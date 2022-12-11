/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/cluster/state"
	"github.ibm.com/mbg-agent/pkg/clusterProxy"
	pb "github.ibm.com/mbg-agent/pkg/protocol"
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
		svcPolicy, _ := cmd.Flags().GetString("policy")
		SendReq, _ := cmd.Flags().GetString("SendConnectReq")

		state.UpdateState()

		if svcId == "" || svcIdDest == "" {
			fmt.Println("Error: please insert all flag arguments for connect command")
			os.Exit(1)
		}
		svc := state.GetService(svcId)
		destSvc := state.GetService(svcIdDest)
		mbgIP := state.GetMbgIP()
		if SendReq == "true" {
			connectType, DestPort, err := SendConnectReq(svcId, svcIdDest, svcPolicy, mbgIP)

			if err != nil {
				log.Infof("[Cluster %v]: Connection request to MBG fail- %v", state.GetId(), err.Error())
			}
			log.Infof("[Cluster %v] Using %v:%v to connect IP-%v", state.GetId(), connectType, DestPort, destSvc.Service.Ip)
			name := state.GetId() + " egress: " + svcIdDest
			srcIp := svc.Service.Ip
			destIp := destSvc.Service.Ip + ":" + DestPort
			connectClient(svc.Service.Id, destSvc.Service.Id, srcIp, destIp, name)
		}
	},
}

func init() {
	rootCmd.AddCommand(connectCmd)
	connectCmd.Flags().String("serviceId", "", "Service Id that the cluster is listen")
	connectCmd.Flags().String("serviceIdDest", "", "Destination service id the cluster is connecting")
	connectCmd.Flags().String("policy", "Forward", "Connection policy")
	connectCmd.Flags().String("SendConnectReq", "true", "Decide if to send connection request to MBG default:True")

}

func connectClient(svcId, svcIdDest, sourceIp, destIp, connName string) {
	var c clusterProxy.ProxyClient
	var stopChan = make(chan os.Signal, 2) //creating channel for interrupt
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	state.AddOpenConnection(svcId, svcIdDest, os.Getpid())

	c.InitClient(sourceIp, destIp, connName)
	done := &sync.WaitGroup{}
	done.Add(1)
	go c.RunClient(done)

	<-stopChan // wait for SIGINT
	log.Infof("Receive SIGINT for connection from %v to %v \n", svcId, svcIdDest)
	c.CloseConnection()
	done.Wait()
	log.Infof("Connection from %v to %v is close\n", svcId, svcIdDest)

}

func SendConnectReq(svcId, svcIdDest, svcPolicy, mbgIP string) (string, string, error) {
	log.Printf("Start connect Request to MBG %v for service %v", svcIdDest, mbgIP)

	conn, err := grpc.Dial(mbgIP, grpc.WithInsecure(), grpc.WithBlock())
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
	if r.GetMessage() == "Success" {
		log.Printf("Successfully Connected : Using Connection:Port - %s:%s", r.GetConnectType(), r.GetConnectDest())
		return r.GetConnectType(), r.GetConnectDest(), nil
	}

	log.Printf("[Cluster %v] Failed to Connect : %s port %s", state.GetId(), r.GetMessage(), r.GetConnectDest())
	if "Connection already setup!" == r.GetMessage() {
		return r.GetConnectType(), r.GetConnectDest(), fmt.Errorf(r.GetMessage())
	} else {
		return "", "", fmt.Errorf("Connect Request Failed")
	}
}
