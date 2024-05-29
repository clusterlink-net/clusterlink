// Copyright (c) The ClusterLink Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package app

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials"

	cpapi "github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
	"github.com/clusterlink-net/clusterlink/pkg/dataplane/api"
	dpclient "github.com/clusterlink-net/clusterlink/pkg/dataplane/client"
	dpserver "github.com/clusterlink-net/clusterlink/pkg/dataplane/server"
	"github.com/clusterlink-net/clusterlink/pkg/util/log"
	"github.com/clusterlink-net/clusterlink/pkg/util/tls"
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

// Run the go dataplane.
func (o *Options) runGoDataplane(dataplaneID string, parsedCertData *tls.ParsedCertData) error {
	controlplaneTarget := net.JoinHostPort(o.ControlplaneHost, strconv.Itoa(cpapi.ListenPort))

	logrus.Infof("Starting go dataplane, ID: %s", dataplaneID)

	controlplaneClient, err := grpc.NewClient(
		controlplaneTarget,
		grpc.WithTransportCredentials(credentials.NewTLS(parsedCertData.ClientConfig("cl-controlplane"))),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  100 * time.Millisecond,
				Multiplier: 1.6,
				Jitter:     0.2,
				MaxDelay:   time.Second,
			},
		}))
	if err != nil {
		return fmt.Errorf("error initializing controlplane client: %w", err)
	}

	dataplaneServerAddress := fmt.Sprintf(":%d", api.ListenPort)
	dataplane := dpserver.NewDataplane(dataplaneID, controlplaneClient, parsedCertData)
	go func() {
		err := dataplane.StartDataplaneServer(dataplaneServerAddress)
		logrus.Errorf("Failed to start dataplane server: %v.", err)
	}()

	// Start xDS client, if it fails to start we keep retrying to connect to the controlplane host
	xdsClient := dpclient.NewXDSClient(dataplane, controlplaneClient)
	err = xdsClient.Run()
	return fmt.Errorf("xDS Client stopped: %w", err)
}

// Run the dataplane.
func (o *Options) Run() error {
	f, err := log.Set(o.LogLevel, o.LogFile)
	if err != nil {
		return err
	}
	if f != nil {
		defer func() {
			if err := f.Close(); err != nil {
				logrus.Errorf("Cannot close log file: %v", err)
			}
		}()
	}

	// parse TLS files
	parsedCertData, _, err := tls.ParseFiles(CAFile, CertificateFile, KeyFile)
	if err != nil {
		return err
	}

	// generate random dataplane ID
	dataplaneID := uuid.New().String()
	logrus.Infof("Dataplane ID: %s.", dataplaneID)

	return o.runGoDataplane(dataplaneID, parsedCertData)
}

// NewCLGoDataplaneCommand creates a *cobra.Command object with default parameters.
func NewCLGoDataplaneCommand() *cobra.Command {
	opts := &Options{}

	cmd := &cobra.Command{
		Use:          "cl-go-dataplane",
		Long:         `cl-go-dataplane: dataplane agent for allowing network connectivity of remote clients and services`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.Run()
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
