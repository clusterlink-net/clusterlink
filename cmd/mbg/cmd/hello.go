/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
	pb "github.ibm.com/mbg-agent/pkg/protocol"
	"google.golang.org/grpc"
)

// helloCmd represents the hello command
var helloCmd = &cobra.Command{
	Use:   "hello",
	Short: "Hello command send hello message to all MBGs in thr MBG neighbor list",
	Long:  `Hello command send hello message to all MBGs in thr MBG neighbor list.`,
	Run: func(cmd *cobra.Command, args []string) {
		state.UpdateState()
		log.Infof("Hello command called")
		MbgArr := state.GetMbgArr()
		MyInfo := state.GetMyInfo()
		for _, m := range MbgArr {
			log.Info(m)
			sendHello(m, MyInfo)
		}
		log.Infof("Finish sending Hello to all Mbgs")
	},
}

func init() {
	rootCmd.AddCommand(helloCmd)
}

func sendHello(m, MyInfo state.MbgInfo) {
	log.Infof("Start Hello message to MBG with IP address %v", m.Ip)

	address := m.Ip + ":" + m.Cport.External
	log.Infof("Start Hello message to MBG with IP address %v", address)

	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewHelloClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.HelloCmd(ctx, &pb.HelloRequest{Id: MyInfo.Id, Ip: MyInfo.Ip, Cport: MyInfo.Cport.External})
	if err != nil {
		log.Fatalf("could not create user: %v", err)
	}
	log.Infof(`Response message for Hello:  %s`, r.GetMessage())

}
