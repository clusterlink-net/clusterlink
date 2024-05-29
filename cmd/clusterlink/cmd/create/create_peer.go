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

package create

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/net/idna"

	"github.com/clusterlink-net/clusterlink/cmd/clusterlink/config"
	"github.com/clusterlink-net/clusterlink/pkg/bootstrap"
)

// PeerOptions contains everything necessary to create and run a 'create peer-cert' subcommand.
type PeerOptions struct {
	// Name of the peer to create.
	Name string
	// Name of the fabric that the peer belongs to.
	Fabric string
	// Path where the certificates will be created.
	Path string
}

// AddFlags adds flags to fs and binds them to options.
func (o *PeerOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Name, "name", "", "Peer name.")
	fs.StringVar(&o.Fabric, "fabric", config.DefaultFabric, "Fabric name.")
	fs.StringVar(&o.Path, "path", ".", "Path where the certificates will be created.")
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

func (o *PeerOptions) createPeerCert(fabricCert *bootstrap.Certificate) (*bootstrap.Certificate, error) {
	cert, err := bootstrap.CreatePeerCertificate(o.Name, fabricCert)
	if err != nil {
		return nil, err
	}

	outDirectory := config.PeerDirectory(o.Name, o.Fabric, o.Path)
	if err := os.Mkdir(outDirectory, 0o755); err != nil {
		return nil, err
	}

	if err := o.saveCertificate(cert, outDirectory); err != nil {
		return nil, err
	}

	return cert, nil
}

// Run the 'create peer-cert' subcommand.
func (o *PeerOptions) Run() error {
	if _, err := idna.Lookup.ToASCII(o.Name); err != nil {
		return fmt.Errorf("peer name is not a valid DNS name: %w", err)
	}

	if err := verifyNotExists(o.Name); err != nil {
		return err
	}

	fabricCert, err := bootstrap.ReadCertificates(config.FabricDirectory(o.Fabric, o.Path), true)
	if err != nil {
		return err
	}

	if _, err := o.createPeerCert(fabricCert); err != nil {
		return err
	}

	return nil
}

// NewCmdCreatePeerCert returns a cobra.Command to run the 'create peer-cert' subcommand.
func NewCmdCreatePeerCert() *cobra.Command {
	opts := &PeerOptions{}

	cmd := &cobra.Command{
		Use:   "peer-cert",
		Short: "Create peer certificate and private key",
		Long:  `Create peer certificate and private key`,

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
