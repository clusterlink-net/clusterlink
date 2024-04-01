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

package bootstrap

import (
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
	dpapi "github.com/clusterlink-net/clusterlink/pkg/dataplane/api"
)

// Certificate represents a clusterlink certificate.
type Certificate struct {
	cert *certificate
}

// RawCert returns the raw certificate bytes.
func (c *Certificate) RawCert() []byte {
	chain := c.cert.certPEM
	if c.cert.parent != nil {
		chain = append(chain, c.cert.parent.certPEM...)
	}
	return chain
}

// RawKey returns the raw private key bytes.
func (c *Certificate) RawKey() []byte {
	return c.cert.keyPEM
}

// CreateFabricCertificate creates a clusterlink fabric (root) certificate.
func CreateFabricCertificate(name string) (*Certificate, error) {
	cert, err := createCertificate(&certificateConfig{
		Name: name,
		IsCA: true,
	})
	if err != nil {
		return nil, err
	}

	return &Certificate{cert: cert}, nil
}

// CreatePeerCertificate creates a peer certificate.
func CreatePeerCertificate(name string, fabricCert *Certificate) (*Certificate, error) {
	cert, err := createCertificate(&certificateConfig{
		Parent:   fabricCert.cert,
		Name:     name,
		IsCA:     true,
		DNSNames: []string{name},
	})
	if err != nil {
		return nil, err
	}

	return &Certificate{cert: cert}, nil
}

// CreatePeerCertificate creates a controlplane certificate.
func CreateControlplaneCertificate(peer string, peerCert *Certificate) (*Certificate, error) {
	cert, err := createCertificate(&certificateConfig{
		Parent:   peerCert.cert,
		Name:     "cl-controlplane",
		IsServer: true,
		IsClient: true,
		DNSNames: []string{peer, api.GRPCServerName(peer)},
	})
	if err != nil {
		return nil, err
	}

	return &Certificate{cert: cert}, nil
}

// CreatePeerCertificate creates a dataplane certificate.
func CreateDataplaneCertificate(peer string, peerCert *Certificate) (*Certificate, error) {
	cert, err := createCertificate(&certificateConfig{
		Parent:   peerCert.cert,
		Name:     "cl-dataplane",
		IsServer: true,
		IsClient: true,
		DNSNames: []string{dpapi.DataplaneServerName(peer)},
	})
	if err != nil {
		return nil, err
	}

	return &Certificate{cert: cert}, nil
}

// CreatePeerCertificate creates a gwctl certificate.
func CreateGWCTLCertificate(peerCert *Certificate) (*Certificate, error) {
	cert, err := createCertificate(&certificateConfig{
		Parent:   peerCert.cert,
		Name:     "gwctl",
		IsClient: true,
	})
	if err != nil {
		return nil, err
	}

	return &Certificate{cert: cert}, nil
}

// CertificateFromRaw initializes a certificate from raw data.
func CertificateFromRaw(rawCert, rawKey []byte) (*Certificate, error) {
	cert, err := certificateFromRaw(rawCert, rawKey)
	if err != nil {
		return nil, err
	}

	return &Certificate{cert: cert}, nil
}
