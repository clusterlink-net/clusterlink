package util

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// ParseTLSFiles parses the given TLS-related files.
func ParseTLSFiles(ca, cert, key string) (*ParsedCertData, error) {
	rawCA, err := os.ReadFile(ca)
	if err != nil {
		return nil, fmt.Errorf("unable to read CA file '%s': %v", ca, err)
	}

	rawCertificate, err := os.ReadFile(cert)
	if err != nil {
		return nil, fmt.Errorf("unable to read certificate file: %v", err)
	}

	rawPrivateKey, err := os.ReadFile(key)
	if err != nil {
		return nil, fmt.Errorf("unable to read private key file: %v", err)
	}

	certificate, err := tls.X509KeyPair(rawCertificate, rawPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("unable to parse certificate keypair: %v", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(rawCA) {
		return nil, fmt.Errorf("unable to parse CA file")
	}

	x509cert, err := x509.ParseCertificate(certificate.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("unable to parse x509 certificate: %v", err)
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
		Certificates: []tls.Certificate{c.certificate},
		ClientCAs:    c.ca,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}
}

// ClientConfig return a TLS configuration for a client.
func (c *ParsedCertData) ClientConfig(sni string) *tls.Config {
	return &tls.Config{
		Certificates: []tls.Certificate{c.certificate},
		RootCAs:      c.ca,
		ServerName:   sni,
	}
}

// DNSNames returns the certificate DNS names.
func (c *ParsedCertData) DNSNames() []string {
	return c.x509cert.DNSNames
}

// CommonName returns the certificate common name.
func (c *ParsedCertData) CommonName() string {
	return c.x509cert.Subject.CommonName
}
