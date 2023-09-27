package app

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/grpc/credentials"

	cpapi "github.com/clusterlink-org/clusterlink/pkg/controlplane/api"
	"github.com/clusterlink-org/clusterlink/pkg/dataplane/api"
	dpclient "github.com/clusterlink-org/clusterlink/pkg/dataplane/client"
	dpserver "github.com/clusterlink-org/clusterlink/pkg/dataplane/server"
	"github.com/clusterlink-org/clusterlink/pkg/util"
)

const (
	// logLevel is the default log level.
	logLevel = "warn"

	// CAFile is the path to the certificate authority file.
	CAFile = "/etc/ssl/certs/clink_ca.pem"
	// CertificateFile is the path to the certificate file.
	CertificateFile = "/etc/ssl/certs/clink-dataplane.pem"
	// KeyFile is the path to the private-key file.
	KeyFile = "/etc/ssl/private/clink-dataplane.pem"

	// dataplaneServerAddress is the address of the dataplane HTTP server for accepting ingress dataplane connections.
	dataplaneServerAddress = "127.0.0.1:8443"
)

// Options contains everything necessary to create and run a dataplane.
type Options struct {
	// ControlplaneHost is the IP/hostname of the controlplane.
	ControlplaneHost string
	// LogFile is the path to file where logs will be written.
	LogFile string
	// LogLevel is the log level.
	LogLevel string
}

// AddFlags adds flags to fs and binds them to options.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.ControlplaneHost, "controlplane-host", "",
		"The controlplane IP/hostname.")
	fs.StringVar(&o.LogFile, "log-file", "",
		"Path to a file where logs will be written. If not specified, logs will be printed to stderr.")
	fs.StringVar(&o.LogLevel, "log-level", logLevel,
		"The log level. One of fatal, error, warn, info, debug.")
}

// RequiredFlags are the names of flags that must be explicitly specified.
func (o *Options) RequiredFlags() []string {
	return []string{"controlplane-host"}
}

func runDataplane(controlPlaneHost string) error {
	parsedCertData, err := util.ParseTLSFiles(CAFile, CertificateFile, KeyFile)
	if err != nil {
		return fmt.Errorf("unable to parse TLS files")
	}
	dnsNames := parsedCertData.DNSNames()
	if len(dnsNames) != 1 {
		return fmt.Errorf("expected certificate to contain a single DNS name, but got %d", len(dnsNames))
	}

	peerName, err := api.StripServerPrefix(dnsNames[0])
	if err != nil {
		return err
	}

	controlplaneTarget := controlPlaneHost + ":" + strconv.Itoa(cpapi.ListenPort)
	dataplaneID := uuid.New().String() // TODO use parsedCertData.CommonName(), when available

	log.Infof("Starting dataplane, peerName : %s, dataplaneID : %s", peerName, dataplaneID)
	log.Debugf("Dialing to GRPC port(%s:%d) : %s", controlPlaneHost, cpapi.ListenPort, controlplaneTarget)

	dataplane := dpserver.NewDataplane(dataplaneID, controlplaneTarget, peerName, parsedCertData)
	go func() {
		err = dataplane.StartDataplaneServer(dataplaneServerAddress)
		log.Error("Failed to start dataplane server", err)
	}()

	go func() {
		err = dataplane.StartSNIServer(dataplaneServerAddress)
		log.Error("Failed to start dataplane server", err)
	}()
	// Start xDS client, if it fails to start we keep retrying to connect to the controlplane host
	for {
		err = dpclient.StartxDSClient(dataplane, controlplaneTarget, credentials.NewTLS(parsedCertData.ClientConfig(cpapi.GRPCServerName(peerName))))
		if err != nil {
			log.Errorf("Failed to start xDS client, retrying : %v", err)
		}
		time.Sleep(10 * time.Second)
	}
}

// Run the dataplane.
func (o *Options) Run() error {
	// set log file
	if o.LogFile != "" {
		f, err := os.OpenFile(o.LogFile, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			return fmt.Errorf("unable to open log file: %v", err)
		}

		defer func() {
			if err := f.Close(); err != nil {
				log.Errorf("Cannot close log file: %v", err)
			}
		}()

		log.SetOutput(f)
	}

	// set log level
	logLevel, err := log.ParseLevel(o.LogLevel)
	if err != nil {
		return fmt.Errorf("unable to set log level: %v", err)
	}
	log.SetLevel(logLevel)
	err = runDataplane(o.ControlplaneHost)
	if err != nil {
		log.Errorf("Failed to run dataplane: %v", err)
	}
	return nil
}

// NewDataplaneCommand creates a *cobra.Command object with default parameters.
func NewDataplaneCommand() *cobra.Command {
	opts := &Options{}

	cmd := &cobra.Command{
		Use:          "dataplane",
		Long:         `dataplane: dataplane agent for allowing network connectivity of remote clients and services`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := opts.Run()
			if err != nil {
				log.Error(err)
			}
			return nil
		},
	}

	opts.AddFlags(cmd.Flags())

	for _, flag := range opts.RequiredFlags() {
		if err := cmd.MarkFlagRequired(flag); err != nil {
			fmt.Printf("Error marking required flag '%s': %v\n", flag, err)
			os.Exit(1)
		}
	}

	return cmd
}
