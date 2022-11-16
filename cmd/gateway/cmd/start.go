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

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "A start command set all parameter state of the gateway",
	Long: `A start command set all parameter state of the gateway-
			1) The MBG that the gateway is connected
			2) The IP of the gateway
			TBD now is done manually need to call some external `,
	Run: func(cmd *cobra.Command, args []string) {
		gwIP, _ := cmd.Flags().GetString("ip")
		gwId, _ := cmd.Flags().GetString("id")
		mbgIP, _ := cmd.Flags().GetString("mbgIP")
		cport, _ := cmd.Flags().GetString("cport")

		state.SetState(mbgIP, gwIP, gwId, cport)
		startServer(gwIP)
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().String("id", "", "Gateway Id")
	startCmd.Flags().String("ip", "", "Gateway IP")
	startCmd.Flags().String("mbgIP", "", "IP address of the MBG connected to the gateway")
	startCmd.Flags().String("cport", "", "Gateway control port")
}

const (
	serverPort = ":50051"
)

/******* Commands **********/
//Expose
func (s *ExposeServer) ExposeCmd(ctx context.Context, in *pb.ExposeRequest) (*pb.ExposeReply, error) {
	log.Printf("Received: %v", in.GetId())
	state.UpdateState()
	state.AddService(in.GetId(), in.GetIp(), in.GetDomain())
	return &pb.ExposeReply{Message: "Done"}, nil
}

type ExposeServer struct {
	pb.UnimplementedExposeServer
}

//Connect
type ConnectServer struct {
	pb.UnimplementedConnectServer
}

func (s *ConnectServer) connectCmd(ctx context.Context, in *pb.ConnectRequest) (*pb.ConnectReply, error) {
	log.Printf("Received Connect request from service: %v to service: %v", in.GetId(), in.GetIdDest())
	state.UpdateState()
	svc := state.GetService(in.GetId())
	destSvc := state.GetService(in.GetIdDest())

	connectClient(svc.Service.Ip, destSvc.Service.Ip)

	return &pb.ConnectReply{Message: "Connecting the services"}, nil
}

/********************************** Server **********************************************************/
func startServer(gwIP string) {
	log.Printf("start gateway [%v] started", state.GetGwId())
	lis, err := net.Listen("tcp", serverPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterExposeServer(s, &ExposeServer{})
	pb.RegisterConnectServer(s, &ConnectServer{})
	log.Printf("Control channel listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}
