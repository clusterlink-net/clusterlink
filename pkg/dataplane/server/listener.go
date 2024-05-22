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
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"

	"github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
)

// CreateListener starts a listener to an imported service.
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
	d.logger.Infof("Ending the listener for imported service %s at %s.", name, listenTarget)
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

		d.logger.Debugf(
			"Received an egress connection at listener for imported service %s from %s.", name, conn.RemoteAddr().String())
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
			d.logger.Errorf("Unable to get cluster host :%v.", err)
			conn.Close()
			continue
		}

		d.tlsConfigLock.RLock()
		tlsConfig := d.tlsConfig.Clone()
		d.tlsConfigLock.RUnlock()

		tlsConfig.ServerName = targetHost

		go func() {
			err := d.initiateEgressConnection(targetPeer, accessToken, conn, tlsConfig)
			if err != nil {
				d.logger.Errorf("Failed to initiate egress connection: %v.", err)
				conn.Close()
			}
		}()
	}
}

// getEgressAuth returns the target cluster and authorization token for the outgoing connection.
func (d *Dataplane) getEgressAuth(name, sourceIP string) (string, string, error) { //nolint:gocritic // unnamedResult
	components := strings.SplitN(name, "/", 2)
	authzReq := &authv3.CheckRequest{
		Attributes: &authv3.AttributeContext{
			Source: &authv3.AttributeContext_Peer{
				Address: &corev3.Address{
					Address: &corev3.Address_EnvoyInternalAddress{
						EnvoyInternalAddress: &corev3.EnvoyInternalAddress{},
					},
				},
			},
			Request: &authv3.AttributeContext_Request{
				Http: &authv3.AttributeContext_HttpRequest{
					Headers: map[string]string{
						api.ImportNamespaceHeader: components[0],
						api.ImportNameHeader:      components[1],
						api.ClientIPHeader:        sourceIP,
					},
				},
			},
		},
	}

	resp, err := d.authzClient.Check(context.Background(), authzReq)
	if err != nil {
		d.logger.Errorf("Error authorizing egress request: %v.", err)
		return "", "", err
	}

	okResp, ok := resp.HttpResponse.(*authv3.CheckResponse_OkResponse)
	if !ok {
		if deniedResp, denied := resp.HttpResponse.(*authv3.CheckResponse_DeniedResponse); denied {
			return "", "", fmt.Errorf("egress connection denied: %s", deniedResp.DeniedResponse.Body)
		}
		return "", "", fmt.Errorf("unknown authorization response: %+v", resp)
	}

	// get target and access token from response headers
	var accessToken, targetCluster string
	for _, header := range okResp.OkResponse.Headers {
		if header.Header.Key == api.TargetClusterHeader {
			targetCluster = header.Header.Value
		} else if header.Header.Key == api.AuthorizationHeader {
			accessToken = header.Header.Value
		}
	}

	if targetCluster == "" {
		return "", "", fmt.Errorf("missing target cluster")
	}

	if accessToken == "" {
		return "", "", fmt.Errorf("missing access token")
	}

	return targetCluster, accessToken, nil
}
