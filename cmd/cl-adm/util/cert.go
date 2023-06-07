package util

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
	"os"
	"time"
)

// CertificateConfig holds a configuration for creating a new signed certificate.
type CertificateConfig struct {
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

	// CAPath is the path to the CA certificate that will sign the certificate.
	// If empty, certificate will self-sign.
	CAPath string
	// CAKeyPath is the path to the private key of the CA that will sign the certificate.
	CAKeyPath string

	// CertOutPath is the path where the certificate will be saved to.
	CertOutPath string
	// CertOutPath is the path where the private key will be saved to.
	PrivateKeyOutPath string
}

// CreateCertificate creates a signed certificate.
func CreateCertificate(config *CertificateConfig) error {
	// generate key pair
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
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

	if config.CAPath != "" {
		// load CA certificate
		rawCA, err := os.ReadFile(config.CAPath)
		if err != nil {
			return err
		}

		block, _ := pem.Decode(rawCA)
		if block == nil {
			return fmt.Errorf("CA certificate file is not in PEM format")
		}

		ca, err = x509.ParseCertificate(block.Bytes)
		if err != nil {
			return err
		}

		// load CA key
		rawCAKey, err := os.ReadFile(config.CAKeyPath)
		if err != nil {
			return err
		}

		block, _ = pem.Decode(rawCAKey)
		if block == nil {
			return fmt.Errorf("CA key file is not in PEM format")
		}

		caKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return err
		}
	} else {
		ca = cert
		caKey = key
	}

	// sign certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &key.PublicKey, caKey)
	if err != nil {
		return err
	}

	// PEM encode the private key
	keyPEM := new(bytes.Buffer)
	err = pem.Encode(keyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	if err != nil {
		return err
	}

	// PEM encode the certificate
	certPEM := new(bytes.Buffer)
	err = pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	if err != nil {
		return err
	}

	if ca != cert {
		// append CA certificate
		err = pem.Encode(certPEM, &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: ca.Raw,
		})
		if err != nil {
			return err
		}
	}

	// save private key to file
	err = os.WriteFile(config.PrivateKeyOutPath, keyPEM.Bytes(), 0600)
	if err != nil {
		return err
	}

	// save certificate to file
	err = os.WriteFile(config.CertOutPath, certPEM.Bytes(), 0600)
	if err != nil {
		return err
	}

	return nil
}
