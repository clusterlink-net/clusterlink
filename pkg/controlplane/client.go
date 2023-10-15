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

package controlplane

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/store"
	"github.com/clusterlink-net/clusterlink/pkg/util/jsonapi"
)

// client for accessing a remote peer.
type client struct {
	// jsonapi clients for connecting to the remote peer (one per each gateway)
	clients []*jsonapi.Client

	logger *logrus.Entry
}

// remoteServerAuthorizationResponse represents an authorization response received from a remote controlplane server.
type remoteServerAuthorizationResponse struct {
	// ServiceExists is true if the requested service exists.
	ServiceExists bool
	// Allowed is true if the request is allowed.
	Allowed bool
	// AccessToken is a token that allows accessing the requested service.
	AccessToken string
}

// authorize a request for accessing a peer exported service, yielding an access token.
func (c *client) Authorize(req *api.AuthorizationRequest) (*remoteServerAuthorizationResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("unable to serialize authorization request: %v", err)
	}

	var serverResp *jsonapi.Response
	for _, client := range c.clients {
		serverResp, err = client.Post(api.RemotePeerAuthorizationPath, body)
		if err == nil {
			break
		}

		c.logger.Errorf("Error authorizing using endpoint %s: %v",
			client.ServerURL(), err)
	}

	if err != nil {
		return nil, err
	}

	resp := &remoteServerAuthorizationResponse{}
	if serverResp.Status == http.StatusNotFound {
		return resp, nil
	}

	resp.ServiceExists = true
	if serverResp.Status == http.StatusUnauthorized {
		return resp, nil
	}

	if serverResp.Status != http.StatusOK {
		return nil, fmt.Errorf("unable to authorize connection (%d), server returned: %s",
			serverResp.Status, serverResp.Body)
	}

	var authResp api.AuthorizationResponse
	if err := json.Unmarshal(serverResp.Body, &authResp); err != nil {
		return nil, fmt.Errorf("unable to parse server response: %v", err)
	}

	resp.Allowed = true
	resp.AccessToken = authResp.AccessToken
	return resp, nil
}

// newClient returns a new Peer API client.
func newClient(peer *store.Peer, tlsConfig *tls.Config) *client {
	clients := make([]*jsonapi.Client, len(peer.Gateways))
	for i, endpoint := range peer.Gateways {
		clients[i] = jsonapi.NewClient(endpoint.Host, endpoint.Port, tlsConfig)
	}
	return &client{
		clients: clients,
		logger: logrus.WithFields(logrus.Fields{
			"component": "peer-client",
			"peer":      peer}),
	}
}
