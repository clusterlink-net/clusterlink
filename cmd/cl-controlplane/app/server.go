// Copyright 2023 The ClusterLink Authors.
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
	"context"
	"fmt"
	"os"

	"github.com/bombsimon/logrusr/v4"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/authz"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/control"
	cprest "github.com/clusterlink-net/clusterlink/pkg/controlplane/rest"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/xds"
	"github.com/clusterlink-net/clusterlink/pkg/store/kv"
	"github.com/clusterlink-net/clusterlink/pkg/store/kv/bolt"
	"github.com/clusterlink-net/clusterlink/pkg/util/controller"
	"github.com/clusterlink-net/clusterlink/pkg/util/grpc"
	"github.com/clusterlink-net/clusterlink/pkg/util/log"
	utilrest "github.com/clusterlink-net/clusterlink/pkg/util/rest"
	"github.com/clusterlink-net/clusterlink/pkg/util/runnable"
	"github.com/clusterlink-net/clusterlink/pkg/util/sniproxy"
	"github.com/clusterlink-net/clusterlink/pkg/util/tls"
	"github.com/clusterlink-net/clusterlink/pkg/versioninfo"
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

	// NamespaceEnvVariable is the environment variable
	// which should hold the clusterlink system namespace name.
	NamespaceEnvVariable = "CL_NAMESPACE"
	// SystemNamespace represents the default clusterlink system namespace.
	SystemNamespace = "clusterlink-system"
)

// Options contains everything necessary to create and run a controlplane.
type Options struct {
	// LogFile is the path to file where logs will be written.
	LogFile string
	// LogLevel is the log level.
	LogLevel string
	// CRDMode indicates a k8s CRD-based controlplane.
	// This flag will be removed once the CRD-based controlplane feature is complete and stable.
	CRDMode bool
}

// AddFlags adds flags to fs and binds them to options.
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.LogFile, "log-file", "",
		"Path to a file where logs will be written. If not specified, logs will be printed to stderr.")
	fs.StringVar(&o.LogLevel, "log-level", logLevel,
		"The log level. One of fatal, error, warn, info, debug.")
	fs.BoolVar(&o.CRDMode, "crd-mode", false, "Run a CRD-based controlplane.")
}

// Run the various controlplane servers.
func (o *Options) Run() error {
	// set log file

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

	logrus.Infof("Starting cl-controlplane (version: %s)", versioninfo.Short())

	namespace := os.Getenv(NamespaceEnvVariable)
	if namespace == "" {
		namespace = SystemNamespace
	}
	logrus.Infof("ClusterLink namespace: %s", namespace)

	parsedCertData, err := tls.ParseFiles(CAFile, CertificateFile, KeyFile)
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

	config, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("unable to get k8s config: %w", err)
	}

	scheme, err := v1alpha1.SchemeBuilder.Build()
	if err != nil {
		return fmt.Errorf("unable to build k8s scheme: %w", err)
	}

	if err := v1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("unable to add core v1 objects to scheme: %w", err)
	}

	// set logger for controller-runtime components
	ctrl.SetLogger(logrusr.New(logrus.WithField("component", "k8s.controller-runtime")))

	// limit watch for v1alpha1.Peer to the namespace given by 'namespace'
	managerOptions := manager.Options{
		Cache: cache.Options{
			ByObject: map[client.Object]cache.ByObject{
				&v1alpha1.Peer{}: {
					Namespaces: map[string]cache.Config{
						namespace: {},
					},
				},
			},
		},
		Scheme: scheme,
	}

	mgr, err := manager.New(config, managerOptions)
	if err != nil {
		return fmt.Errorf(
			"unable to create k8s controller manager: %w", err)
	}

	controlplaneServerListenAddress := fmt.Sprintf("0.0.0.0:%d", api.ListenPort)
	sniProxy := sniproxy.NewServer(map[string]string{
		serverName:     httpServerAddress,
		grpcServerName: grpcServerAddress,
	})

	httpServer := utilrest.NewServer("controlplane-http", parsedCertData.ServerConfig())
	grpcServer := grpc.NewServer("controlplane-grpc", parsedCertData.ServerConfig())

	authzManager, err := authz.NewManager(parsedCertData)
	if err != nil {
		return fmt.Errorf("cannot create authorization manager: %w", err)
	}

	err = authz.CreateControllers(authzManager, mgr, o.CRDMode)
	if err != nil {
		return fmt.Errorf("cannot create authz controllers: %w", err)
	}

	authz.RegisterHandlers(authzManager, &httpServer.Server)

	controlManager := control.NewManager(mgr.GetClient(), parsedCertData, o.CRDMode)

	xdsManager := xds.NewManager()
	xds.RegisterService(
		context.Background(), xdsManager, grpcServer.GetGRPCServer())

	if o.CRDMode {
		err := xds.CreateControllers(xdsManager, mgr)
		if err != nil {
			return fmt.Errorf("cannot create xDS controllers: %w", err)
		}

		err = control.CreateControllers(controlManager, mgr)
		if err != nil {
			return fmt.Errorf("cannot create control controllers: %w", err)
		}
	} else {
		// open store
		kvStore, err := bolt.Open(StoreFile)
		if err != nil {
			return err
		}

		defer func() {
			if err := kvStore.Close(); err != nil {
				logrus.Warnf("Cannot close store: %v.", err)
			}
		}()

		storeManager := kv.NewManager(kvStore)

		restManager, err := cprest.NewManager(
			namespace, storeManager, xdsManager, authzManager, controlManager)
		if err != nil {
			return err
		}

		cprest.RegisterHandlers(restManager, httpServer)

		controlManager.SetStatusCallback(func(pr *v1alpha1.Peer) {
			authzManager.AddPeer(pr)
		})
	}

	runnableManager := runnable.NewManager()
	runnableManager.Add(controller.NewManager(mgr))
	runnableManager.Add(controlManager)
	runnableManager.AddServer(httpServerAddress, httpServer)
	runnableManager.AddServer(grpcServerAddress, grpcServer)
	runnableManager.AddServer(controlplaneServerListenAddress, sniProxy)

	return runnableManager.Run()
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
