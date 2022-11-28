/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"context"
	"net"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
	"google.golang.org/grpc"

	pb "github.ibm.com/mbg-agent/pkg/protocol"
)

/// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "A start command set all parameter state of the Multi-cloud Border Gateway",
	Long: `A start command set all parameter state of the MBg-
			The  id, IP cport(Cntrol port for grpc) and localDataPortRange,externalDataPortRange
			TBD now is done manually need to call some external `,
	Run: func(cmd *cobra.Command, args []string) {
		ip, _ := cmd.Flags().GetString("ip")
		id, _ := cmd.Flags().GetString("id")
		cportLocal, _ := cmd.Flags().GetString("cportLocal")
		cport, _ := cmd.Flags().GetString("cport")
		localDataPortRange, _ := cmd.Flags().GetString("localDataPortRange")
		externalDataPortRange, _ := cmd.Flags().GetString("externalDataPortRange")

		if ip == "" || id == "" || cport == "" {
			log.Println("Error: please insert all flag arguments for Mbg start command")
			os.Exit(1)
		}
		state.SetState(id, ip, cportLocal, cport, localDataPortRange, externalDataPortRange)
		startServer()
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().String("id", "", "Multi-cloud Border Gateway id")
	startCmd.Flags().String("ip", "", "Multi-cloud Border Gateway ip")
	startCmd.Flags().String("cportLocal", "50051", "Multi-cloud Border Gateway control local port inside the MBG")
	startCmd.Flags().String("cport", "", "Multi-cloud Border Gateway control external port for the MBG neighbors ")
	startCmd.Flags().String("localDataPortRange", "5000", "Set the port range for data connection in the MBG")
	startCmd.Flags().String("externalDataPortRange", "30000", "Set the port range for exposing data connection (each expose port connect to localDataPort")
}

/******* Commands **********/
//Expose
type ExposeServer struct {
	pb.UnimplementedExposeServer
}

func (s *ExposeServer) ExposeCmd(ctx context.Context, in *pb.ExposeRequest) (*pb.ExposeReply, error) {
	log.Infof("Received: %v", in.GetId())
	state.UpdateState()
	if in.GetDomain() == "Internal" {
		state.AddLocalService(in.GetId(), in.GetIp(), in.GetDomain())
		ExposeToMbg(in.GetId())
	} else { //Got the service from MBG so expose to local Cluster
		state.AddRemoteService(in.GetId(), in.GetIp(), in.GetDomain(), in.GetMbgID())
		ExposeToCluster(in.GetId())
	}
	return &pb.ExposeReply{Message: "Done"}, nil
}

//Hello
type HelloServer struct {
	pb.UnimplementedHelloServer
}

func (s *HelloServer) HelloCmd(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Infof("Received Hello from MBG ip: %v", in.GetIp())
	state.UpdateState()
	state.AddMbgNbr(in.GetId(), in.GetIp(), in.GetCport())

	return &pb.HelloReply{Message: "MBG: " + state.GetMyIp() + " get hello message"}, nil
}

//Connect
type ConnectServer struct {
	pb.UnimplementedConnectServer
}

func (s *ConnectServer) ConnectCmd(ctx context.Context, in *pb.ConnectRequest) (*pb.ConnectReply, error) {
	state.UpdateState()
	//svc := state.GetService(in.GetID())
	connectionID := in.GetId() + ":" + in.GetIdDest()
	if state.IsServiceLocal(in.GetIdDest()) {
		log.Infof("[MBG %v] Received Incoming Connect request from service: %v to service: %v", state.GetMyId(), in.GetId(), in.GetIdDest())
		// Get a free local/external port
		// Send the external port as reply to the MBG
		localSvc := state.GetLocalService(in.GetIdDest())

		myConnectionPorts, err := state.GetFreePorts(connectionID)
		if err != nil {
			log.Infof("[MBG %v] Error getting free ports %s", state.GetMyId(), err.Error())
			return &pb.ConnectReply{Message: err.Error()}, nil
		}
		log.Infof("[MBG %v] Using ConnectionPorts : %v", state.GetMyId(), myConnectionPorts)
		clusterIpPort := localSvc.Service.Ip

		go ConnectService(myConnectionPorts.Local, clusterIpPort, in.GetPolicy())

		log.Infof("[MBG %v] Sending Connect reply to Connection(%v) to use Dest:%v", state.GetMyId(), connectionID, myConnectionPorts.External)
		return &pb.ConnectReply{Message: "Success", ConnectType: "tcp", ConnectDest: myConnectionPorts.External}, nil

	} else { //For Remote service
		log.Infof("[MBG %v] Received Outgoing Connect request from service: %v to service: %v", state.GetMyId(), in.GetId(), in.GetIdDest())
		destSvc := state.GetRemoteService(in.GetIdDest())
		mbgIP := state.GetServiceMbgIp(destSvc.Service.Ip)
		//Send connection request to other MBG
		connectType, connectDest, err := SendConnectReq(in.GetId(), in.GetIdDest(), in.GetPolicy(), mbgIP)
		if err != nil {
			log.Infof("[MBG %v] Send connect failure to Cluster", state.GetMyId())
			return &pb.ConnectReply{Message: "Failure"}, nil
		}
		log.Infof("[MBG %v] Using %v:%v to connect IP-%v", state.GetMyId(), connectType, connectDest, destSvc.Service.Ip)

		//Randomize listen ports for return
		myConnectionPorts, err := state.GetFreePorts(connectionID)
		if err != nil {
			log.Infof("[MBG %v] Error getting free ports %s", state.GetMyId(), err.Error())
			return &pb.ConnectReply{Message: err.Error()}, nil
		}
		log.Infof("[MBG %v] Using ConnectionPorts : %v", state.GetMyId(), myConnectionPorts)
		//Create data connection
		destIp := destSvc.Service.Ip + ":" + connectDest
		go ConnectService(myConnectionPorts.Local, destIp, in.GetPolicy())
		//Return a reply with to connect request
		log.Infof("[MBG %v] Sending Connect reply to Connection(%v) to use Dest:%v", state.GetMyId(), connectionID, myConnectionPorts.External)
		return &pb.ConnectReply{Message: "Success", ConnectType: "tcp", ConnectDest: myConnectionPorts.External}, nil

	}
}

//Disconnect
type DisconnectServer struct {
	pb.UnimplementedDisconnectServer
}

func (s *DisconnectServer) DisconnectCmd(ctx context.Context, in *pb.DisconnectRequest) (*pb.DisconnectReply, error) {
	state.UpdateState()
	connectionID := in.GetId() + ":" + in.GetIdDest()
	if state.IsServiceLocal(in.GetIdDest()) {
		state.FreeUpPorts(connectionID)
		// Need to Kill the corresponding process
	} else {
		// Need to just Kill the corresponding process
	}
	return &pb.DisconnectReply{Message: "Success"}, nil
}

/********************************** Server **********************************************************/
func startServer() {
	log.Infof("MBG [%v] started", state.GetMyId())
	mbgCPort := ":" + state.GetMyCport().Local //TBD - not supporting using several MBGs in same node
	lis, err := net.Listen("tcp", mbgCPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()

	pb.RegisterExposeServer(s, &ExposeServer{})
	pb.RegisterConnectServer(s, &ConnectServer{})
	pb.RegisterDisconnectServer(s, &DisconnectServer{})
	pb.RegisterHelloServer(s, &HelloServer{})

	log.Infof("Control channel listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}
