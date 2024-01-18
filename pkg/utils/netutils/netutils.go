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

/**********************************************************/
/* Package netutils contain helper functions for network
/* connection
/**********************************************************/

package netutils

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	dnsPattern = `^[a-zA-Z0-9-]{1,63}(\.[a-zA-Z0-9-]{1,63})*$`
	dnsRegex   = regexp.MustCompile(dnsPattern)
)

// GetConnIP returns the connection's local IP and port.
func GetConnIP(c net.Conn) (string, string) { //nolint:gocritic // unnamedResult: named in comment
	s := strings.Split(c.LocalAddr().String(), ":")
	ip := s[0]
	port := s[1]
	return ip, port
}

// IsIP returns true if the input is valid IPv4 or IPv6.
func IsIP(str string) bool {
	return net.ParseIP(str) != nil
}

// IsDNS returns true if the input is valid DNS.
func IsDNS(s string) bool {
	return dnsRegex.MatchString(s)
}

// Start HTTP server.
func StartHTTPServer(ip string, handler http.Handler) {
	s := CreateDefaultResilientHTTPServer(ip, handler)
	log.Fatal(s.ListenAndServe())
}

func StartMTLSServer(ip, certca, certificate, key string, handler http.Handler) {
	// Create the TLS Config with the CA pool and enable Client certificate validation
	caCert, err := os.ReadFile(certca)
	if err != nil {
		log.Fatal("read CA certificate:", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := ConfigureSafeTLSConfig()
	tlsConfig.ClientCAs = caCertPool
	tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert

	// Create a Server instance to listen on port 443 with the TLS config
	server := CreateResilientHTTPServer(ip, handler, tlsConfig, nil, nil, nil)
	log.Fatal(server.ListenAndServeTLS(certificate, key))
}

// ConfigureSafeTLSConfig creates a default tls.Config that's considered "safe" for HTTP serving.
func ConfigureSafeTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		// Causes servers to use Go's default ciphersuite preferences,
		// which are tuned to avoid attacks. Does nothing on clients.
		PreferServerCipherSuites: true,
		// Only use curves which have assembly implementations
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
			tls.X25519, // Go 1.8 only
		},
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305, // Go 1.8 only
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,   // Go 1.8 only
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}
}

// CreateDefaultResilientHTTPServer returns an http.Server configured with default configuration.
func CreateDefaultResilientHTTPServer(addr string, mux http.Handler) *http.Server {
	return CreateResilientHTTPServer(addr, mux, ConfigureSafeTLSConfig(), nil, nil, nil)
}

// CreateResilientHTTPServer returns an http.Server configured with the timeouts provided.
func CreateResilientHTTPServer(addr string, mux http.Handler, tlsConfig *tls.Config,
	headerReadTimeout, writeTimeout, idleTimeout *time.Duration,
) *http.Server {
	const (
		defaultReadHeaderTimeout = 2 * time.Second
		defaultWriteTimeout      = 2 * time.Second
		defaultIdleTimeout       = 120 * time.Second
		defaultMaxHeaderBytes    = 10 * 1024
	)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		ReadTimeout:       time.Duration(0), // use header timeout only
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
		MaxHeaderBytes:    defaultMaxHeaderBytes,
	}

	if headerReadTimeout != nil {
		srv.ReadHeaderTimeout = *headerReadTimeout
	}
	if writeTimeout != nil {
		srv.WriteTimeout = *writeTimeout
	}
	if idleTimeout != nil {
		srv.IdleTimeout = *idleTimeout
	}
	if tlsConfig != nil {
		srv.TLSConfig = tlsConfig
	}
	return srv
}
