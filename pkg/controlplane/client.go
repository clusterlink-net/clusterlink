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
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/store"
	"github.com/clusterlink-net/clusterlink/pkg/util/jsonapi"
)

const (
	// heartbeatInterval is the time lapse between consecutive heartbeat requests to a responding peer.
	heartbeatInterval = 10 * time.Second
	// heartbeatRetransmissionTime is the time lapse between consecutive heartbeat requests to a non-responding peer.
	heartbeatRetransmissionTime = 60 * time.Second
)

// client for accessing a remote peer.
type client struct {
	// jsonapi clients for connecting to the remote peer (one per each gateway)
	clients            []*jsonapi.Client
	lastSeen           time.Time
	active             bool
	stopSignal         chan struct{}
	lock               sync.RWMutex
	logger             *logrus.Entry
	peerStatusCallback func(bool) // Callback function for notifying changes in peer
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
		return nil, fmt.Errorf("unable to serialize authorization request: %w", err)
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
		return nil, fmt.Errorf("unable to parse server response: %w", err)
	}

	resp.Allowed = true
	resp.AccessToken = authResp.AccessToken
	return resp, nil
}

// IsActive returns if the peer is active or not.
func (c *client) IsActive() bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.active
}

// setActive the peer status (active or not).
func (c *client) setActive(active bool) {
	c.lock.Lock()
	activePrevState := c.active
	c.active = active
	if active || c.lastSeen.IsZero() {
		c.lastSeen = time.Now()
	}
	c.lock.Unlock()

	// Update other components like the policy engine with the peer status.
	if active != activePrevState && c.peerStatusCallback != nil {
		c.peerStatusCallback(active)
	}
}

// GetHeartbeat get a heartbeat from other peers.
func (c *client) getHeartbeat() error {
	var retErr error
	// copy peer clients array aside
	peerClients := make([]*jsonapi.Client, len(c.clients))
	{
		c.lock.RLock()
		defer c.lock.RUnlock()
		copy(peerClients, c.clients)
	}

	for _, client := range peerClients {
		serverResp, err := client.Get(api.HeartbeatPath)
		if err != nil {
			retErr = errors.Join(retErr, err)
			continue
		}

		if serverResp.Status == http.StatusOK {
			return nil
		}

		retErr = errors.Join(retErr, fmt.Errorf("unable to get heartbeat (%d), server returned: %s",
			serverResp.Status, serverResp.Body))
	}

	return retErr // Return an error if all client targets are unreachable
}

// StopMonitor send signal to stop heartbeat monitor.
func (c *client) StopMonitor() {
	close(c.stopSignal)
}

// heartbeatMonitor checks all peers for responsiveness, every fixed amount of time.
func (c *client) heartbeatMonitor() {
	c.logger.Info("Start sending heartbeat requests to peer")
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.stopSignal:
			return
		default:
			t := time.Now()
			if c.IsActive() || (!c.IsActive() && (t.Sub(c.lastSeen) > heartbeatRetransmissionTime)) ||
				c.lastSeen.IsZero() {
				if err := c.getHeartbeat(); err != nil {
					if c.IsActive() {
						c.logger.Errorf("Unable to get heartbeat from peer error: %v", err.Error())
						c.setActive(false)
					}
				} else {
					c.setActive(true)
				}
			}
		}
		// wait till it's time for next heartbeat round
		<-ticker.C
	}
}

// SetPeerStatusCallback set the peerStatusCallback.
func (c *client) SetPeerStatusCallback(callback func(bool)) {
	c.peerStatusCallback = callback
}

// newClient returns a new Peer API client.
func newClient(peer *store.Peer, tlsConfig *tls.Config) *client {
	clients := make([]*jsonapi.Client, len(peer.Gateways))
	for i, endpoint := range peer.Gateways {
		clients[i] = jsonapi.NewClient(endpoint.Host, endpoint.Port, tlsConfig)
	}
	c := &client{
		clients:    clients,
		active:     false,
		lastSeen:   time.Time{},
		stopSignal: make(chan struct{}),
		logger: logrus.WithFields(logrus.Fields{
			"component": "peer-client",
			"peer":      peer}),
	}

	go c.heartbeatMonitor()
	return c
}
