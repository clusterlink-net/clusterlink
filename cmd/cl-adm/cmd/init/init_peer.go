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

package initcmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"

	"github.com/clusterlink-net/clusterlink/cmd/cl-adm/config"
	"github.com/clusterlink-net/clusterlink/cmd/cl-controlplane/app"
	configFiles "github.com/clusterlink-net/clusterlink/config"
	"github.com/clusterlink-net/clusterlink/pkg/bootstrap/platform"
)

// InitPeerOptions contains everything necessary to create and run a 'init peer' subcommand.
type InitPeerOptions struct {
	// Name of the peer to create.
	Name string
	// Namespace where the ClusterLink components are deployed.
	Namespace string
	// CertFolder is the folder where the certificates for the fabric and peer are located.
	CertFolder string
	// RunInstance, if set to true, deploys a ClusterLink instance that will create the ClusterLink components.
	RunInstance bool
}

// NewCmdInitPeer returns a cobra.Command to run the 'create peer' subcommand.
func NewCmdInitPeer() *cobra.Command {
	opts := &InitPeerOptions{}

	cmd := &cobra.Command{
		Use:   "peer",
		Short: "Init a peer",
		Long:  `Init a peer`,

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

// AddFlags adds flags to fs and binds them to options.
func (o *InitPeerOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Name, "name", "", "Peer name.")
	fs.StringVar(&o.Namespace, "namespace", app.SystemNamespace, "Namespace where the ClusterLink components are deployed.")
	fs.StringVar(&o.CertFolder, "cert-folder", ".", "The folder where the certificates for the fabric and peer are located.")
	fs.BoolVar(&o.RunInstance, "run", false, "If true, deploys a ClusterLink instance that will create the ClusterLink components.")
}

// RequiredFlags are the names of flags that must be explicitly specified.
func (o *InitPeerOptions) RequiredFlags() []string {
	return []string{"name"}
}

// Run the 'create peer' subcommand.
func (o *InitPeerOptions) Run() error {
	peerFolder := o.CertFolder + "/" + o.Name
	if err := o.verifyExists(peerFolder); err != nil {
		return err
	}

	// Create k8s resources
	cfg, err := ctrl.GetConfig()
	if err != nil {
		return err
	}

	resource, err := resources.New(cfg)
	if err != nil {
		return err
	}

	// Create operator
	if err := o.deployFolder("operator/manager/*", resource); err != nil {
		return err
	}

	if err := o.deployFolder("operator/rbac/*", resource); err != nil {
		return err
	}
	if err := o.deployFolder("crds/*", resource); err != nil {
		return err
	}

	// Create cl-secret
	secretFileName := peerFolder + "/" + config.K8SSecretYAMLFile
	secretFile, err := os.ReadFile(secretFileName)
	if err != nil {
		return err
	}

	if err := o.deploySecretFile(string(secretFile), o.Namespace, resource); err != nil {
		return err
	}

	// Create ClusterLink instance
	if o.RunInstance {
		instance, err := platform.K8SClusterLinkInstanceConfig(&platform.Config{
			Peer:              o.Name,
			Dataplanes:        1,
			DataplaneType:     platform.DataplaneTypeEnvoy,
			LogLevel:          "info",
			ContainerRegistry: "ghcr.io/clusterlink-net", // Tell kind to use local image.
			Namespace:         o.Namespace,
			IngressType:       "NodePort",
		}, "cl-instance")
		if err != nil {
			return err
		}

		err = decoder.DecodeEach(context.Background(), strings.NewReader(string(instance)), decoder.CreateHandler(resource))
		if err != nil {
			return err
		}
	}

	return nil
}

// verifyExists verifies a given path exist.
func (o *InitPeerOptions) verifyExists(path string) error {
	_, err := os.Stat(path)
	return err
}

// deployFolder deploys K8s yaml from a folder.
func (o *InitPeerOptions) deployFolder(path string, resource *resources.Resources) error {
	err := decoder.DecodeEachFile(context.Background(), configFiles.ConfigFiles, path, decoder.CreateHandler(resource))
	if errors.IsAlreadyExists(err) {
		return nil
	}

	return err
}

// deploySecretFile deploys all the secret in YAML file.
func (o *InitPeerOptions) deploySecretFile(yamlContent, newNamespace string, resource *resources.Resources) error {
	// Split the YAML content into separate documents
	yamlDocuments := strings.Split(yamlContent, "---")

	for _, doc := range yamlDocuments {
		if strings.TrimSpace(doc) == "" {
			continue
		}

		var secretMap map[string]interface{}
		if err := yaml.Unmarshal([]byte(doc), &secretMap); err != nil {
			return err
		}
		// Update namespce
		if metadata, ok := secretMap["metadata"].(map[interface{}]interface{}); ok {
			metadata["namespace"] = newNamespace
			secretMap["metadata"] = metadata
		}

		secretMapUpdate, err := yaml.Marshal(&secretMap)
		if err != nil {
			return err
		}
		// Create secrets
		err = decoder.DecodeEach(context.Background(), strings.NewReader(string(secretMapUpdate)), decoder.CreateHandler(resource))
		if err != nil {
			return err
		}
	}

	return nil
}
