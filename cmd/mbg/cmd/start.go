/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"net/http"
	"os"

	"github.com/go-chi/chi"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/mbg/state"

	md "github.ibm.com/mbg-agent/pkg/mbgDataplane"
	handler "github.ibm.com/mbg-agent/pkg/protocol/http/mbg"
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
		caFile, _ := cmd.Flags().GetString("rootCa")
		certificateFile, _ := cmd.Flags().GetString("certificate")
		keyFile, _ := cmd.Flags().GetString("key")
		dataplane, _ := cmd.Flags().GetString("dataplane")
		mtlsportLocal, _ := cmd.Flags().GetString("mtlsportLocal")
		mtlsport, _ := cmd.Flags().GetString("mtlsport")

		if ip == "" || id == "" || cport == "" {
			log.Println("Error: please insert all flag arguments for Mbg start command")
			os.Exit(1)
		}
		state.SetState(id, ip, cportLocal, cport, localDataPortRange, externalDataPortRange, caFile, certificateFile, keyFile, dataplane, mtlsportLocal, mtlsport)
		if dataplane == "mtls" {
			go md.StartMtlsServer(":"+mtlsportLocal, caFile, certificateFile, keyFile)
		}
		startHttpServer()
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().String("id", "", "Multi-cloud Border Gateway id")
	startCmd.Flags().String("ip", "", "Multi-cloud Border Gateway ip")
	startCmd.Flags().String("cportLocal", "50051", "Multi-cloud Border Gateway control local port inside the MBG")
	startCmd.Flags().String("cport", "", "Multi-cloud Border Gateway control external port for the MBG neighbors ")
	startCmd.Flags().String("mtlsportLocal", "8443", "Multi-cloud Border Gateway mtls local port inside the MBG")
	startCmd.Flags().String("mtlsport", "8443", "Multi-cloud Border Gateway mtls external port for the MBG neighbors ")
	startCmd.Flags().String("localDataPortRange", "5000", "Set the port range for data connection in the MBG")
	startCmd.Flags().String("externalDataPortRange", "30000", "Set the port range for exposing data connection (each expose port connect to localDataPort")
	startCmd.Flags().String("rootCa", "", "Path to the Root Certificate Auth File (.pem)")
	startCmd.Flags().String("certificate", "", "Path to the Certificate File (.pem)")
	startCmd.Flags().String("key", "", "Path to the Key File (.pem)")
	startCmd.Flags().String("dataplane", "tcp", "tcp/mtls based data-plane proxies")
}

/********************************** Server **********************************************************/
func startHttpServer() {
	log.Infof("MBG [%v] started", state.GetMyId())

	//Create a new router
	r := chi.NewRouter()
	r.Mount("/", handler.MbgHandler{}.Routes())

	//Use router to start the server
	mbgCPort := ":" + state.GetMyCport().Local
	log.Infof("Control channel listening at %v", mbgCPort)
	err := http.ListenAndServe(mbgCPort, r)
	if err != nil {
		log.Println(err)
	}

}
