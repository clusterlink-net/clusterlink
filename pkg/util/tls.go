package util

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// ReadTLSFiles reads the given TLS-related files.
func ReadTLSFiles(ca, cert, key string) (*RawCertData, error) {
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

	return &RawCertData{
		CA:          rawCA,
		Certificate: rawCertificate,
		PrivateKey:  rawPrivateKey,
	}, nil
}

// RawCertData contains raw cert/ca/key data.
type RawCertData struct {
	// CA holds the raw bytes of the certificate authority.
	CA []byte
	// Certificate holds the raw bytes of the TLS certificate.
	Certificate []byte
	// PrivateKey holds the raw bytes of the TLS private key.
	PrivateKey []byte
}

// Parse the raw byte arrays of this RawCertData.
func (d *RawCertData) Parse() (*ParsedCertData, error) {
	certificate, err := tls.X509KeyPair(d.Certificate, d.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("unable to parse certificate keypair: %v", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(d.CA) {
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
