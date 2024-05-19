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

// Copyright (c) 2022 The ClusterLink Authors.
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

// Copyright (C) The ClusterLink Authors.
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

package tls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// ParseFiles parses the given TLS-related files.
func ParseFiles(ca, cert, key string) (*ParsedCertData, error) {
	rawCA, err := os.ReadFile(ca)
	if err != nil {
		return nil, fmt.Errorf("unable to read CA file '%s': %w", ca, err)
	}

	rawCertificate, err := os.ReadFile(cert)
	if err != nil {
		return nil, fmt.Errorf("unable to read certificate file: %w", err)
	}

	rawPrivateKey, err := os.ReadFile(key)
	if err != nil {
		return nil, fmt.Errorf("unable to read private key file: %w", err)
	}

	certificate, err := tls.X509KeyPair(rawCertificate, rawPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("unable to parse certificate keypair: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(rawCA) {
		return nil, fmt.Errorf("unable to parse CA file")
	}

	x509cert, err := x509.ParseCertificate(certificate.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("unable to parse x509 certificate: %w", err)
	}

	return &ParsedCertData{
		certificate: certificate,
		ca:          caCertPool,
		x509cert:    x509cert,
	}, nil
}

// ParsedCertData contains a parsed CA and TLS certificate.
type ParsedCertData struct {
	certificate tls.Certificate
	ca          *x509.CertPool
	x509cert    *x509.Certificate
}

// ServerConfig return a TLS configuration for a server.
func (c *ParsedCertData) ServerConfig() *tls.Config {
	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{c.certificate},
		ClientCAs:    c.ca,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}
}

// ClientConfig return a TLS configuration for a client.
func (c *ParsedCertData) ClientConfig(sni string) *tls.Config {
	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{c.certificate},
		RootCAs:      c.ca,
		ServerName:   sni,
	}
}

// DNSNames returns the certificate DNS names.
func (c *ParsedCertData) DNSNames() []string {
	return c.x509cert.DNSNames
}
