package main

import (
	"flag"

	log "github.com/sirupsen/logrus"
	dp "github.ibm.com/mbg-agent/pkg/dataplane"
	"github.ibm.com/mbg-agent/pkg/dataplane/store"
	"github.ibm.com/mbg-agent/pkg/utils/logutils"
)

const (
	LogFileName = "dataplane.log"
)

func main() {
	var id, port, ca, cert, key, logLevel, dataplane string
	// Initialize the variable with the flag
	flag.StringVar(&id, "id", "", "Data plane gateway id")
	flag.StringVar(&port, "port", "443", "Default port data-plane start to listen")
	flag.StringVar(&ca, "certca", "", "Path to the Root Certificate Auth File (.pem)")
	flag.StringVar(&cert, "cert", "", "Path to the Certificate File (.pem)")
	flag.StringVar(&key, "key", "", "Path to the Key File (.pem)")
	flag.StringVar(&dataplane, "dataplane", "mtls", "tcp/mtls based data-plane proxies")
	flag.StringVar(&logLevel, "logLevel", "info", "Log level: debug, info, warning, error")
	// Parse command-line flags
	flag.Parse()
	//Set log file
	logutils.SetLog(logLevel, true, LogFileName)
	log.Infof("Dataplane main starterd")

	//Set Dataplane
	dp := dp.NewDataplane(&store.Store{Id: id, CertAuthority: ca, Cert: cert, Key: key, Dataplane: dataplane})
	dp.StartServer(port)
	log.Infof("Dataplane main process is finished")
}
