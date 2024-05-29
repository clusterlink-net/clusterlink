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

package server

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"

	cpapi "github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
)

// StartDataplaneServer starts the Dataplane server.
func (d *Dataplane) StartDataplaneServer(dataplaneServerAddress string) error {
	d.logger.Infof("Dataplane server starting at %s.", dataplaneServerAddress)
	server := &http.Server{
		Addr:              dataplaneServerAddress,
		Handler:           d.router,
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       time.Duration(0), // use header timeout only
		WriteTimeout:      2 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    10 * 1024,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			ClientAuth: tls.RequireAndVerifyClientCert,
			GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
				// this function is defined for the sake of skipping certificate file reading
				// in net/http.Server.ServeTLS
				return nil, fmt.Errorf("invalid")
			},
			GetConfigForClient: func(*tls.ClientHelloInfo) (*tls.Config, error) {
				// return certificate set by the controlplane (using the SDS protocol)
				d.tlsConfigLock.RLock()
				defer d.tlsConfigLock.RUnlock()
				return d.tlsConfig, nil
			},
		},
	}

	return server.ListenAndServeTLS("", "")
}

func (d *Dataplane) addAuthzHandlers() {
	d.router.NotFound(d.dataplaneIngressAuthorize)
}

func (d *Dataplane) dataplaneIngressAuthorize(w http.ResponseWriter, r *http.Request) {
	if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 || len(r.TLS.PeerCertificates[0].DNSNames) == 0 {
		http.Error(w, "certificate does not contain a valid DNS name for the peer gateway", http.StatusBadRequest)
		return
	}

	headers := make(map[string]string)
	allowedHeaders := []string{cpapi.AuthorizationHeader}
	for _, header := range allowedHeaders {
		if value := r.Header.Get(header); value != "" {
			headers[header] = value
		}
	}

	defer func() {
		if err := r.Body.Close(); err != nil {
			d.logger.Warnf("Cannot close response body: %v.", err)
		}
	}()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		errorString := fmt.Sprintf("Unable to read response body: %v.", err)
		d.logger.Errorf(errorString)
		http.Error(w, errorString, http.StatusInternalServerError)
		return
	}

	authzReq := &authv3.CheckRequest{
		Attributes: &authv3.AttributeContext{
			Source: &authv3.AttributeContext_Peer{
				Principal: r.TLS.PeerCertificates[0].DNSNames[0],
			},
			Request: &authv3.AttributeContext_Request{
				Http: &authv3.AttributeContext_HttpRequest{
					Method:  r.Method,
					Path:    r.URL.Path,
					Headers: headers,
					Body:    string(body),
				},
			},
		},
	}

	resp, err := d.authzClient.Check(r.Context(), authzReq)
	if err != nil {
		d.logger.Errorf("Error authorizing ingress request: %v.", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if okResp, ok := resp.HttpResponse.(*authv3.CheckResponse_OkResponse); ok {
		d.routeIngress(w, r, okResp.OkResponse)
		return
	}
	if deniedResp, ok := resp.HttpResponse.(*authv3.CheckResponse_DeniedResponse); ok {
		d.logger.Infof("Ingress connection denied: %s", deniedResp.DeniedResponse.Body)
		http.Error(w, deniedResp.DeniedResponse.Body, http.StatusForbidden)
		return
	}

	d.logger.Errorf("Unknown authorization response: %+v", resp)
	http.Error(w, "Unknown authorization response.", http.StatusInternalServerError)
}

func (d *Dataplane) routeIngress(w http.ResponseWriter, r *http.Request, authzResp *authv3.OkHttpResponse) {
	if r.Method != http.MethodConnect {
		for _, header := range authzResp.ResponseHeadersToAdd {
			w.Header().Set(header.Header.Key, header.Header.Value)
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	// get target cluster (for export tunnel)
	var targetCluster string
	for _, header := range authzResp.Headers {
		if header.Header.Key == cpapi.TargetClusterHeader {
			targetCluster = header.Header.Value
			break
		}
	}

	serviceTarget, err := d.GetClusterTarget(targetCluster)
	if err != nil {
		d.logger.Errorf("Unable to get cluster target: %v.", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	d.logger.Infof("Initiating connection with %s.", serviceTarget)

	appConn, err := net.DialTimeout("tcp", serviceTarget, time.Second)
	if err != nil {
		d.logger.Errorf("Dial to export service failed: %v.", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// hijack connection
	peerConn, err := d.hijackConn(w)
	if err != nil {
		d.logger.Errorf("Hijacking failed: %v.", err)
		http.Error(w, "hijacking failed", http.StatusInternalServerError)
		appConn.Close()
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
		return nil, fmt.Errorf("hijacking failed: %w", err)
	}

	if err = peerConn.SetDeadline(time.Time{}); err != nil {
		return nil, fmt.Errorf("failed to clear deadlines on connection: %w", err)
	}

	if _, err := peerConn.Write([]byte{}); err != nil {
		_ = peerConn.Close() // close the connection ignoring errors
		return nil, fmt.Errorf("failed to write to connection: %w", err)
	}

	fmt.Fprintf(peerConn, "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\n")
	d.logger.Debugf("Connection hijacked %v->%v.", peerConn.RemoteAddr().String(), peerConn.LocalAddr().String())
	return peerConn, nil
}

func (d *Dataplane) initiateEgressConnection(targetCluster, authToken string, appConn net.Conn, tlsConfig *tls.Config) error {
	target, err := d.GetClusterTarget(targetCluster)
	if err != nil {
		d.logger.Error(err)
		return err
	}

	targetHostname, err := d.GetClusterHostname(targetCluster)
	if err != nil {
		d.logger.Errorf("Unable to get cluster hostname: %v.", err)
		return err
	}

	url := "https://" + targetHostname
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

	egressReq, err := http.NewRequest(http.MethodConnect, url, http.NoBody)
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
		d.logger.Infof("Error in TLS connection: %v.", err)
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("got HTTP %d while trying to establish dataplane connection", resp.StatusCode)
	}

	d.logger.Infof("Connection established successfully!")

	forward := newForwarder(appConn, peerConn)
	forward.run()
	return nil
}
