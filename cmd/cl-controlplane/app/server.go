package app

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/clusterlink-net/clusterlink/pkg/controlplane"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/server"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/server/grpc"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/server/http"
	"github.com/clusterlink-net/clusterlink/pkg/store/kv"
	"github.com/clusterlink-net/clusterlink/pkg/store/kv/bolt"
	"github.com/clusterlink-net/clusterlink/pkg/util"
	logutils "github.com/clusterlink-net/clusterlink/pkg/util/log"
	"github.com/clusterlink-net/clusterlink/pkg/util/sniproxy"
)

const (
	// logLevel is the default log level.
	logLevel = "warn"

	// StoreFile is the path to the file holding the persisted state.
	StoreFile = "/var/lib/clink/controlplane.db"

	// CAFile is the path to the certificate authority file.
	CAFile = "/etc/ssl/certs/clink_ca.pem"
	// CertificateFile is the path to the certificate file.
	CertificateFile = "/etc/ssl/certs/clink-controlplane.pem"
	// KeyFile is the path to the private-key file.
	KeyFile = "/etc/ssl/private/clink-controlplane.pem"

	// httpServerAddress is the address of the localhost HTTP server.
	httpServerAddress = "127.0.0.1:1100"
	// grpcServerAddress is the address of the localhost gRPC server.
	grpcServerAddress = "127.0.0.1:1101"
)

// Options contains everything necessary to create and run a controlplane.
type Options struct {
	// LogFile is the path to file where logs will be written.
	LogFile string
	// LogLevel is the log level.
	LogLevel string
}

// AddFlags adds flags to fs and binds them to options.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.LogFile, "log-file", "",
		"Path to a file where logs will be written. If not specified, logs will be printed to stderr.")
	fs.StringVar(&o.LogLevel, "log-level", logLevel,
		"The log level. One of fatal, error, warn, info, debug.")
}

// Run the various controlplane servers.
func (o *Options) Run() error {
	// set log file

	logutils.SetLog(o.LogLevel, o.LogFile)

	parsedCertData, err := util.ParseTLSFiles(CAFile, CertificateFile, KeyFile)
	if err != nil {
		return err
	}

	dnsNames := parsedCertData.DNSNames()
	if len(dnsNames) != 2 {
		return fmt.Errorf("expected peer certificate to contain 2 DNS names, but got %d", len(dnsNames))
	}

	serverName := dnsNames[0]
	grpcServerName := dnsNames[1]

	expectedGRPCServerName := api.GRPCServerName(serverName)
	if grpcServerName != expectedGRPCServerName {
		return fmt.Errorf("expected second DNS name to be '%s', but got: '%s'",
			expectedGRPCServerName, grpcServerName)
	}

	// open store
	kvStore, err := bolt.Open(StoreFile)
	if err != nil {
		return err
	}

	defer func() {
		if err := kvStore.Close(); err != nil {
			log.Warnf("Cannot close store: %v.", err)
		}
	}()

	storeManager := kv.NewManager(kvStore)

	// TODO: initialize kubernetes client and pass to NewInstance
	cp, err := controlplane.NewInstance(parsedCertData, storeManager, nil)
	if err != nil {
		return err
	}

	controlplaneServerListenAddress := fmt.Sprintf("0.0.0.0:%d", api.ListenPort)
	sniProxy := sniproxy.NewServer(map[string]string{
		serverName:     httpServerAddress,
		grpcServerName: grpcServerAddress,
	})

	servers := server.NewController()
	servers.Add(httpServerAddress, http.NewServer(cp, parsedCertData.ServerConfig()))
	servers.Add(grpcServerAddress, grpc.NewServer(cp, parsedCertData.ServerConfig()))
	servers.Add(controlplaneServerListenAddress, sniProxy)

	return servers.Run()
}

// NewCLControlplaneCommand creates a *cobra.Command object with default parameters.
func NewCLControlplaneCommand() *cobra.Command {
	opts := &Options{}

	cmd := &cobra.Command{
		Use:          "cl-controlplane",
		Long:         `cl-controlplane: controlplane agent for allowing network connectivity of remote clients and services`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.Run()
		},
	}

	opts.AddFlags(cmd.Flags())

	return cmd
}
