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
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/clusterlink-net/clusterlink/cmd/clusterlink/config"
	"github.com/clusterlink-net/clusterlink/pkg/bootstrap"
)

// FabricOptions contains everything necessary to create and run a 'createfabric' subcommand.
type FabricOptions struct {
	// Name of the fabric to create.
	Name string
	// Path where the certificates will be created.
	Path string
}

// AddFlags adds flags to fs and binds them to options.
func (o *FabricOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Name, "name", config.DefaultFabric, "Fabric name.")
	fs.StringVar(&o.Path, "path", ".", "Path where the certificates will be created.")
}

// NewCmdCreateFabric returns a cobra.Command to run the 'create fabric' subcommand.
func NewCmdCreateFabric() *cobra.Command {
	opts := &FabricOptions{}
	cmd := &cobra.Command{
		Use:   "fabric",
		Short: "Create fabric certificates",
		Long:  `Create fabric certificates`,

		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.Run()
		},
	}

	opts.AddFlags(cmd.Flags())
	return cmd
}

// Run the 'create fabric' subcommand.
func (o *FabricOptions) Run() error {
	fabricCert, err := bootstrap.CreateFabricCertificate(o.Name)
	if err != nil {
		return err
	}

	if err := os.Mkdir(config.FabricDirectory(o.Name, o.Path), 0o755); err != nil {
		return err
	}
	// save certificate to file
	err = os.WriteFile(config.FabricCertificate(o.Name, o.Path), fabricCert.RawCert(), 0o600)
	if err != nil {
		return err
	}

	// save private key to file
	return os.WriteFile(config.FabricKey(o.Name, o.Path), fabricCert.RawKey(), 0o600)
}
