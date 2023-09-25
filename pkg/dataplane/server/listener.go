package server

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/clusterlink-org/clusterlink/pkg/controlplane/api"
	dpapi "github.com/clusterlink-org/clusterlink/pkg/dataplane/api"
)

// DeleteListener deletes the listener to an imported service
func (d *Dataplane) DeleteListener(name string) {
	listenerChan[name] <- true
}

// CreateListenerToImportServiceEndpoint starts a listener to an imported service
func (d *Dataplane) CreateListenerToImportServiceEndpoint(name, ip string, port uint32) {
	listenTarget := ip + ":" + strconv.Itoa(int(port))
	listenerChan[name] = make(chan bool)
	d.logger.Infof("Starting an listener for imported service %s at  %s ", name, listenTarget)
	acceptor, err := net.Listen("tcp", listenTarget)
	if err != nil {
		d.logger.Infof("Error Listen to port %v", err)
		return
	}
	go func() {
		if err = d.serveEgressConnections(name, acceptor); err != nil {
			d.logger.Infof("failed to serve egress connection on  %s: %+v", listenTarget, err)
		}
	}()
	<-listenerChan[name]
	acceptor.Close()
}

func (d *Dataplane) serveEgressConnections(name string, listener net.Listener) error {
	for {
		d.logger.Infof("Serving for imported service %s at  %s ", name, listener.Addr())
		conn, err := listener.Accept()
		if err != nil {
			d.logger.Error("Failed to accept egress connection", err)
			return err
		}

		d.logger.Infof("Received an egress connection at listener for imported service %s from %s ", name, conn.RemoteAddr().String())
		d.logger.Infof("Connection : %+v", conn)
		targetPeer, accessToken, err := d.getEgressAuth(name, strings.Split(conn.RemoteAddr().String(), ":")[0])
		if err != nil {
			d.logger.Error("Failed egress authorization", err)
			conn.Close()
			continue
		}
		d.logger.Infof("Received auth from controlplane: target peer: %s with %s", targetPeer, accessToken)
		tlsConfig := d.parsedCertData.ClientConfig(dpapi.DataplaneServerName(strings.TrimPrefix(targetPeer, api.RemotePeerClusterPrefix)))

		go func() {
			err := d.initiateEgressConnection(targetPeer, accessToken, conn, tlsConfig)
			if err != nil {
				d.logger.Error("Failed to initiate egress connection ", err)
			}
		}()
	}
}

func (d *Dataplane) getEgressAuth(name, sourceIP string) (string, string, error) {
	url := "https://" + d.controlplaneTarget + api.DataplaneEgressAuthorizationPath
	egressAuthReq, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return "", "", err
	}
	egressAuthReq.Close = true

	egressAuthReq.Header.Add(api.ClientIPHeader, sourceIP)
	egressAuthReq.Header.Add(api.ImportHeader, name)
	egressAuthResp, err := d.apiClient.Do(egressAuthReq)
	if err != nil {
		d.logger.Error("Unable to send auth/egress request ", err)
		return "", "", err
	}
	if egressAuthResp.StatusCode != http.StatusOK {
		d.logger.Infof("Failed to obtained egress authorization: %s", egressAuthResp.Status)
		return "", "", fmt.Errorf("failed egress authorization:%s", egressAuthResp.Status)
	}
	return egressAuthResp.Header.Get(api.TargetClusterHeader), egressAuthResp.Header.Get(api.AuthorizationHeader), nil
}
