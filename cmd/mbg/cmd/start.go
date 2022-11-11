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

/// startCmd represents the init command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "A start command set all parameter state of the Multi-cloud Border Gateway",
	Long: `A start command set all parameter state of the gateway-
			1) The MBG that the gateway is connected
			2) The IP of the gateway
			TBD now is done manually need to call some external `,
	Run: func(cmd *cobra.Command, args []string) {
		ip, _ := cmd.Flags().GetString("ip")
		id, _ := cmd.Flags().GetString("id")

		if ip == "" || id == "" {
			log.Println("Error: please insert all flag arguments for mbg start command")
			os.Exit(1)
		}
		state.SetState(id, ip)
		startServer(ip)
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().String("id", "", "Multi-cloud Border Gateway id")
	startCmd.Flags().String("ip", "", "Multi-cloud Border Gateway ip")

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
	log.Printf("Received Expose of %v at %v ",in.GetId(), in.GetIp())

	// TODO : Handle logic of receiving expose from other MBG
	state.UpdateLocalService(in.GetId(), in.GetIp(), in.GetDomain(), in.GetPolicy())
	ExposeToNeighborMbgs(in.GetId(), in.GetMbgID())
	//ExposeToLocalGw()
	return &pb.ExposeReply{Message: "Done"}, nil
}

func ExposeToNeighborMbgs(serviceId, sourceMbgId string) {
	state.UpdateState()
	MbgArr := state.GetMbgArr()
	myIp := state.GetMyIp()
	for _, m := range MbgArr {
		if m.Id != sourceMbgId { //Do not expose back to the source MBG
			ExposeToMBGs(serviceId, m, myIp)
		}
	}
}

//Hello
type HelloServer struct {
	pb.UnimplementedHelloServer
}

func (s *HelloServer) HelloCmd(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Printf("Received Hello from MBG ip: %v", in.GetIp())
	state.UpdateMbgArr(in.GetId(), in.GetIp())

	return &pb.HelloReply{Message: "MBG: " + state.GetMyIp() + " get hello message"}, nil
}

//Connect
type ConnectServer struct {
	pb.UnimplementedConnectServer
}

func (s *ConnectServer) connectCmd(ctx context.Context, in *pb.ConnectRequest) (*pb.ConnectReply, error) {
	log.Printf("Received Connect request from service: %v to service: %v", in.GetSourceId(), in.GetDestId())

	return &pb.ConnectReply{Message: "Connecting the services"}, nil
}

/******* Server **********/
func startServer(ip string) {
	log.Printf("MBG [%v] started", state.GetId())
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
