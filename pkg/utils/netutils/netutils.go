/**********************************************************/
/* Package netutils contain helper functions for network
/* connection
/**********************************************************/

package netutils

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	tcpReadTimeoutMs = uint(0)
)

// Read function with timeout
func setReadTimeout(connRead net.Conn) error {
	if tcpReadTimeoutMs == 0 {
		return nil
	}

	tcpReadDeadline := time.Duration(tcpReadTimeoutMs) * time.Millisecond
	deadline := time.Now().Add(tcpReadDeadline)
	return connRead.SetReadDeadline(deadline)
}

// Return connection IP and port
func GetConnIp(c net.Conn) (string, string) {
	s := strings.Split(c.LocalAddr().String(), ":")
	ip := s[0]
	port := s[1]
	return ip, port
}

// Start HTTP server
func StartHTTPServer(ip string, handler http.Handler) {

	//Use router to start the server
	log.Fatal(http.ListenAndServe(ip, handler))
}

func StartMTLSServer(ip, certca, certificate, key string, handler http.Handler) {
	// Create the TLS Config with the CA pool and enable Client certificate validation
	caCert, err := ioutil.ReadFile(certca)
	if err != nil {
		log.Fatal("certca reed:", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		ClientCAs:  caCertPool,
		ClientAuth: tls.RequireAndVerifyClientCert,
	}
	// Create a Server instance to listen on port 443 with the TLS config
	server := &http.Server{
		Addr:      ip,
		TLSConfig: tlsConfig,
		Handler:   handler,
	}
	// Listen to HTTPS connections with the server certificate and wait
	log.Fatal(server.ListenAndServeTLS(certificate, key))
}
