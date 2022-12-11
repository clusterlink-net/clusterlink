// /******* Commands **********/
// package grpcHandler

// import (
// 	"context"

// 	log "github.com/sirupsen/logrus"
// 	cmd "github.ibm.com/mbg-agent/cmd/cluster/cmd"
// 	"github.ibm.com/mbg-agent/cmd/cluster/state"
// 	pb "github.ibm.com/mbg-agent/pkg/protocol/grpc"
// )

// /******* Commands **********/
// //Expose
// func (s *ExposeServer) exposeCmd(ctx context.Context, in *pb.ExposeRequest) (*pb.ExposeReply, error) {
// 	log.Printf("Received: %v", in.GetId())
// 	state.UpdateState()
// 	state.AddService(in.GetId(), in.GetIp(), in.GetDomain())
// 	return &pb.ExposeReply{Message: "Done"}, nil
// }

// func (s *ConnectServer) connectCmd(ctx context.Context, in *pb.ConnectRequest) (*pb.ConnectReply, error) {
// 	log.Printf("Received Connect request from service: %v to service: %v", in.GetId(), in.GetIdDest())
// 	state.UpdateState()
// 	svc := state.GetService(in.GetId())
// 	destSvc := state.GetService(in.GetIdDest())
// 	name := state.GetId() + " egress: " + in.GetIdDest()
// 	cmd.ConnectClient(svc.Service.Id, destSvc.Service.Id, svc.Service.Ip, destSvc.Service.Ip, name)

// 	return &pb.ConnectReply{Message: "Connecting the services"}, nil
// }
