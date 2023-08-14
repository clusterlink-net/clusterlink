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
	// s := CreateDefaultResilientHTTPServer(ip, handler)
	// log.Fatal(s.ListenAndServe())
	// Commenting the Resilient server until we identify the issue & fix it
	log.Fatal(http.ListenAndServe(ip, handler))
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
	//server := CreateResilientHTTPServer(ip, handler, tlsConfig, 0, 0, 0, 0)
	server := &http.Server{
		Addr:      ip,
		TLSConfig: tlsConfig,
		Handler:   handler,
	}
	log.Fatal(server.ListenAndServeTLS(certificate, key))
}

// ConfigureSafeTLSConfig creates a default tls.Config that's considered "safe" for HTTP serving
func ConfigureSafeTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion:               tls.VersionTLS12,
		PreferServerCipherSuites: true,
	}
}

// CreateDefaultResilientHTTPServer returns an http.Server configured with default configuration
func CreateDefaultResilientHTTPServer(addr string, mux http.Handler) *http.Server {
	return CreateResilientHTTPServer(addr, mux, ConfigureSafeTLSConfig(), 0, 0, 0, 0)
}

// CreateResilientHTTPServer returns an http.Server configured with the timeouts provided
func CreateResilientHTTPServer(addr string, mux http.Handler, tlsConfig *tls.Config,
	readTimeout, writeTimeout, idleTimeout, requestTimeout time.Duration) *http.Server {

	const (
		defaultReadTimeout       = 2 * time.Second
		defaultReadHeaderTimeout = 1 * time.Second
		defaultWriteTimeout      = 2 * time.Second
		defaultIdleTimeout       = 120 * time.Second
		defaultRequestTimeout    = 1 * time.Second
		defaultMaxHeaderBytes    = 10 * 1024
	)

	if requestTimeout <= 0 {
		requestTimeout = defaultRequestTimeout
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           http.TimeoutHandler(mux, requestTimeout, "request timeout\n"),
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		ReadTimeout:       defaultReadTimeout,
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
		MaxHeaderBytes:    defaultMaxHeaderBytes,
	}

	if readTimeout != 0 {
		srv.ReadTimeout = readTimeout
	}
	if writeTimeout != 0 {
		srv.WriteTimeout = writeTimeout
	}
	if idleTimeout != 0 {
		srv.IdleTimeout = idleTimeout
	}
	if tlsConfig != nil {
		srv.TLSConfig = tlsConfig
	}
	return srv
}
