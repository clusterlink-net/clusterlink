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
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// Return connection IP and port
func GetConnIp(c net.Conn) (string, string) {
	s := strings.Split(c.LocalAddr().String(), ":")
	ip := s[0]
	port := s[1]
	return ip, port
}

// Start HTTP server
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

// ConfigureSafeTLSConfig creates a default tls.Config that's considered "safe" for HTTP serving
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

// CreateDefaultResilientHTTPServer returns an http.Server configured with default configuration
func CreateDefaultResilientHTTPServer(addr string, mux http.Handler) *http.Server {
	return CreateResilientHTTPServer(addr, mux, ConfigureSafeTLSConfig(), nil, nil, nil)
}

// CreateResilientHTTPServer returns an http.Server configured with the timeouts provided
func CreateResilientHTTPServer(addr string, mux http.Handler, tlsConfig *tls.Config,
	headerReadTimeout, writeTimeout, idleTimeout *time.Duration) *http.Server {

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
