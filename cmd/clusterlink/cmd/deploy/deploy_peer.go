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

package deploy

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"

	// Importing this package for initializing the OIDC authentication plugin for client-go.
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	"github.com/clusterlink-net/clusterlink/cmd/cl-controlplane/app"
	"github.com/clusterlink-net/clusterlink/cmd/clusterlink/config"
	configFiles "github.com/clusterlink-net/clusterlink/config"
	apis "github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/bootstrap"
	"github.com/clusterlink-net/clusterlink/pkg/bootstrap/platform"
)

const (
	// StartAll deploys the clusterlink operator, converts the peer certificates to secrets,
	// creates and deploys the operator ClusterLink custom resource to create the ClusterLink components.
	StartAll = "all"
	// StartOperator deploys only the operator and converts the peer certificates to secrets.
	// Creates a custom resource example file that can be deployed to the operator.
	StartOperator = "operator"
	// NoStart doesn't deploy the operator and creates a "k8s.yaml" file that allow to deploy ClusterLink without the operator.
	NoStart = "none"
)

// PeerOptions contains everything necessary to create and run a 'deploy peer' subcommand.
type PeerOptions struct {
	// Name of the peer to deploy.
	Name string
	// Name of the fabric that the peer belongs to.
	Fabric string
	// Namespace where the ClusterLink components are deployed.
	Namespace string
	// Path is the directory where the certificates for the fabric and peer are located.
	Path string
	// StartInstance, represents which component to deploy:
	// `all` (clusterlink control-plane, data-plane and operator), `operator`, or `none`.
	StartInstance string
	// Ingress, represents the type of service used to expose the ClusterLink deployment.
	Ingress string
	// IngressPort, represents the port number of the service used to expose the ClusterLink deployment.
	IngressPort uint16
	// IngressAnnotations represents the annotations that will be added to the ingress service.
	IngressAnnotations map[string]string
	// ContainerRegistry is the container registry to pull the project images.
	ContainerRegistry string
	// Tag represents the tag of the project images.
	Tag string
	// Dataplanes is the number of dataplanes to create.
	DataplaneReplicas uint16
	// DataplaneType is the type of dataplane to create (envoy or go-based)
	DataplaneType string
	// LogLevel is the log level.
	LogLevel string
}

// NewCmdDeployPeer returns a cobra.Command to run the 'deploy peer' subcommand.
func NewCmdDeployPeer() *cobra.Command {
	opts := &PeerOptions{}

	cmd := &cobra.Command{
		Use:   "peer",
		Short: "Deploy ClusterLink components to a peer (K8s cluster).",
		Long:  `Deploy ClusterLink components to a peer (K8s cluster).`,

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
	fs.StringVar(&o.Fabric, "fabric", config.DefaultFabric, "Fabric name.")
	fs.StringVar(&o.Path, "path", ".", "The directory where the certificates for the fabric and peer are located.")
	fs.StringVar(&o.Namespace, "namespace", app.SystemNamespace,
		"Namespace where the ClusterLink components are deployed.")
	fs.StringVar(&o.ContainerRegistry, "container-registry", config.DefaultRegistry,
		"The container registry to pull the project images.")
	fs.StringVar(&o.Tag, "tag", "latest", "The tag of the project images.")
	fs.StringVar(&o.StartInstance, "start", StartAll,
		"Represents which component to deploy and start in the cluster: "+
			"`all` (clusterlink components and operator), `operator`, or `none`.")
	fs.StringVar(&o.Ingress, "ingress", string(apis.IngressTypeLoadBalancer), "Represents the type of service used"+
		"to expose the ClusterLink deployment (LoadBalancer/NodePort/none).")
	fs.Uint16Var(&o.IngressPort, "ingress-port", apis.DefaultExternalPort,
		"Represents the ingress port. By default it is set to 443 for LoadBalancer"+
			" and a random port in range (30000 to 32767) for NodePort.")
	fs.StringToStringVar(&o.IngressAnnotations, "ingress-annotations", nil, "Represents the annotations that"+
		"will be added to ingress services.\nThe flag can be repeated to add several annotations.\n"+
		"For example: --ingress-annotations <key1>=<value1> --ingress-annotations <key2>=<value2>.")
	fs.StringVar(&o.DataplaneType, "dataplane", platform.DataplaneTypeEnvoy,
		"Type of dataplane, Supported values: \"envoy\", \"go\"")
	fs.Uint16Var(&o.DataplaneReplicas, "dataplane-replicas", 1, "Number of dataplanes.")
	fs.StringVar(&o.LogLevel, "log-level", "info",
		"The log level. One of fatal, error, warn, info, debug.")
}

// RequiredFlags are the names of flags that must be explicitly specified.
func (o *PeerOptions) RequiredFlags() []string {
	return []string{"name"}
}

// Run the 'deploy peer' subcommand.
func (o *PeerOptions) Run() error {
	peerDir := config.PeerDirectory(o.Name, o.Fabric, o.Path)
	if _, err := os.Stat(peerDir); err != nil {
		return fmt.Errorf("failed to open certificates folder: %w", err)
	}

	if err := o.verifyDataplaneType(o.DataplaneType); err != nil {
		return err
	}
	if err := o.verifyStartInstance(o.StartInstance); err != nil {
		return err
	}
	// Read certificates
	fabricCert, err := bootstrap.ReadCertificates(config.FabricDirectory(o.Fabric, o.Path), false)
	if err != nil {
		return fmt.Errorf("failed to read fabric certificate: %w", err)
	}

	peerCert, err := bootstrap.ReadCertificates(config.PeerDirectory(o.Name, o.Fabric, o.Path), true)
	if err != nil {
		return fmt.Errorf("failed to read peer certificate: %w", err)
	}

	caCert, err := bootstrap.CreateCACertificate()
	if err != nil {
		return fmt.Errorf("failed to create CA certificate: %w", err)
	}

	controlplaneCert, err := bootstrap.CreateControlplaneCertificate(caCert)
	if err != nil {
		return fmt.Errorf("failed to create controlplane certificates: %w", err)
	}

	dataplaneCert, err := bootstrap.CreateDataplaneCertificate(caCert)
	if err != nil {
		return fmt.Errorf("failed to create dataplane certificates: %w", err)
	}

	// Create k8s deployment YAML
	platformCfg := &platform.Config{
		Peer:                    o.Name,
		FabricCertificate:       fabricCert,
		PeerCertificate:         peerCert,
		CACertificate:           caCert,
		ControlplaneCertificate: controlplaneCert,
		DataplaneCertificate:    dataplaneCert,
		Dataplanes:              o.DataplaneReplicas,
		DataplaneType:           o.DataplaneType,
		LogLevel:                o.LogLevel,
		ContainerRegistry:       o.ContainerRegistry,
		Namespace:               o.Namespace,
		IngressType:             o.Ingress,
		IngressAnnotations:      o.IngressAnnotations,
		Tag:                     o.Tag,
	}

	// Create clusterlink instance YAML for the operator.
	if o.IngressPort != apis.DefaultExternalPort { // Set the port config only if it has changed.
		platformCfg.IngressPort = o.IngressPort
	}

	if o.StartInstance == NoStart {
		// Create a YAML file for deployment without using the operator.
		k8sConfig, err := platform.K8SConfig(platformCfg)
		if err != nil {
			return err
		}

		outPath := filepath.Join(peerDir, config.K8SYAMLFile)
		if err := os.WriteFile(outPath, k8sConfig, 0o600); err != nil {
			return fmt.Errorf("failed to write YAML file: %w", err)
		}

		return nil
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
	newImage := path.Join(o.ContainerRegistry, "cl-operator:"+o.Tag)
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

	// Create k8s secrets that contains the components certificates.
	secretConfig, err := platform.K8SCertificateConfig(platformCfg)
	if err != nil {
		return err
	}

	err = decoder.DecodeEach(
		context.Background(),
		strings.NewReader(string(secretConfig)),
		decoder.CreateIgnoreAlreadyExists(resource),
		decoder.MutateNamespace(o.Namespace),
	)
	if err != nil {
		return err
	}

	instance, err := platform.K8SClusterLinkInstanceConfig(platformCfg, "cl-instance")
	if err != nil {
		return err
	}

	// Create ClusterLink instance
	if o.StartInstance == StartAll {
		err = decoder.DecodeEach(context.Background(), strings.NewReader(string(instance)), decoder.CreateHandler(resource))
		if errors.IsAlreadyExists(err) {
			fmt.Println("CRD instance for ClusterLink (\"cl-instance\") was already exist.")
		} else if err != nil {
			return err
		}
	} else {
		// Store an example for clusterlink instance YAML.
		clOutPath := filepath.Join(peerDir, config.K8SClusterLinkInstanceYAMLFile)
		if err := os.WriteFile(clOutPath, instance, 0o600); err != nil {
			return fmt.Errorf("failed to write YAML file: %w", err)
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

// verifyDataplaneType checks if the given dataplane type is valid.
func (o *PeerOptions) verifyDataplaneType(dType string) error {
	switch dType {
	case platform.DataplaneTypeEnvoy:
		return nil
	case platform.DataplaneTypeGo:
		return nil
	default:
		return fmt.Errorf("undefined dataplane-type %s", dType)
	}
}

// verifyStartInstance checks if the given start instance is valid.
func (o *PeerOptions) verifyStartInstance(sType string) error {
	switch sType {
	case StartAll, StartOperator, NoStart:
		return nil
	default:
		return fmt.Errorf("undefined start type %s", sType)
	}
}
