/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
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
