// /******* Commands **********/
// package grpcHandler

// import (
// 	"context"

// 	log "github.com/sirupsen/logrus"
// 	cmd "github.ibm.com/mbg-agent/cmd/mbg/cmd"
// 	"github.ibm.com/mbg-agent/cmd/mbg/state"
// 	pb "github.ibm.com/mbg-agent/pkg/protocol/grpc"
// )

// //Expose
// type ExposeServer struct {
// 	pb.UnimplementedExposeServer
// }

// func (s *ExposeServer) ExposeCmd(ctx context.Context, in *pb.ExposeRequest) (*pb.ExposeReply, error) {
// 	log.Infof("Received: %v", in.GetId())
// 	state.UpdateState()
// 	if in.GetDomain() == "Internal" {
// 		state.AddLocalService(in.GetId(), in.GetIp(), in.GetDomain())
// 		cmd.ExposeToMbg(in.GetId())
// 	} else { //Got the service from MBG so expose to local Cluster
// 		state.AddRemoteService(in.GetId(), in.GetIp(), in.GetDomain(), in.GetMbgID())
// 		cmd.ExposeToCluster(in.GetId())
// 	}
// 	return &pb.ExposeReply{Message: "Done"}, nil
// }

// //Hello
// type HelloServer struct {
// 	pb.UnimplementedHelloServer
// }

// func (s *HelloServer) HelloCmd(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
// 	log.Infof("Received Hello from MBG ip: %v", in.GetIp())
// 	state.UpdateState()
// 	state.AddMbgNbr(in.GetId(), in.GetIp(), in.GetCport())

// 	return &pb.HelloReply{Message: "MBG: " + state.GetMyIp() + " get hello message"}, nil
// }

// //Connect
// type ConnectServer struct {
// 	pb.UnimplementedConnectServer
// }

// func (s *ConnectServer) ConnectCmd(ctx context.Context, in *pb.ConnectRequest) (*pb.ConnectReply, error) {
// 	state.UpdateState()
// 	//svc := state.GetService(in.GetID())
// 	connectionID := in.GetId() + ":" + in.GetIdDest()
// 	if state.IsServiceLocal(in.GetIdDest()) {
// 		log.Infof("[MBG %v] Received Incoming Connect request from service: %v to service: %v", state.GetMyId(), in.GetId(), in.GetIdDest())
// 		// Get a free local/external port
// 		// Send the external port as reply to the MBG
// 		localSvc := state.GetLocalService(in.GetIdDest())

// 		myConnectionPorts, err := state.GetFreePorts(connectionID)
// 		if err != nil {
// 			log.Infof("[MBG %v] Error getting free ports %s", state.GetMyId(), err.Error())
// 			return &pb.ConnectReply{Message: err.Error(), ConnectType: "tcp", ConnectDest: myConnectionPorts.External}, nil
// 		}
// 		log.Infof("[MBG %v] Using ConnectionPorts : %v", state.GetMyId(), myConnectionPorts)
// 		clusterIpPort := localSvc.Service.Ip
// 		// TODO Need to check Policy before accepting connections
// 		//ApplyGlobalPolicies
// 		//ApplyServicePolicies
// 		go cmd.ConnectService(myConnectionPorts.Local, clusterIpPort, in.GetPolicy(), connectionID)
// 		log.Infof("[MBG %v] Sending Connect reply to Connection(%v) to use Dest:%v", state.GetMyId(), connectionID, myConnectionPorts.External)
// 		return &pb.ConnectReply{Message: "Success", ConnectType: "tcp", ConnectDest: myConnectionPorts.External}, nil

// 	} else { //For Remote service
// 		log.Infof("[MBG %v] Received Outgoing Connect request from service: %v to service: %v", state.GetMyId(), in.GetId(), in.GetIdDest())
// 		destSvc := state.GetRemoteService(in.GetIdDest())
// 		mbgIP := state.GetServiceMbgIp(destSvc.Service.Ip)
// 		//Send connection request to other MBG
// 		connectType, connectDest, err := cmd.SendConnectReq(in.GetId(), in.GetIdDest(), in.GetPolicy(), mbgIP)
// 		if err != nil && err.Error() != "Connection already setup!" {
// 			log.Infof("[MBG %v] Send connect failure to Cluster =%v ", state.GetMyId(), err.Error())
// 			return &pb.ConnectReply{Message: "Failure", ConnectType: "tcp", ConnectDest: connectDest}, nil
// 		}
// 		log.Infof("[MBG %v] Using %v:%v to connect IP-%v", state.GetMyId(), connectType, connectDest, destSvc.Service.Ip)

// 		//Randomize listen ports for return
// 		myConnectionPorts, err := state.GetFreePorts(connectionID)
// 		if err != nil {
// 			log.Infof("[MBG %v] Error getting free ports %s", state.GetMyId(), err.Error())
// 			return &pb.ConnectReply{Message: err.Error(), ConnectType: "tcp", ConnectDest: myConnectionPorts.External}, nil
// 		}
// 		log.Infof("[MBG %v] Using ConnectionPorts : %v", state.GetMyId(), myConnectionPorts)
// 		//Create data connection
// 		destIp := destSvc.Service.Ip + ":" + connectDest
// 		go cmd.ConnectService(myConnectionPorts.Local, destIp, in.GetPolicy(), connectionID)
// 		//Return a reply with to connect request
// 		log.Infof("[MBG %v] Sending Connect reply to Connection(%v) to use Dest:%v", state.GetMyId(), connectionID, myConnectionPorts.External)
// 		return &pb.ConnectReply{Message: "Success", ConnectType: "tcp", ConnectDest: myConnectionPorts.External}, nil

// 	}
// }

// //Disconnect
// type DisconnectServer struct {
// 	pb.UnimplementedDisconnectServer
// }

// func (s *DisconnectServer) DisconnectCmd(ctx context.Context, in *pb.DisconnectRequest) (*pb.DisconnectReply, error) {
// 	state.UpdateState()
// 	connectionID := in.GetId() + ":" + in.GetIdDest()
// 	if state.IsServiceLocal(in.GetIdDest()) {
// 		state.FreeUpPorts(connectionID)
// 		// Need to Kill the corresponding process
// 	} else {
// 		// Need to just Kill the corresponding process
// 	}
// 	return &pb.DisconnectReply{Message: "Success"}, nil
// }
