package server

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	cpapi "github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
	"github.com/clusterlink-net/clusterlink/pkg/dataplane/api"
	"github.com/clusterlink-net/clusterlink/pkg/util/sniproxy"
	"github.com/clusterlink-net/clusterlink/pkg/utils/netutils"
)

const (
	httpSchemaPrefix = "https://"
)

// StartDataplaneServer starts the Dataplane server
func (d *Dataplane) StartDataplaneServer(dataplaneServerAddress string) error {
	d.logger.Infof("Dataplane server starting  at %s", dataplaneServerAddress)
	server := netutils.CreateResilientHTTPServer(dataplaneServerAddress, d.router, d.parsedCertData.ServerConfig(), nil, nil, nil)

	return server.ListenAndServeTLS("", "")
}

// StartSNIServer starts the SNI Proxy in the dataplane
func (d *Dataplane) StartSNIServer(dataplaneServerAddress string) error {
	dataplaneListenAddress := ":" + strconv.Itoa(api.ListenPort)
	sniProxy := sniproxy.NewServer(map[string]string{
		d.peerName:                          d.controlplaneTarget,
		api.DataplaneServerName(d.peerName): dataplaneServerAddress,
	})

	d.logger.Infof("SNI proxy starting at %s.", dataplaneListenAddress)
	err := sniProxy.Listen(dataplaneListenAddress)
	if err != nil {
		return fmt.Errorf("unable to create listener for server on %s: %v",
			dataplaneListenAddress, err)
	}
	return sniProxy.Serve()
}

func (d *Dataplane) addAuthzHandlers() {
	d.router.Post(cpapi.DataplaneIngressAuthorizationPath, d.dataplaneIngressAuthorize)
}

func (d *Dataplane) dataplaneIngressAuthorize(w http.ResponseWriter, r *http.Request) {
	forwardingURL := httpSchemaPrefix + d.controlplaneTarget + r.URL.Path

	forwardingReq, err := http.NewRequest(r.Method, forwardingURL, r.Body)
	if err != nil {
		d.logger.Error("Forwarding error in NewRequest", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	forwardingReq.ContentLength = r.ContentLength
	for key, values := range r.Header {
		for _, value := range values {
			forwardingReq.Header.Add(key, value)
		}
	}

	resp, err := d.apiClient.Do(forwardingReq)
	if err != nil {
		d.logger.Error("Forwarding error in sending operation", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		d.logger.Infof("Failed to obtained ingress authorization: %s", resp.Status)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	d.logger.Infof("Got authorization to use service :%s", resp.Header.Get(cpapi.TargetClusterHeader))

	serviceTarget, err := GetClusterTarget(resp.Header.Get(cpapi.TargetClusterHeader))
	if err != nil {
		d.logger.Errorf("Unable to get cluster target: %v.", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// hijack connection
	peerConn, err := d.hijackConn(w)
	if err != nil {
		d.logger.Error("hijacking failed ", err)
		http.Error(w, "hijacking failed", http.StatusInternalServerError)
		return
	}

	d.logger.Infof("Initiating connection with %s.", serviceTarget)

	appConn, err := net.Dial("tcp", serviceTarget)
	if err != nil {
		d.logger.Errorf("Dial to export service failed: %v.", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	forward := newForwarder(appConn, peerConn)
	forward.run()
}

func (d *Dataplane) hijackConn(w http.ResponseWriter) (net.Conn, error) {
	d.logger.Debugf("Starting to hijack connection.")
	hj, ok := w.(http.Hijacker)
	if !ok {
		return nil, fmt.Errorf("server doesn't support hijacking")
	}
	// Hijack the connection
	peerConn, _, err := hj.Hijack()
	if err != nil {
		return nil, fmt.Errorf("hijacking failed: %v", err)
	}

	if err = peerConn.SetDeadline(time.Time{}); err != nil {
		return nil, fmt.Errorf("failed to clear deadlines on connection: %v", err)
	}

	if _, err := peerConn.Write([]byte{}); err != nil {
		_ = peerConn.Close() // close the connection ignoring errors
		return nil, fmt.Errorf("failed to write to connection: %v", err)
	}

	fmt.Fprintf(peerConn, "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\n")
	d.logger.Debugf("Connection Hijacked  %v->%v", peerConn.RemoteAddr().String(), peerConn.LocalAddr().String())
	return peerConn, nil
}

func (d *Dataplane) initiateEgressConnection(targetCluster, authToken string, appConn net.Conn, tlsConfig *tls.Config) error {
	target, err := GetClusterTarget(targetCluster)
	if err != nil {
		d.logger.Error(err)
		return err
	}
	url := httpSchemaPrefix + target + cpapi.DataplaneIngressAuthorizationPath
	d.logger.Debugf("Starting to initiate egress connection to: %s.", url)

	peerConn, err := tls.Dial("tcp", target, tlsConfig)
	if err != nil {
		d.logger.Infof("Error in connecting.. %+v", err)
		return err
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
			DialTLS:         connDialer{peerConn}.Dial,
		},
	}

	egressReq, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	egressReq.Header.Add(cpapi.AuthorizationHeader, authToken)
	d.logger.Debugf("Setting %s header to %s.", cpapi.AuthorizationHeader, authToken)

	resp, err := client.Do(egressReq)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		d.logger.Infof("Error in TLS Connection %v", err)
		return err
	}
	d.logger.Infof("Connection established successfully!")

	forward := newForwarder(appConn, peerConn)
	forward.run()
	return nil
}
