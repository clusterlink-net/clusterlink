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
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	mathrand "math/rand"
	"time"
)

type certificate struct {
	parent *certificate

	cert *x509.Certificate
	key  *rsa.PrivateKey

	certPEM []byte
	keyPEM  []byte
}

// certificateConfig holds a configuration for creating a new signed certificate.
type certificateConfig struct {
	// Name is the common name that will be used in the certificate.
	Name string

	// IsCA should be set to true if creating a CA certificate.
	IsCA bool
	// IsServer should be set to true if certificate should allow server authentication.
	IsServer bool
	// IsServer should be set to true if certificate should allow client authentication.
	IsClient bool
	// DNSNames are the DNS names to be set in the certificate.
	// For a CA certificate, these are the permitted DNS names.
	DNSNames []string

	// Parent certificate that will sign the certificate.
	// If nil, certificate will self-sign.
	Parent *certificate
}

// createCertificate creates a signed certificate.
func createCertificate(config *certificateConfig) (*certificate, error) {
	// generate key pair
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	// RNG for generating certificate serial number.
	//#nosec G404 -- certificate serial number does not need secure random
	rng := mathrand.New(mathrand.NewSource(time.Now().UTC().UnixNano()))

	// create certificate
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(rng.Int63()),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		IsCA:         config.IsCA,
		Subject:      pkix.Name{CommonName: config.Name},
	}

	if config.IsCA {
		cert.BasicConstraintsValid = true
		cert.PermittedDNSDomains = config.DNSNames
		cert.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageCRLSign
	} else {
		cert.DNSNames = config.DNSNames
	}

	if config.IsServer {
		cert.ExtKeyUsage = append(cert.ExtKeyUsage, x509.ExtKeyUsageServerAuth)
	}

	if config.IsClient {
		cert.ExtKeyUsage = append(cert.ExtKeyUsage, x509.ExtKeyUsageClientAuth)
	}

	var ca *x509.Certificate
	var caKey *rsa.PrivateKey

	if config.Parent != nil {
		ca = config.Parent.cert
		caKey = config.Parent.key
	} else {
		ca = cert
		caKey = key
	}

	// sign certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &key.PublicKey, caKey)
	if err != nil {
		return nil, err
	}

	// PEM encode certificate
	certPEM := new(bytes.Buffer)
	err = pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	if err != nil {
		return nil, err
	}

	// PEM encode private key
	keyPEM := new(bytes.Buffer)
	err = pem.Encode(keyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	if err != nil {
		return nil, err
	}

	signedCert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, err
	}

	return &certificate{
		parent:  config.Parent,
		cert:    signedCert,
		key:     key,
		certPEM: certPEM.Bytes(),
		keyPEM:  keyPEM.Bytes(),
	}, nil
}

// certificateFromRaw initializes a certificate from raw data.
func certificateFromRaw(certPEM, keyPEM []byte) (*certificate, error) {
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return nil, fmt.Errorf("key is not in PEM format")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	block, _ = pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("certificate is not in PEM format")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	return &certificate{
		parent:  nil,
		cert:    cert,
		key:     key,
		certPEM: certPEM,
		keyPEM:  keyPEM,
	}, nil
}
