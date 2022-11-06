/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"context"
	"log"
	"net"
	"os"

	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
	"google.golang.org/grpc"

	pb "github.ibm.com/mbg-agent/pkg/protocol"
)

/// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "A init command set all parameter state of the Multi-cloud Border Gateway",
	Long: `A init command set all parameter state of the gateway-
			1) The MBG that the gateway is connected
			2) The IP of the gateway
			TBD now is done manually need to call some external `,
	Run: func(cmd *cobra.Command, args []string) {
		ip, _ := cmd.Flags().GetString("ip")
		name, _ := cmd.Flags().GetString("name")
		id, _ := cmd.Flags().GetString("id")

		if ip == "" || name == "" || id == "" {
			log.Println("Error: please insert all flag arguments for Mbg Init command")
			os.Exit(1)
		}
		state.SetState(name, id, ip)
		startServer(ip)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().String("name", "", "Multi-cloud Border Gateway name")
	initCmd.Flags().String("ip", "", "Multi-cloud Border Gateway ip")
	initCmd.Flags().String("id", "", "Multi-cloud Border Gateway id")

}

const (
	port = ":50051"
)

/******* Commands **********/
//Expose
type ExposeServer struct {
	pb.UnimplementedExposeServer
}

func (s *ExposeServer) ExposeCmd(ctx context.Context, in *pb.ExposeRequest) (*pb.ExposeReply, error) {
	log.Printf("Received: %v", in.GetName())
	state.UpdateService(in.GetName(), in.GetId(), in.GetIp(), in.GetDomain(), in.GetPolicy())
	ExposeToMbg()
	ExposeToLocalGw()
	return &pb.ExposeReply{Message: "Done"}, nil
}

func ExposeToMbg() {
}

func ExposeToLocalGw() {
}

//Hello
type HelloServer struct {
	pb.UnimplementedHelloServer
}

func (s *HelloServer) HelloCmd(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Printf("Received Hello from MBG ip: %v", in.GetIp())
	state.UpdateMbgArr(in.GetName(), in.GetId(), in.GetIp())

	return &pb.HelloReply{Message: "MBG: " + state.GetMyIp() + " get hello message"}, nil
}

//Connect
type ConnectServer struct {
	pb.UnimplementedConnectServer
}

func (s *ConnectServer) connectCmd(ctx context.Context, in *pb.ConnectRequest) (*pb.ConnectReply, error) {
	log.Printf("Received Connect request from service: %v to service: %v", in.GetName(), in.GetNameDest())

	return &pb.ConnectReply{Message: "Connect ing the services"}, nil
}

/******* Server **********/
func startServer(ip string) {
	log.Printf("Init gateway [%v] started", state.GetMyName())
	lis, err := net.Listen("tcp", ip+port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()

	pb.RegisterExposeServer(s, &ExposeServer{})
	pb.RegisterConnectServer(s, &ConnectServer{})
	pb.RegisterHelloServer(s, &HelloServer{})

	log.Printf("Control channel listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}
