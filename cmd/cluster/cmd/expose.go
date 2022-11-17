/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/cluster/state"

	"context"
	"log"
	"time"

	pb "github.ibm.com/mbg-agent/pkg/protocol"
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

		mbgIP := state.GetMbgIP()
		expose(serviceId, mbgIP)

	},
}

func init() {
	rootCmd.AddCommand(exposeCmd)
	exposeCmd.Flags().String("serviceId", "", "Service Id for exposing")

}

func expose(serviceId, mbgIP string) {
	log.Printf("Start expose %v to MBG with IP address %v", serviceId, mbgIP)
	s := state.GetService(serviceId)
	svcExp := s.Service

	log.Printf("Service %v", s)

	conn, err := grpc.Dial(mbgIP, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewExposeClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.ExposeCmd(ctx, &pb.ExposeRequest{Id: svcExp.Id, Ip: svcExp.Ip, Domain: svcExp.Domain})
	if err != nil {
		log.Fatalf("could not create user: %v", err)
	}
	log.Printf(`Response message:  %s`, r.GetMessage())

}
