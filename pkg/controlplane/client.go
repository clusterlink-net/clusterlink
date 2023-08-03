package controlplane

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.ibm.com/mbg-agent/pkg/controlplane/api"
	"github.ibm.com/mbg-agent/pkg/controlplane/store"
	"github.ibm.com/mbg-agent/pkg/util/jsonapi"
)

// Client for accessing a remote peer.
type Client struct {
	clients []*jsonapi.Client

	logger *logrus.Entry
}

// Authorize a request for accessing a peer exported service, yielding an access token.
func (c *Client) Authorize(req *api.AuthorizationRequest) (string, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("unable to serialize authorization request: %v", err)
	}

	var resp *jsonapi.Response
	for _, client := range c.clients {
		resp, err = client.Post(api.RemotePeerAuthorizationPath, body)
		if err == nil {
			break
		}

		c.logger.Errorf("Error authorizing using endpoint %s: %v",
			client.ServerURL(), err)
	}

	if err != nil {
		return "", err
	}

	if resp.Status != http.StatusOK {
		return "", fmt.Errorf("unable to authorize connection (%d), server returned: %s",
			resp.Status, resp.Body)
	}

	var authResp api.AuthorizationResponse
	if err := json.Unmarshal(resp.Body, &authResp); err != nil {
		return "", fmt.Errorf("unable to parse server response: %v", err)
	}

	return authResp.AccessToken, nil
}

// NewClient returns a new Peer API client.
func NewClient(peer *store.Peer, tlsConfig *tls.Config) *Client {
	clients := make([]*jsonapi.Client, len(peer.Gateways))
	for i, endpoint := range peer.Gateways {
		clients[i] = jsonapi.NewClient(endpoint.Host, endpoint.Port, tlsConfig)
	}
	return &Client{
		clients: clients,
		logger: logrus.WithFields(logrus.Fields{
			"component": "peer-client",
			"peer":      peer}),
	}
}
