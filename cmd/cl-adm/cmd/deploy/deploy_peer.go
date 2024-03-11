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

package deploy

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"

	"github.com/clusterlink-net/clusterlink/cmd/cl-adm/config"
	"github.com/clusterlink-net/clusterlink/cmd/cl-controlplane/app"
	configFiles "github.com/clusterlink-net/clusterlink/config"
	apis "github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/bootstrap/platform"
)

// PeerOptions contains everything necessary to create and run a 'deploy peer' subcommand.
type PeerOptions struct {
	// Name of the peer to create.
	Name string
	// Namespace where the ClusterLink components are deployed.
	Namespace string
	// CertDir is the directory where the certificates for the fabric and peer are located.
	CertDir string
	// StartInstance, if set to true, deploys a ClusterLink instance that will create the ClusterLink components.
	StartInstance bool
	// Ingress, represents the type of service used to expose the ClusterLink deployment.
	Ingress string
	// IngressPort, represents the port number of the service used to expose the ClusterLink deployment.
	IngressPort uint16
	// ContainerRegistry is the container registry to pull the project images.
	ContainerRegistry string
}

// NewCmdDeployPeer returns a cobra.Command to run the 'create peer' subcommand.
func NewCmdDeployPeer() *cobra.Command {
	opts := &PeerOptions{}

	cmd := &cobra.Command{
		Use:   "peer",
		Short: "Deploy a peer",
		Long:  `Deploy a peer`,

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
func (o *PeerOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.Name, "name", "", "Peer name.")
	fs.StringVar(&o.CertDir, "cert-dir", ".", "The directory where the certificates for the fabric and peer are located.")
	fs.StringVar(&o.Namespace, "namespace", app.SystemNamespace,
		"Namespace where the ClusterLink components are deployed.")
	fs.StringVar(&o.ContainerRegistry, "container-registry", config.DefaultRegistry,
		"The container registry to pull the project images.")
	fs.BoolVar(&o.StartInstance, "autostart", false,
		"If false, it will deploy only the ClusteLink operator and ClusterLink K8s secrets."+
			"If true, it will also deploy the ClusterLink instance CRD, which will create the ClusterLink components.")
	fs.StringVar(&o.Ingress, "ingress", string(apis.IngressTypeLoadBalancer), "Represents the type of service used"+
		"to expose the ClusterLink deployment (LoadBalancer/NodePort/none). This option is only valid if --autostart is set.")
	fs.Uint16Var(&o.IngressPort, "ingress-port", apis.DefaultExternalPort,
		"Represents the ingress port. By default it is set to 443 for LoadBalancer"+
			" and a random port in range (30000 to 32767) for NodePort. This option is only valid if --autostart is set.")
}

// RequiredFlags are the names of flags that must be explicitly specified.
func (o *PeerOptions) RequiredFlags() []string {
	return []string{"name"}
}

// Run the 'create peer' subcommand.
func (o *PeerOptions) Run() error {
	peerDir := path.Join(o.CertDir, o.Name)
	if _, err := os.Stat(peerDir); err != nil {
		return fmt.Errorf("failed to open certificates folder: %w", err)
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
	ghImage := path.Join(config.DefaultRegistry, "cl-operator:latest")
	newImage := path.Join(o.ContainerRegistry, "cl-operator:latest")
	managerFile, err := configFiles.ConfigFiles.ReadFile("operator/manager/manager.yaml")
	if err != nil {
		return err
	}

	managerModified := strings.ReplaceAll(string(managerFile), ghImage, newImage)
	err = decoder.DecodeEach(context.Background(), strings.NewReader(managerModified), decoder.CreateIgnoreAlreadyExists(resource))
	if err != nil {
		return err
	}

	if err := o.deployDir("operator/rbac/*", resource); err != nil {
		return err
	}
	if err := o.deployDir("crds/*", resource); err != nil {
		return err
	}

	// Create cl-secret
	secretFileName := path.Join(peerDir, config.K8SSecretYAMLFile)
	secretFile, err := os.ReadFile(secretFileName)
	if err != nil {
		return err
	}

	err = decoder.DecodeEach(
		context.Background(),
		strings.NewReader(string(secretFile)),
		decoder.CreateIgnoreAlreadyExists(resource),
		decoder.MutateNamespace(o.Namespace),
	)
	if err != nil {
		return err
	}

	// Create ClusterLink instance
	if o.StartInstance {
		cfg := &platform.Config{
			Peer:              o.Name,
			Dataplanes:        1,
			DataplaneType:     platform.DataplaneTypeEnvoy,
			LogLevel:          "info",
			ContainerRegistry: o.ContainerRegistry,
			Namespace:         o.Namespace,
			IngressType:       o.Ingress,
		}
		if o.IngressPort != apis.DefaultExternalPort {
			cfg.IngressPort = o.IngressPort
		}
		instance, err := platform.K8SClusterLinkInstanceConfig(cfg, "cl-instance")
		if err != nil {
			return err
		}

		err = decoder.DecodeEach(context.Background(), strings.NewReader(string(instance)), decoder.CreateHandler(resource))
		if errors.IsAlreadyExists(err) {
			fmt.Println("CRD instance for ClusterLink (\"cl-instance\") was already exist.")
		} else if err != nil {
			return err
		}
	} else {
		if o.Ingress != string(apis.IngressTypeLoadBalancer) {
			fmt.Println("flag --autostart is not set, ignoring --ingres flag")
		}
		if o.IngressPort != apis.DefaultExternalPort {
			fmt.Println("flag --autostart is not set, ignoring --ingres-port flag")
		}
	}

	return nil
}

// deployDir deploys K8s yaml from a directory.
func (o *PeerOptions) deployDir(dir string, resource *resources.Resources) error {
	err := decoder.DecodeEachFile(context.Background(), configFiles.ConfigFiles, dir, decoder.CreateHandler(resource))
	if errors.IsAlreadyExists(err) {
		return nil
	}

	return err
}
