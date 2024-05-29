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

package peer

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
	"github.com/clusterlink-net/clusterlink/pkg/util/jsonapi"
)

// Client for accessing a remote peer.
type Client struct {
	pr *v1alpha1.Peer
	// jsonapi clients for connecting to the remote peer (one per each gateway)
	clients []*jsonapi.Client
	logger  *logrus.Entry
}

// RemoteServerAuthorizationResponse represents an authorization response received from a remote controlplane server.
type RemoteServerAuthorizationResponse struct {
	// ServiceExists is true if the requested service exists.
	ServiceExists bool
	// Allowed is true if the request is allowed.
	Allowed bool
	// AccessToken is a token that allows accessing the requested service.
	AccessToken string
}

// getResponse tries all gateways in parallel for a response.
// The first successful response is returned.
// If all responses failed, a joined error of all responses is returned.
func (c *Client) getResponse(
	getRespFunc func(client *jsonapi.Client) (*jsonapi.Response, error),
) (*jsonapi.Response, error) {
	if len(c.clients) == 1 {
		return getRespFunc(c.clients[0])
	}

	results := make(chan struct {
		*jsonapi.Response
		error
	})
	var done bool
	var lock sync.Mutex
	for _, client := range c.clients {
		go func(currClient *jsonapi.Client) {
			resp, err := getRespFunc(currClient)
			lock.Lock()
			defer lock.Unlock()
			if done {
				return
			}
			results <- struct {
				*jsonapi.Response
				error
			}{resp, err}
		}(client)
	}

	var retErr error
	for range c.clients {
		result := <-results
		if result.error == nil {
			lock.Lock()
			done = true
			lock.Unlock()
			return result.Response, nil
		}

		retErr = errors.Join(retErr, result.error)
	}

	return nil, retErr
}

// Authorize a request for accessing a peer exported service, yielding an access token.
func (c *Client) Authorize(req *api.AuthorizationRequest) (string, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("unable to serialize authorization request: %w", err)
	}

	serverResp, err := c.getResponse(func(client *jsonapi.Client) (*jsonapi.Response, error) {
		return client.Post(api.RemotePeerAuthorizationPath, body)
	})
	if err != nil {
		return "", err
	}

	if serverResp.Status != http.StatusOK {
		return "", fmt.Errorf("unable to authorize connection (%d), server returned: %s",
			serverResp.Status, serverResp.Body)
	}

	return serverResp.Headers.Get(api.AccessTokenHeader), nil
}

// GetHeartbeat get a heartbeat from other peers.
func (c *Client) GetHeartbeat() error {
	serverResp, err := c.getResponse(func(client *jsonapi.Client) (*jsonapi.Response, error) {
		return client.Get(api.HeartbeatPath)
	})
	if err != nil {
		return err
	}

	if serverResp.Status != http.StatusOK {
		return fmt.Errorf("unable to get heartbeat (%d), server returned: %s",
			serverResp.Status, serverResp.Body)
	}

	return nil
}

// Peer object this client corresponds to.
func (c *Client) Peer() *v1alpha1.Peer {
	return c.pr
}

// NewClient returns a new Peer API client.
func NewClient(peer *v1alpha1.Peer, tlsConfig *tls.Config) *Client {
	clients := make([]*jsonapi.Client, len(peer.Spec.Gateways))
	for i, endpoint := range peer.Spec.Gateways {
		clients[i] = jsonapi.NewClient(endpoint.Host, endpoint.Port, tlsConfig)
	}

	return &Client{
		pr:      peer,
		clients: clients,
		logger: logrus.WithFields(logrus.Fields{
			"component": "controlplane.peer.client",
			"peer":      peer,
		}),
	}
}
