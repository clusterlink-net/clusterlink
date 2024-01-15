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
	"os"

	"github.com/spf13/cobra"

	"github.com/clusterlink-net/clusterlink/cmd/cl-adm/config"
	"github.com/clusterlink-net/clusterlink/pkg/bootstrap"
)

// NewCmdCreateFabric returns a cobra.Command to run the 'create fabric' subcommand.
func NewCmdCreateFabric() *cobra.Command {
	return &cobra.Command{
		Use:   "fabric",
		Short: "Create a fabric",
		Long:  `Create a fabric`,

		RunE: func(cmd *cobra.Command, args []string) error {
			fabricCert, err := bootstrap.CreateFabricCertificate()
			if err != nil {
				return err
			}

			// save certificate to file
			err = os.WriteFile(config.CertificateFileName, fabricCert.RawCert(), 0o600)
			if err != nil {
				return err
			}

			// save private key to file
			return os.WriteFile(config.PrivateKeyFileName, fabricCert.RawKey(), 0o600)
		},
	}
}
