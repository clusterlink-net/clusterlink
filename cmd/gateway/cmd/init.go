/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/gateway/state"

	"context"
	"log"
	"net"

	pb "github.ibm.com/mbg-agent/pkg/protocol"
	"google.golang.org/grpc"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "A init command set all parameter state of the gateway",
	Long: `A init command set all parameter state of the gateway-
			1) The MBG that the gateway is connected
			2) The IP of the gateway
			TBD now is done manually need to call some external `,
	Run: func(cmd *cobra.Command, args []string) {
		gwIP, _ := cmd.Flags().GetString("ip")
		gwName, _ := cmd.Flags().GetString("name")
		mbgIP, _ := cmd.Flags().GetString("mbgIP")

		state.SetState(mbgIP, gwIP, gwName)
		startServer(gwIP)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().String("name", "", "Gateway name")
	initCmd.Flags().String("ip", "", "Gateway IP")
	initCmd.Flags().String("mbgIP", "", "IP address of the MBG connected to the gateway")

}

const (
	port = ":50051"
)

func (s *ExposeServer) ExposeCmd(ctx context.Context, in *pb.ExposeRequest) (*pb.ExposeReply, error) {
	log.Printf("Received: %v", in.GetName())
	state.UpdateService(in.GetName(), in.GetId(), in.GetIp(), in.GetDomain(), in.GetPolicy())
	return &pb.ExposeReply{Message: "Done"}, nil
}

type ExposeServer struct {
	pb.UnimplementedExposeServer
}

func startServer(gwIP string) {
	log.Printf("Init gateway [%v] started", state.GetGwName())
	lis, err := net.Listen("tcp", gwIP+port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterExposeServer(s, &ExposeServer{})
	log.Printf("Control channel listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}
