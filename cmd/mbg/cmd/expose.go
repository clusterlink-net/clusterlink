/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"context"
	"log"
	"time"

	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
	pb "github.ibm.com/mbg-agent/pkg/protocol"
	service "github.ibm.com/mbg-agent/pkg/serviceMap"
	"google.golang.org/grpc"
)

// exposeCmd represents the expose command
var exposeCmd = &cobra.Command{
	Use:   "expose",
	Short: "Expose command send an expose message to Multi-cloud Border Gateway",
	Long:  `Expose command send an expose message to Multi-cloud Border Gateway`,
	Run: func(cmd *cobra.Command, args []string) {
		serviceId, _ := cmd.Flags().GetString("serviceId")
		state.UpdateState()
		ExposeToMbg(serviceId)

	},
}

func init() {
	rootCmd.AddCommand(exposeCmd)
	exposeCmd.Flags().String("serviceId", "", "Service Id for exposing")

}

func ExposeToMbg(serviceId string) {
	MbgArr := state.GetMbgArr()
	myIp := state.GetMyIp()

	s := state.GetLocalService(serviceId)
	svcExp := s.Service
	svcExp.Ip = myIp + ":" + s.ExposeDataPort //update port to connect data
	svcExp.Domain = "Remote"
	for _, m := range MbgArr {
		destIp := m.Ip + ":" + m.Cport
		expose(svcExp, destIp, "MBG")
	}
}

func ExposeToCluster(serviceId string) {
	clusterArr := state.GetLocalClusterArr()
	myIp := state.GetMyIp()
	s := state.GetRemoteService(serviceId)
	svcExp := s.Service
	svcExp.Ip = myIp + ":" + s.ExposeDataPort //update port to connect data
	svcExp.Domain = "Remote"

	for _, g := range clusterArr {
		destIp := g.Ip
		expose(svcExp, destIp, "Gateway")
	}
}

func expose(svcExp service.Service, destIp, cType string) {
	log.Printf("Start expose %v to %v with IP address %v", svcExp.Id, cType, destIp)

	conn, err := grpc.Dial(destIp, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewExposeClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.ExposeCmd(ctx, &pb.ExposeRequest{Id: svcExp.Id, Ip: svcExp.Ip, Domain: svcExp.Domain, MbgID: state.GetMyId()}) //TBD- No need to expose the domains
	if err != nil {
		log.Fatalf("could not create user: %v", err)
	}
	log.Printf(`Response message:  %s`, r.GetMessage())

}
