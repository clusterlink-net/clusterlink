/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/mbgctl/state"
	"github.ibm.com/mbg-agent/pkg/clusterProxy"
)

// connectCmd represents the connect command
var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "connect flow connection to the closest MBG",
	Long:  `connect flow connection to the closest MBG`,
	Run: func(cmd *cobra.Command, args []string) {
		svcId, _ := cmd.Flags().GetString("serviceId")
		svcIp, _ := cmd.Flags().GetString("serviceIp")
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
		if svcIp != "" {
			srcIp = svcIp
		}
		destIp := destSvc.Service.Ip

		log.Infof("[mbgctl %v] Using %v:%v to connect IP-%v", state.GetId(), "TCP", destIp, destSvc.Service.Ip)
		connectClient(svc.Service.Id, destSvc.Service.Id, srcIp, destIp, name)

	},
}

func init() {
	rootCmd.AddCommand(connectCmd)
	connectCmd.Flags().String("serviceId", "", "Service Id for connection")
	connectCmd.Flags().String("serviceIp", "", "Service Ip for connection to listen")
	connectCmd.Flags().String("serviceIdDest", "", "Destination service id for connection")
	connectCmd.Flags().String("policy", "Forward", "Connection policy")
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
