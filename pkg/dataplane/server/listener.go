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

package server

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
)

// DeleteListener deletes the listener to an imported service
func (d *Dataplane) DeleteListener(name string) {
	d.listenerEnd[name] <- true
}

// CreateListener starts a listener to an imported service
func (d *Dataplane) CreateListener(name, ip string, port uint32) {
	listenTarget := ip + ":" + strconv.Itoa(int(port))
	d.listenerEnd[name] = make(chan bool)
	d.logger.Infof("Starting a listener for imported service %s at %s.", name, listenTarget)
	acceptor, err := net.Listen("tcp", listenTarget)
	if err != nil {
		d.logger.Infof("Error listening to por: %v.", err)
		return
	}
	go func() {
		if err = d.serveEgressConnections(name, acceptor); err != nil {
			d.logger.Errorf("Failed to serve egress connection on %s: %+v.", listenTarget, err)
		}
	}()
	<-d.listenerEnd[name]
	acceptor.Close()
}

func (d *Dataplane) serveEgressConnections(name string, listener net.Listener) error {
	for {
		d.logger.Infof("Serving for imported service %s at %s.", name, listener.Addr())
		conn, err := listener.Accept()
		if err != nil {
			d.logger.Error("Failed to accept egress connection", err)
			return err
		}

		d.logger.Debugf("Received an egress connection at listener for imported service %s from %s.", name, conn.RemoteAddr().String())
		d.logger.Debugf("Connection: %+v.", conn)

		targetPeer, accessToken, err := d.getEgressAuth(name, strings.Split(conn.RemoteAddr().String(), ":")[0])
		if err != nil {
			d.logger.Infof("Failed egress authorization: %v.", err)
			conn.Close()
			continue
		}
		d.logger.Infof("Received auth from controlplane: target peer: %s with %s", targetPeer, accessToken)

		targetHost, err := d.GetClusterHost(targetPeer)
		if err != nil {
			return err
		}
		tlsConfig := d.parsedCertData.ClientConfig(targetHost)

		go func() {
			err := d.initiateEgressConnection(targetPeer, accessToken, conn, tlsConfig)
			if err != nil {
				d.logger.Errorf("Failed to initiate egress connection:  %v.", err)
			}
		}()
	}
}

func (d *Dataplane) getEgressAuth(name, sourceIP string) (string, string, error) {
	url := "https://" + d.controlplaneTarget + api.DataplaneEgressAuthorizationPath
	egressAuthReq, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return "", "", err
	}
	egressAuthReq.Close = true

	egressAuthReq.Header.Add(api.ClientIPHeader, sourceIP)
	egressAuthReq.Header.Add(api.ImportHeader, name)
	egressAuthResp, err := d.apiClient.Do(egressAuthReq)
	if err != nil {
		d.logger.Errorf("Unable to send auth/egress request: %v.", err)
		return "", "", err
	}
	defer egressAuthResp.Body.Close()
	if egressAuthResp.StatusCode != http.StatusOK {
		d.logger.Infof("Failed to obtain egress authorization: %s", egressAuthResp.Status)
		return "", "", fmt.Errorf("failed egress authorization:%s", egressAuthResp.Status)
	}
	return egressAuthResp.Header.Get(api.TargetClusterHeader), egressAuthResp.Header.Get(api.AuthorizationHeader), nil
}
