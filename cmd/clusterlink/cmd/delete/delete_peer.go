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

package deletion

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"

	// Importing this package for initializing the OIDC authentication plugin for client-go.
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	"github.com/clusterlink-net/clusterlink/cmd/cl-controlplane/app"
	configfiles "github.com/clusterlink-net/clusterlink/config"
	apis "github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/bootstrap/platform"
)

// PeerOptions contains everything necessary to create and run a 'delete peer' subcommand.
type PeerOptions struct {
	// Name of the peer to delete.
	Name string
	// Namespace where the ClusterLink components and secrets are deployed.
	Namespace string
}

// NewCmdDeletePeer returns a cobra.Command to run the 'delete peer' subcommand.
func NewCmdDeletePeer() *cobra.Command {
	opts := &PeerOptions{}

	cmd := &cobra.Command{
		Use:   "peer",
		Short: "Delete ClusterLink components from the cluster.",
		Long:  `Delete ClusterLink components from the cluster.`,

		RunE: func(_ *cobra.Command, _ []string) error {
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
	fs.StringVar(&o.Namespace, "namespace", app.SystemNamespace,
		"Namespace where the ClusterLink secrets are deployed.")
}

// RequiredFlags are the names of flags that must be explicitly specified.
func (o *PeerOptions) RequiredFlags() []string {
	return []string{"name"}
}

// Run the 'delete peer' subcommand.
func (o *PeerOptions) Run() error {
	// Create k8s resources
	cfg, err := ctrl.GetConfig()
	if err != nil {
		return err
	}

	resource, err := resources.New(cfg)
	if err != nil {
		return err
	}

	if err := apis.AddToScheme(resource.GetScheme()); err != nil {
		return err
	}

	// List all instances of the CR
	instList := &apis.InstanceList{}
	err = resource.List(context.Background(), instList)
	if client.IgnoreNotFound(err) != nil && !meta.IsNoMatchError(err) {
		return fmt.Errorf("unable to get instances list: %w", err)
	}

	// Delete each instance.
	for i := range instList.Items {
		err = resource.Delete(context.Background(), &instList.Items[i])
		if err != nil {
			fmt.Printf("Error deleting instance %s in namespace %s: %v\n", instList.Items[i].Name, instList.Items[i].Namespace, err)
			continue
		}

		if err := o.waitForInstanceDeletion(resource, &instList.Items[i]); err != nil {
			return err
		}
	}

	// Delete operator
	if err := o.deleteDir("operator/manager/*", resource); err != nil {
		return err
	}

	if err := o.deleteDir("operator/rbac/*", resource); err != nil {
		return err
	}

	if err := o.deleteDir("crds/*", resource); err != nil {
		return err
	}

	// Delete secrets
	platformCfg := &platform.Config{
		Peer:      o.Name,
		Namespace: o.Namespace,
	}

	secretConfig, err := platform.K8SEmptyCertificateConfig(platformCfg)
	if err != nil {
		return err
	}

	err = decoder.DecodeEach(
		context.Background(),
		strings.NewReader(string(secretConfig)),
		decoder.DeleteIgnoreNotFound(resource),
		decoder.MutateNamespace(o.Namespace),
	)
	if err != nil {
		return fmt.Errorf("fail to delete certificate secrets %w", err)
	}

	return nil
}

// deleteDir deletes K8s yaml from a directory.
func (o *PeerOptions) deleteDir(dir string, resource *resources.Resources) error {
	err := decoder.DecodeEachFile(context.Background(), configfiles.ConfigFiles, dir, decoder.DeleteIgnoreNotFound(resource))
	if err != nil {
		return fmt.Errorf("failed to delete directory '%s': %w", dir, err)
	}

	return nil
}

// waitForInstanceDeletion waits for the deletion of the instance to complete.
func (o *PeerOptions) waitForInstanceDeletion(resource *resources.Resources, inst *apis.Instance) error {
	for t := time.Now(); time.Since(t) < time.Second*60; time.Sleep(time.Millisecond * 500) {
		instance := &apis.Instance{}
		err := resource.Get(context.Background(), inst.Name, inst.Namespace, instance)
		if err != nil {
			if errors.IsNotFound(err) {
				return nil // Instance deletion completed
			}
			return err // Error occurred during deletion
		}
	}

	return fmt.Errorf("timeout exceeded while waiting for instance deletion")
}
