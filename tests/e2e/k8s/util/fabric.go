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

package util

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"

	"github.com/clusterlink-net/clusterlink/pkg/bootstrap"
	"github.com/clusterlink-net/clusterlink/pkg/bootstrap/platform"
	"github.com/clusterlink-net/clusterlink/pkg/client"
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
}

type peer struct {
	AsyncRunner

	cluster          *KindCluster
	peerCert         *bootstrap.Certificate
	controlplaneCert *bootstrap.Certificate
	dataplaneCert    *bootstrap.Certificate
	gwctlCert        *bootstrap.Certificate
}

// CreateControlplaneCertificate creates the controlplane certificate.
func (p *peer) CreateControlplaneCertificate() {
	p.Run(func() error {
		cert, err := bootstrap.CreateControlplaneCertificate(p.cluster.Name(), p.peerCert)
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
		cert, err := bootstrap.CreateDataplaneCertificate(p.cluster.Name(), p.peerCert)
		if err != nil {
			return fmt.Errorf("cannot create dataplane certificate: %w", err)
		}

		p.dataplaneCert = cert
		return nil
	})
}

// CreateGWCTLCertificate creates the gwctl certificate.
func (p *peer) CreateGWCTLCertificate() {
	p.Run(func() error {
		cert, err := bootstrap.CreateGWCTLCertificate(p.peerCert)
		if err != nil {
			return fmt.Errorf("cannot create controlplane certificate: %w", err)
		}

		p.gwctlCert = cert
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
		cert, err := bootstrap.CreatePeerCertificate(p.cluster.Name(), f.cert)
		if err != nil {
			return fmt.Errorf("cannot create peer certificate: %w", err)
		}

		p.peerCert = cert
		p.CreateControlplaneCertificate()
		p.CreateDataplaneCertificate()
		p.CreateGWCTLCertificate()

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
			if err := p.cluster.DeleteNamespace(f.namespace); err != nil {
				return fmt.Errorf("cannot delete namespace %s: %w", f.namespace, err)
			}
		}
	}

	f.namespace = name
	return nil
}

// deployClusterLink deploys clusterlink to the given peer.
func (f *Fabric) deployClusterLink(pr *peer, cfg *PeerConfig) (*ClusterLink, error) {
	if f.namespace == "" {
		return nil, fmt.Errorf("namespace not set")
	}

	k8sYAML, err := f.generateK8SYAML(pr, cfg)
	if err != nil {
		return nil, fmt.Errorf("cannot generate k8s yaml: %w", err)
	}

	if err := pr.cluster.CreateFromYAML(k8sYAML, f.namespace); err != nil {
		return nil, fmt.Errorf("cannot create k8s objects: %w", err)
	}

	var service v1.Service
	err = pr.cluster.resources.Get(context.Background(), "cl-dataplane", f.namespace, &service)
	if err != nil {
		return nil, fmt.Errorf("error getting dataplane service: %w", err)
	}

	port := uint16(service.Spec.Ports[0].NodePort)

	cert := pr.gwctlCert
	certificate, err := tls.X509KeyPair(cert.RawCert(), cert.RawKey())
	if err != nil {
		return nil, fmt.Errorf("cannot parse gwctl certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(pr.peerCert.RawCert()) {
		return nil, fmt.Errorf("unable to parse peer certificate")
	}

	c := client.New(pr.cluster.IP(), port, &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{certificate},
		RootCAs:      caCertPool,
		ServerName:   pr.cluster.Name(),
	})

	clink := &ClusterLink{
		cluster:   pr.cluster,
		namespace: f.namespace,
		client:    c,
		port:      port,
	}

	// wait for default service account to be created
	for t := time.Now(); time.Since(t) < time.Second*30; time.Sleep(time.Millisecond * 100) {
		err = pr.cluster.resources.Get(context.Background(), "default", f.namespace, &v1.ServiceAccount{})
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("error getting default service account: %w", err)
	}

	if err := clink.WaitForControlplaneAPI(); err != nil {
		return nil, fmt.Errorf("error waiting for controlplane API server: %w", err)
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

// NewFabric returns a new empty fabric.
func NewFabric() (*Fabric, error) {
	cert, err := bootstrap.CreateFabricCertificate()
	if err != nil {
		return nil, fmt.Errorf("cannot create fabric certificate: %w", err)
	}

	return &Fabric{cert: cert}, nil
}
