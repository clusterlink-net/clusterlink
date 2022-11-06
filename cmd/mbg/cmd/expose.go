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
	"google.golang.org/grpc"
)

// exposeCmd represents the expose command
var exposeCmd = &cobra.Command{
	Use:   "expose",
	Short: "Expose command send an expose message to Multi-cloud Border Gateway",
	Long:  `Expose command send an expose message to Multi-cloud Border Gateway`,
	Run: func(cmd *cobra.Command, args []string) {
		serviceName, _ := cmd.Flags().GetString("serviceName")
		serviceId, _ := cmd.Flags().GetString("serviceId")
		state.UpdateState()
		MbgArr := state.GetMbgArr()
		myIp := state.GetMyIp()
		for _, m := range MbgArr {
			expose(serviceName, serviceId, m, myIp)

		}

	},
}

func init() {
	rootCmd.AddCommand(exposeCmd)
	exposeCmd.Flags().String("serviceName", "", "Service name for exposing")
	exposeCmd.Flags().String("serviceId", "", "Service Id for exposing")

}

func expose(serviceName, serviceId string, m state.MbgInfo, myIp string) {
	log.Printf("Start expose %v to MBG with IP address %v", serviceName, m.Ip)
	s := state.GetService(serviceName, serviceId)
	svcExp := s.Service
	svcExp.Ip = myIp + ":" + s.LocalPort //update port to connect data

	log.Printf("Service %v", s)
	address := m.Ip + ":50051"

	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewExposeClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.ExposeCmd(ctx, &pb.ExposeRequest{Name: svcExp.Name, Id: svcExp.Id, Ip: svcExp.Ip, Domain: svcExp.Domain, Policy: svcExp.Policy})
	if err != nil {
		log.Fatalf("could not create user: %v", err)
	}
	log.Printf(`Response message:  %s`, r.GetMessage())

}
