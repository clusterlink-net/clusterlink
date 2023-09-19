package controlplane

import (
	"crypto/tls"
	"encoding/json"
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
	clients  []*jsonapi.Client
	lastSeen time.Time
	active   bool
	lock     sync.RWMutex
	logger   *logrus.Entry
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

// Active returns if the peer is active or not.
func (c *client) Active() bool {
	c.lock.RLock()
	a := c.active
	c.lock.RUnlock()
	return a
}

// GetHeartbeat get a heartbeat from other peers.
func (c *client) getHeartbeat() error {
	var retErr error
	// copy peer clients array aside
	peerClients := make([]*jsonapi.Client, len(c.clients))
	c.lock.RLock()
	copy(peerClients, c.clients)
	c.lock.RUnlock()

	for _, client := range peerClients {
		serverResp, err := client.Post(api.HeartbeatPath, nil)
		if err != nil {
			retErr = err
		} else {
			if serverResp.Status == http.StatusOK {
				return nil
			}
			retErr = fmt.Errorf("unable to get heartbeat (%d), server returned: %s",
				serverResp.Status, serverResp.Body)
		}
	}

	return retErr // Return an error if all client targets are unreachable
}

// heartbeatMonitor checks all peers for responsiveness, every fixed amount of time.
func (c *client) heartbeatMonitor() {
	c.logger.Info("Starting sending heartbeats to peer")
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	// get heartbeat from all the peers
	for {
		t := time.Now()
		if c.active || (!c.active && (t.Sub(c.lastSeen) > heartbeatRetransmissionTime)) {
			if err := c.getHeartbeat(); err != nil {
				if c.active {
					c.logger.Errorf("Unable to get heartbeat from peer  Error: %v", err.Error())
					c.lock.Lock()
					c.active = false
					c.lock.Unlock()
				}
			} else {
				c.lock.Lock()
				c.active = true
				c.lastSeen = t
				c.lock.Unlock()
			}
		}

		// wait till it's time for next heartbeat round
		<-ticker.C
	}
}

// newClient returns a new Peer API client.
func newClient(peer *store.Peer, tlsConfig *tls.Config) *client {
	clients := make([]*jsonapi.Client, len(peer.Gateways))
	for i, endpoint := range peer.Gateways {
		clients[i] = jsonapi.NewClient(endpoint.Host, endpoint.Port, tlsConfig)
	}
	c := &client{
		clients: clients,
		active:  true,
		logger: logrus.WithFields(logrus.Fields{
			"component": "peer-client",
			"peer":      peer}),
	}

	go c.heartbeatMonitor()
	return c
}
