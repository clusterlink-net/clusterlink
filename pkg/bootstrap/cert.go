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

package bootstrap

import (
	"os"
	"path/filepath"

	"github.com/clusterlink-net/clusterlink/cmd/clusterlink/config"
	cpapi "github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
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

// CreateCACertificate creates a site CA certificate for controlplane <-> dataplane trust.
func CreateCACertificate() (*Certificate, error) {
	cert, err := createCertificate(&certificateConfig{
		Name: "cl-ca",
		IsCA: true,
	})
	if err != nil {
		return nil, err
	}

	return &Certificate{cert: cert}, nil
}

// CreateControlplaneCertificate creates a controlplane certificate.
func CreateControlplaneCertificate(caCert *Certificate) (*Certificate, error) {
	cert, err := createCertificate(&certificateConfig{
		Parent:   caCert.cert,
		Name:     cpapi.Name,
		IsServer: true,
		DNSNames: []string{cpapi.Name},
	})
	if err != nil {
		return nil, err
	}

	return &Certificate{cert: cert}, nil
}

// CreateDataplaneCertificate creates a dataplane certificate.
func CreateDataplaneCertificate(caCert *Certificate) (*Certificate, error) {
	cert, err := createCertificate(&certificateConfig{
		Parent:   caCert.cert,
		Name:     dpapi.Name,
		IsClient: true,
		DNSNames: []string{dpapi.Name},
	})
	if err != nil {
		return nil, err
	}

	return &Certificate{cert: cert}, nil
}

// CreatePeerCertificate creates a peer certificate.
func CreatePeerCertificate(peer string, fabricCert *Certificate) (*Certificate, error) {
	cert, err := createCertificate(&certificateConfig{
		Parent:   fabricCert.cert,
		Name:     peer,
		IsServer: true,
		IsClient: true,
		DNSNames: []string{peer},
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

// ReadCertificates read certificate and key from folder.
func ReadCertificates(dir string, withKey bool) (*Certificate, error) {
	// Read certificate
	rawCert, err := os.ReadFile(filepath.Join(dir, config.CertificateFileName))
	if err != nil {
		return nil, err
	}

	var rawFabricKey []byte
	if withKey {
		// Read key
		rawFabricKey, err = os.ReadFile(filepath.Join(dir, config.PrivateKeyFileName))
		if err != nil {
			return nil, err
		}
	}

	cert, err := CertificateFromRaw(rawCert, rawFabricKey)
	if err != nil {
		return nil, err
	}

	return cert, nil
}
