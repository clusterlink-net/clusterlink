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

package create

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/net/idna"

	"github.com/clusterlink-net/clusterlink/cmd/cl-adm/config"
	"github.com/clusterlink-net/clusterlink/pkg/bootstrap"
	"github.com/clusterlink-net/clusterlink/pkg/bootstrap/platform"
)

// PeerOptions contains everything necessary to create and run a 'create peer' subcommand.
type PeerOptions struct {
	// Name of the peer to create.
	Name string
	// Namespace where the ClusterLink components are deployed.
	Namespace string
	// Dataplanes is the number of dataplanes to create.
	Dataplanes uint16
	// DataplaneType is the type of dataplane to create (envoy or go-based)
	DataplaneType string
	// LogLevel is the log level.
	LogLevel string
	// ContainerRegistry is the container registry to pull the project images.
	ContainerRegistry string
	// CRDMode indicates whether to run a k8s CRD-based controlplane.
	// This flag will be removed once the CRD-based controlplane feature is complete and stable.
	CRDMode bool
}

// AddFlags adds flags to fs and binds them to options.
func (o *PeerOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Name, "name", "", "Peer name.")
	fs.StringVar(&o.Namespace, "namespace", platform.SystemNamespace, "Namespace where the ClusterLink components are deployed.")
	fs.Uint16Var(&o.Dataplanes, "dataplanes", 1, "Number of dataplanes.")
	fs.StringVar(&o.DataplaneType, "dataplane-type", platform.DataplaneTypeEnvoy,
		"Type of dataplane, Supported values: \"envoy\" (default), \"go\"")
	fs.StringVar(&o.LogLevel, "log-level", "info",
		"The log level. One of fatal, error, warn, info, debug.")
	fs.StringVar(&o.ContainerRegistry, "container-registry", "ghcr.io/clusterlink-net",
		"The container registry to pull the project images. If empty will use local registry.")
	fs.BoolVar(&o.CRDMode, "crd-mode", false, "Run a CRD-based controlplane.")
}

// RequiredFlags are the names of flags that must be explicitly specified.
func (o *PeerOptions) RequiredFlags() []string {
	return []string{"name"}
}

func (o *PeerOptions) saveCertificate(cert *bootstrap.Certificate, outDirectory string) error {
	// save certificate to file
	err := os.WriteFile(filepath.Join(outDirectory, config.CertificateFileName), cert.RawCert(), 0o600)
	if err != nil {
		return err
	}

	// save private key to file
	return os.WriteFile(filepath.Join(outDirectory, config.PrivateKeyFileName), cert.RawKey(), 0o600)
}

func (o *PeerOptions) createControlplane(peerCert *bootstrap.Certificate) (*bootstrap.Certificate, error) {
	cert, err := bootstrap.CreateControlplaneCertificate(o.Name, peerCert)
	if err != nil {
		return nil, err
	}

	outDirectory := config.ControlplaneDirectory(o.Name)
	if err := os.Mkdir(outDirectory, 0o755); err != nil {
		return nil, err
	}

	if err := o.saveCertificate(cert, outDirectory); err != nil {
		return nil, err
	}

	return cert, nil
}

func (o *PeerOptions) createDataplane(peerCert *bootstrap.Certificate) (*bootstrap.Certificate, error) {
	cert, err := bootstrap.CreateDataplaneCertificate(o.Name, peerCert)
	if err != nil {
		return nil, err
	}

	outDirectory := config.DataplaneDirectory(o.Name)
	if err := os.Mkdir(outDirectory, 0o755); err != nil {
		return nil, err
	}

	if err := o.saveCertificate(cert, outDirectory); err != nil {
		return nil, err
	}

	return cert, nil
}

func (o *PeerOptions) createGWCTL(peerCert *bootstrap.Certificate) (*bootstrap.Certificate, error) {
	cert, err := bootstrap.CreateGWCTLCertificate(peerCert)
	if err != nil {
		return nil, err
	}

	outDirectory := config.GWCTLDirectory(o.Name)
	if err := os.Mkdir(outDirectory, 0o755); err != nil {
		return nil, err
	}

	if err := o.saveCertificate(cert, outDirectory); err != nil {
		return nil, err
	}

	return cert, nil
}

// Run the 'create peer' subcommand.
func (o *PeerOptions) Run() error {
	if _, err := idna.Lookup.ToASCII(o.Name); err != nil {
		return fmt.Errorf("peer name is not a valid DNS name: %w", err)
	}

	if err := verifyNotExists(o.Name); err != nil {
		return err
	}

	if err := verifyDataplaneType(o.DataplaneType); err != nil {
		return err
	}

	// read fabric certificate
	rawFabricCert, err := os.ReadFile(config.CertificateFileName)
	if err != nil {
		return err
	}

	// read fabric key
	rawFabricKey, err := os.ReadFile(config.PrivateKeyFileName)
	if err != nil {
		return err
	}

	fabricCert, err := bootstrap.CertificateFromRaw(rawFabricCert, rawFabricKey)
	if err != nil {
		return err
	}

	peerDirectory := config.PeerDirectory(o.Name)
	if err := os.Mkdir(peerDirectory, 0o755); err != nil {
		return err
	}

	peerCertificate, err := bootstrap.CreatePeerCertificate(o.Name, fabricCert)
	if err != nil {
		return err
	}

	err = o.saveCertificate(peerCertificate, config.PeerDirectory(o.Name))
	if err != nil {
		return err
	}

	controlplaneCert, err := o.createControlplane(peerCertificate)
	if err != nil {
		return err
	}

	dataplaneCert, err := o.createDataplane(peerCertificate)
	if err != nil {
		return err
	}

	gwctlCert, err := o.createGWCTL(peerCertificate)
	if err != nil {
		return err
	}

	// create k8s deployment YAML
	platformCfg := &platform.Config{
		Peer:                    o.Name,
		FabricCertificate:       fabricCert,
		PeerCertificate:         peerCertificate,
		ControlplaneCertificate: controlplaneCert,
		DataplaneCertificate:    dataplaneCert,
		GWCTLCertificate:        gwctlCert,
		Dataplanes:              o.Dataplanes,
		DataplaneType:           o.DataplaneType,
		LogLevel:                o.LogLevel,
		ContainerRegistry:       o.ContainerRegistry,
		CRDMode:                 o.CRDMode,
		Namespace:               o.Namespace,
	}
	k8sConfig, err := platform.K8SConfig(platformCfg)
	if err != nil {
		return err
	}

	outPath := filepath.Join(peerDirectory, config.K8SYAMLFile)
	if err := os.WriteFile(outPath, k8sConfig, 0o600); err != nil {
		return err
	}

	// Create k8s secrets YAML file that contains the components certificates.
	certConfig, err := platform.K8SCertificateConfig(platformCfg)
	if err != nil {
		return err
	}

	certOutPath := filepath.Join(peerDirectory, config.K8SSecretYAMLFile)
	if err := os.WriteFile(certOutPath, certConfig, 0o600); err != nil {
		return err
	}

	// Create clusterlink instance YAML for the operator.
	clConfig, err := platform.K8SClusterLinkInstanceConfig(platformCfg)
	if err != nil {
		return err
	}

	clOutPath := filepath.Join(peerDirectory, config.K8SClusterLinkInstanceYAMLFile)
	return os.WriteFile(clOutPath, clConfig, 0o600)
}

// NewCmdCreatePeer returns a cobra.Command to run the 'create peer' subcommand.
func NewCmdCreatePeer() *cobra.Command {
	opts := &PeerOptions{}

	cmd := &cobra.Command{
		Use:   "peer",
		Short: "Create a peer",
		Long:  `Create a peer`,

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

// verifyNotExists verifies a given path does not exist.
func verifyNotExists(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return fmt.Errorf("path %s exists", path)
	}

	if !os.IsNotExist(err) {
		return err
	}

	return nil
}

// verifyDataplaneType checks if the given dataplane type is valid.
func verifyDataplaneType(dType string) error {
	switch dType {
	case platform.DataplaneTypeEnvoy:
		return nil
	case platform.DataplaneTypeGo:
		return nil
	default:
		return fmt.Errorf("undefined dataplane-type %s", dType)
	}
}
