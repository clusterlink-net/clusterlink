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

package util

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"

	"github.com/clusterlink-net/clusterlink/cmd/clusterlink/config"
	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/bootstrap"
	"github.com/clusterlink-net/clusterlink/pkg/bootstrap/platform"
	"github.com/clusterlink-net/clusterlink/pkg/operator/controller"
)

// PeerConfig is a peer configuration.
type PeerConfig struct {
	// DataplaneType is the dataplane type (envoy / go).
	DataplaneType string
	// Dataplanes is the number of dataplane instances.
	Dataplanes uint16
	// ControlplanePersistency should be true if controlplane should use persistent storage.
	ControlplanePersistency bool
	// ExpectLargeDataplaneTraffic hints that a large amount of dataplane traffic is expected.
	ExpectLargeDataplaneTraffic bool
	// DeployWithOperator deploys clusterlink using an operator.
	DeployWithOperator bool
}

type peer struct {
	AsyncRunner

	cluster          *KindCluster
	fabricCert       *bootstrap.Certificate
	peerCert         *bootstrap.Certificate
	caCert           *bootstrap.Certificate
	controlplaneCert *bootstrap.Certificate
	dataplaneCert    *bootstrap.Certificate
}

// CreatePeerCertificate creates the peer certificate.
func (p *peer) CreatePeerCertificate() {
	p.Run(func() error {
		cert, err := bootstrap.CreatePeerCertificate(p.cluster.Name(), p.fabricCert)
		if err != nil {
			return fmt.Errorf("cannot create peer certificate: %w", err)
		}

		p.peerCert = cert
		return nil
	})
}

// CreateCACertificate creates the site CA certificate.
func (p *peer) CreateCACertificate() error {
	cert, err := bootstrap.CreateCACertificate()
	if err != nil {
		return fmt.Errorf("cannot create site CA certificate: %w", err)
	}

	p.caCert = cert
	return nil
}

// CreateControlplaneCertificate creates the controlplane certificate.
func (p *peer) CreateControlplaneCertificate() {
	p.Run(func() error {
		cert, err := bootstrap.CreateControlplaneCertificate(p.caCert)
		if err != nil {
			return fmt.Errorf("cannot create controlplane certificate: %w", err)
		}

		p.controlplaneCert = cert
		return nil
	})
}

// CreateDataplaneCertificate creates the dataplane certificate.
func (p *peer) CreateDataplaneCertificate() {
	p.Run(func() error {
		cert, err := bootstrap.CreateDataplaneCertificate(p.caCert)
		if err != nil {
			return fmt.Errorf("cannot create dataplane certificate: %w", err)
		}

		p.dataplaneCert = cert
		return nil
	})
}

// Fabric represents a collection of clusterlinks.
type Fabric struct {
	AsyncRunner

	cert          *bootstrap.Certificate
	peers         []*peer
	namespace     string
	baseNamespace string
}

// CreatePeer creates certificates for a new peer on a given kind cluster.
func (f *Fabric) CreatePeer(cluster *KindCluster) {
	p := &peer{cluster: cluster}
	f.peers = append(f.peers, p)
	f.Run(func() error {
		p.fabricCert = f.cert
		p.CreatePeerCertificate()
		if err := p.CreateCACertificate(); err != nil {
			return err
		}
		p.CreateControlplaneCertificate()
		p.CreateDataplaneCertificate()

		return p.Wait()
	})
}

// SwitchToNewNamespace creates a new namespace to be used for deploying clusterlink.
// It also updates the current nodeport value.
func (f *Fabric) SwitchToNewNamespace(name string, appendName bool) error {
	if appendName {
		name = f.baseNamespace + "-" + name
	} else {
		f.baseNamespace = name
	}

	// create new namespace
	for _, p := range f.peers {
		if err := p.cluster.CreateNamespace(name); err != nil {
			return fmt.Errorf("cannot create namespace %s: %w", name, err)
		}
	}

	if f.namespace != "" {
		// delete old namespace
		for _, p := range f.peers {
			// delete imports to avoid slowing down upcoming tests
			var imports v1alpha1.ImportList
			if err := p.cluster.Resources().List(context.Background(), &imports); err != nil {
				return err
			}

			for i := range imports.Items {
				err := p.cluster.Resources().Delete(context.Background(), &(imports.Items[i]))
				if err != nil {
					return err
				}
			}

			if err := p.cluster.DeleteNamespace(f.namespace); err != nil {
				return fmt.Errorf("cannot delete namespace %s: %w", f.namespace, err)
			}
		}
	}

	f.namespace = name
	return nil
}

var deployFunc func(target *peer, cfg *PeerConfig) error

// deployUsingOperator deploys ClusterLink using operator.
func (f *Fabric) deployUsingOperator(target *peer, cfg *PeerConfig) error {
	instanceName := "cl-instance" + f.namespace

	// Create ClusterLink instance
	instance, err := f.generateClusterlinkInstance(instanceName, target, cfg)
	if err != nil {
		return fmt.Errorf("cannot generate ClusterLink instance: %w", err)
	}

	if err := target.cluster.CreateFromYAML(instance, controller.OperatorNamespace); err != nil {
		return fmt.Errorf("cannot create k8s objects: %w", err)
	}

	// Create k8s secrets
	secretsYAML, err := f.generateClusterlinkSecrets(target)
	if err != nil {
		return fmt.Errorf("cannot generate ClusterLink secrets: %w", err)
	}

	if err := target.cluster.CreateFromYAML(secretsYAML, f.namespace); err != nil {
		return fmt.Errorf("cannot create k8s objects: %w", err)
	}

	// wait for operator to be ready
	dep := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cl-operator-controller-manager",
			Namespace: controller.OperatorNamespace,
		},
	}
	waitCon := conditions.New(target.cluster.resources).DeploymentConditionMatch(
		&dep, appsv1.DeploymentAvailable, v1.ConditionTrue)
	err = wait.For(waitCon, wait.WithTimeout(time.Second*60))
	if err != nil {
		return fmt.Errorf("failed waiting for operator to be ready: %w", err)
	}

	// wait for instance to be ready
	err = wait.For(func(ctx context.Context) (bool, error) {
		var inst v1alpha1.Instance
		err := target.cluster.Resources().Get(ctx, instanceName, controller.OperatorNamespace, &inst)
		if err != nil {
			return false, err
		}

		if c, ok := inst.Status.Controlplane.Conditions[string(v1alpha1.DeploymentReady)]; ok {
			if c.Status == metav1.ConditionTrue {
				return true, nil
			}
		}

		return false, nil
	}, wait.WithTimeout(time.Second*60))
	if err != nil {
		return fmt.Errorf("failed waiting for instance to be ready: %w", err)
	}

	return nil
}

// deployUsingK8sYAML deploys ClusterLink using K8s yaml.
func (f *Fabric) deployUsingK8sYAML(target *peer, cfg *PeerConfig) error {
	k8sYAML, err := f.generateK8SYAML(target, cfg)
	if err != nil {
		return fmt.Errorf("cannot generate k8s yaml: %w", err)
	}

	if err := target.cluster.CreateFromYAML(k8sYAML, f.namespace); err != nil {
		return fmt.Errorf("cannot create k8s objects: %w", err)
	}

	return nil
}

// deployClusterLink deploys clusterlink to the given peer.
func (f *Fabric) deployClusterLink(target *peer, cfg *PeerConfig) (*ClusterLink, error) {
	var err error
	if f.namespace == "" {
		return nil, fmt.Errorf("namespace not set")
	}

	svcNodePort := "cl-dataplane"
	if cfg.DeployWithOperator {
		svcNodePort = controller.IngressName
		deployFunc = f.deployUsingOperator
	} else {
		deployFunc = f.deployUsingK8sYAML
	}

	if err := deployFunc(target, cfg); err != nil {
		return nil, err
	}

	// Wait for dataplane will be ready.
	dep := appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "cl-dataplane", Namespace: f.namespace}}
	waitCon := conditions.New(target.cluster.resources).DeploymentConditionMatch(&dep, appsv1.DeploymentAvailable, v1.ConditionTrue)
	err = wait.For(waitCon, wait.WithTimeout(time.Second*60))
	if err != nil {
		return nil, err
	}

	var service v1.Service
	err = target.cluster.resources.Get(context.Background(), svcNodePort, f.namespace, &service)
	if err != nil {
		return nil, fmt.Errorf("error getting dataplane service: %w", err)
	}

	port := uint16(service.Spec.Ports[0].NodePort)

	clink := &ClusterLink{
		cluster:   target.cluster,
		namespace: f.namespace,
		port:      port,
	}

	// wait for default service account to be created
	for t := time.Now(); time.Since(t) < time.Second*30; time.Sleep(time.Millisecond * 100) {
		err = target.cluster.resources.Get(context.Background(), "default", f.namespace, &v1.ServiceAccount{})
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("error getting default service account: %w", err)
	}

	return clink, nil
}

// DeployClusterlinks deploys clusterlink to <peerCount> clusters.
func (f *Fabric) DeployClusterlinks(peerCount uint8, cfg *PeerConfig) ([]*ClusterLink, error) {
	if int(peerCount) > len(f.peers) {
		return nil, fmt.Errorf(
			"cannot deploy %d clusterlinks to %d clusters",
			peerCount, len(f.peers))
	}

	if cfg == nil {
		// default config
		cfg = &PeerConfig{
			DataplaneType: platform.DataplaneTypeEnvoy,
			Dataplanes:    1,
		}
	}

	clusterlinks := make([]*ClusterLink, peerCount)
	for i := uint8(0); i < peerCount; i++ {
		f.Run(func(i uint8) func() error {
			return func() error {
				cl, err := f.deployClusterLink(f.peers[i], cfg)
				if err != nil {
					return fmt.Errorf(
						"cannot deploy clusterlink to cluster %s: %w",
						f.peers[i].cluster.Name(), err)
				}

				clusterlinks[i] = cl
				return nil
			}
		}(i))
	}

	if err := f.Wait(); err != nil {
		return nil, err
	}

	return clusterlinks, nil
}

// PeerKindCluster returns the peer kind cluster.
func (f *Fabric) PeerKindCluster(num int) *KindCluster {
	return f.peers[num].cluster
}

// Namespace returns fabric namespace.
func (f *Fabric) Namespace() string {
	return f.namespace
}

// NewFabric returns a new empty fabric.
func NewFabric() (*Fabric, error) {
	cert, err := bootstrap.CreateFabricCertificate(config.DefaultFabric)
	if err != nil {
		return nil, fmt.Errorf("cannot create fabric certificate: %w", err)
	}

	return &Fabric{cert: cert}, nil
}
