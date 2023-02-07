/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/mbg/state"

	"github.ibm.com/mbg-agent/pkg/policyEngine"
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
		startPolicyEngine, _ := cmd.Flags().GetBool("startPolicyEngine")
		policyEngineIp, _ := cmd.Flags().GetString("policyEngineIp")

		if ip == "" || id == "" || cport == "" {
			fmt.Println("Error: please insert all flag arguments for Mbg start command")
			os.Exit(1)
		}
		state.SetState(id, ip, cportLocal, cport, localDataPortRange, externalDataPortRange, caFile, certificateFile, keyFile, dataplane)
		if startPolicyEngine {
			state.GetEventManager().AssignPolicyDispatcher("http://" + policyEngineIp + "/policy")
			state.SaveState()
			serverPolicyIp := policyEngineIp
			if strings.Contains(policyEngineIp, "localhost") {
				serverPolicyIp = ":" + strings.Split(policyEngineIp, ":")[1]
			}
			go policyEngine.StartPolicyDispatcher(state.GetChiRouter(), serverPolicyIp)
		}

		if dataplane == "mtls" {
			StartMtlsServer(":"+cportLocal, caFile, certificateFile, keyFile)
		} else {
			startHttpServer(":" + cportLocal)
		}
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().String("id", "", "Multi-cloud Border Gateway id")
	startCmd.Flags().String("ip", "", "Multi-cloud Border Gateway ip")
	startCmd.Flags().String("cportLocal", "8443", "Multi-cloud Border Gateway control local port inside the MBG")
	startCmd.Flags().String("cport", "8443", "Multi-cloud Border Gateway control external port for the MBG neighbors ")
	startCmd.Flags().String("localDataPortRange", "5000", "Set the port range for data connection in the MBG")
	startCmd.Flags().String("externalDataPortRange", "30000", "Set the port range for exposing data connection (each expose port connect to localDataPort")
	startCmd.Flags().String("rootCa", "", "Path to the Root Certificate Auth File (.pem)")
	startCmd.Flags().String("certificate", "", "Path to the Certificate File (.pem)")
	startCmd.Flags().String("key", "", "Path to the Key File (.pem)")
	startCmd.Flags().String("dataplane", "tcp", "tcp/mtls based data-plane proxies")
	startCmd.Flags().Bool("startPolicyEngine", false, "Start policy engine in port")
	startCmd.Flags().String("policyEngineIp", "localhost:9990", "Set the policy engine ip")
}

/********************************** Server **********************************************************/
func startHttpServer(ip string) {
	log.Infof("MBG [%v] started", state.GetMyId())

	//Set chi router
	r := state.GetChiRouter()
	r.Mount("/", handler.MbgHandler{}.Routes())

	//Use router to start the server
	log.Infof("Starting HTTP server, listening to: %v", ip)
	err := http.ListenAndServe(ip, r)
	if err != nil {
		log.Println(err)
	}

}

func StartMtlsServer(ip, rootCA, certificate, key string) {
	// Create the TLS Config with the CA pool and enable Client certificate validation
	caCert, err := ioutil.ReadFile(rootCA)
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		ClientCAs:  caCertPool,
		ClientAuth: tls.RequireAndVerifyClientCert,
	}
	//Set chi router
	r := state.GetChiRouter()
	r.Mount("/", handler.MbgHandler{}.Routes())

	// Create a Server instance to listen on port 8443 with the TLS config
	server := &http.Server{
		Addr:      ip,
		TLSConfig: tlsConfig,
		Handler:   r,
	}
	log.Infof("Starting mTLS Server for MBG Dataplane/Controlplane")

	// Listen to HTTPS connections with the server certificate and wait
	log.Fatal(server.ListenAndServeTLS(certificate, key))
}
