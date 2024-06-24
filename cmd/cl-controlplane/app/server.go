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
	"context"
	"fmt"
	"os"
	"path"

	"github.com/bombsimon/logrusr/v4"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"

	discv1 "k8s.io/api/discovery/v1"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/authz"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/control"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/peer"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/xds"
	"github.com/clusterlink-net/clusterlink/pkg/util/controller"
	"github.com/clusterlink-net/clusterlink/pkg/util/grpc"
	"github.com/clusterlink-net/clusterlink/pkg/util/log"
	"github.com/clusterlink-net/clusterlink/pkg/util/runnable"
	"github.com/clusterlink-net/clusterlink/pkg/util/tls"
	"github.com/clusterlink-net/clusterlink/pkg/versioninfo"
)

const (
	// logLevel is the default log level.
	logLevel = "warn"

	// CAFile is the path to the certificate authority file.
	CAFile = "/etc/ssl/certs/clink_ca.pem"
	// CertificateFile is the path to the certificate file.
	CertificateFile = "/etc/ssl/certs/clink-controlplane.pem"
	// KeyFile is the path to the private-key file.
	KeyFile = "/etc/ssl/key/clink-controlplane.pem"

	// PeerTLSDirectory is the path to the directory holding the peer TLS certificates.
	PeerTLSDirectory = "/etc/ssl/certs/clink"
	// PeerCertificateFile is the name to the peer certificate file.
	PeerCertificateFile = "cert.pem"
	// PeerKeyFile is the name of the peer private-key file.
	PeerKeyFile = "key.pem"
	// FabricCertificateFile is the name of the fabric CA file.
	FabricCertificateFile = "ca.pem"

	// NamespaceEnvVariable is the environment variable
	// which should hold the clusterlink system namespace name.
	NamespaceEnvVariable = "CL_NAMESPACE"
	// SystemNamespace represents the default clusterlink system namespace.
	SystemNamespace = "clusterlink-system"
)

// PeerCertificateFilePath returns the path to the peer certificate file.
func PeerCertificateFilePath() string {
	return path.Join(PeerTLSDirectory, PeerCertificateFile)
}

// PeerKeyFilePath returns the path to the peer private key file.
func PeerKeyFilePath() string {
	return path.Join(PeerTLSDirectory, PeerKeyFile)
}

// FabricCertificateFilePath returns the path to the fabric CA file.
func FabricCertificateFilePath() string {
	return path.Join(PeerTLSDirectory, FabricCertificateFile)
}

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

	controlplaneCertData, _, err := tls.ParseFiles(CAFile, CertificateFile, KeyFile)
	if err != nil {
		return err
	}

	peerCertsWatcher := peer.NewWatcher(
		FabricCertificateFilePath(), PeerCertificateFilePath(), PeerKeyFilePath())

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

	if err := discv1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("unable to add discovery v1 objects to scheme: %w", err)
	}

	if err := appsv1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("unable to add core appsv1 objects to scheme: %w", err)
	}

	// set logger for controller-runtime components
	ctrl.SetLogger(logrusr.New(logrus.WithField("component", "k8s.controller-runtime")))

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
	grpcServer := grpc.NewServer("controlplane-grpc", controlplaneCertData.ServerConfig())

	authzManager, err := authz.NewManager(mgr.GetClient(), namespace)
	if err != nil {
		return fmt.Errorf("cannot create authorization manager: %w", err)
	}

	peerCertsWatcher.AddConsumer(authzManager)

	err = authz.CreateControllers(authzManager, mgr)
	if err != nil {
		return fmt.Errorf("cannot create authz controllers: %w", err)
	}

	authz.RegisterService(authzManager, grpcServer.GetGRPCServer())

	controlManager := control.NewManager(mgr.GetClient(), namespace)
	peerCertsWatcher.AddConsumer(controlManager)

	err = control.CreateControllers(controlManager, mgr)
	if err != nil {
		return fmt.Errorf("cannot create control controllers: %w", err)
	}

	xdsManager := xds.NewManager()
	xds.RegisterService(
		context.Background(), xdsManager, grpcServer.GetGRPCServer())
	peerCertsWatcher.AddConsumer(xdsManager)

	if err := xds.CreateControllers(xdsManager, mgr); err != nil {
		return fmt.Errorf("cannot create xDS controllers: %w", err)
	}

	if err := peerCertsWatcher.ReadCertsAndUpdateConsumers(); err != nil {
		return err
	}

	runnableManager := runnable.NewManager()
	runnableManager.Add(peerCertsWatcher)
	runnableManager.Add(controller.NewManager(mgr))
	runnableManager.Add(controlManager)
	runnableManager.AddServer(controlplaneServerListenAddress, grpcServer)

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
