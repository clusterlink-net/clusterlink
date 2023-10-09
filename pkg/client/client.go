package client

import (
	"crypto/tls"
	"encoding/json"
	"fmt"

	"github.com/clusterlink-net/clusterlink/pkg/api"
	event "github.com/clusterlink-net/clusterlink/pkg/controlplane/eventmanager"
	"github.com/clusterlink-net/clusterlink/pkg/policyengine"
	"github.com/clusterlink-net/clusterlink/pkg/policyengine/policytypes"
	"github.com/clusterlink-net/clusterlink/pkg/util/jsonapi"
	"github.com/clusterlink-net/clusterlink/pkg/util/rest"
)

// Client for accessing the API.
type Client struct {
	client *jsonapi.Client

	// Peers client.
	Peers *rest.Client
	// Exports client.
	Exports *rest.Client
	// Imports client.
	Imports *rest.Client
	// Bindings client.
	Bindings *rest.Client
}

// New returns a new client.
func New(host string, port uint16, tlsConfig *tls.Config) *Client {
	client := jsonapi.NewClient(host, port, tlsConfig)
	return &Client{
		client: client,
		Peers: rest.NewClient(&rest.Config{
			Client:       client,
			BasePath:     "/peers",
			SampleObject: api.Peer{},
			SampleList:   []api.Peer{},
		}),
		Exports: rest.NewClient(&rest.Config{
			Client:       client,
			BasePath:     "/exports",
			SampleObject: api.Export{},
			SampleList:   []api.Export{},
		}),
		Imports: rest.NewClient(&rest.Config{
			Client:       client,
			BasePath:     "/imports",
			SampleObject: api.Import{},
			SampleList:   []api.Import{},
		}),
		Bindings: rest.NewClient(&rest.Config{
			Client:       client,
			BasePath:     "/bindings",
			SampleObject: []api.Binding{},
			SampleList:   []api.Binding{},
		}),
	}
}

// Const for Add or Del policy- TODO remove it when the new policy engine is integrated
const (
	Add int = iota
	Del
)

// SendAccessPolicy sends the policy engine a request to add, update (using add) or delete an access policy
func (c *Client) SendAccessPolicy(policy api.Policy, command int) error {
	path := policyengine.PolicyRoute + policyengine.AccessRoute
	switch command {
	case Add:
		path += policyengine.AddRoute
	case Del:
		path += policyengine.DelRoute
	default:
		return fmt.Errorf("unknown command")
	}

	_, err := c.client.Post(path, policy.Spec.Blob)
	return err
}

// SendLBPolicy sends an LB request to the GW.
func (c *Client) SendLBPolicy(serviceSrc, serviceDst string, scheme policyengine.LBScheme, gwDest string, command int) error {
	path := policyengine.PolicyRoute + policyengine.LbRoute
	switch command {
	case Add:
		path += policyengine.AddRoute
	case Del:
		path += policyengine.DelRoute
	default:
		return fmt.Errorf("unknown command")
	}
	jsonReq, err := json.Marshal(policyengine.LBPolicy{ServiceSrc: serviceSrc, ServiceDst: serviceDst, Scheme: scheme, DefaultMbg: gwDest})
	if err != nil {
		return err
	}
	_, err = c.client.Post(path, jsonReq)
	return err
}

// GetLBPolicies sends an LB get request to the GW.
func (c *Client) GetLBPolicies() (map[string]map[string]policyengine.LBScheme, error) {
	var policies map[string]map[string]policyengine.LBScheme
	path := policyengine.PolicyRoute + policyengine.LbRoute
	resp, err := c.client.Get(path)
	if err != nil {
		return make(map[string]map[string]policyengine.LBScheme), err
	}

	if err := json.Unmarshal(resp.Body, &policies); err != nil {
		return make(map[string]map[string]policyengine.LBScheme), err
	}
	return policies, nil
}

// GetAccessPolicies returns a slice of ConnectivityPolicies, that are currently used by the connectivity PDP
func (c *Client) GetAccessPolicies() ([]policytypes.ConnectivityPolicy, error) {
	var policies []policytypes.ConnectivityPolicy
	path := policyengine.PolicyRoute + policyengine.AccessRoute
	resp, err := c.client.Get(path)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(resp.Body, &policies); err != nil {
		return nil, err
	}
	return policies, nil
}

func (c *Client) GetMetrics() (map[string]event.ConnectionStatusAttr, error) {
	var connections map[string]event.ConnectionStatusAttr
	path := "/metrics/" + event.ConnectionStatus
	resp, err := c.client.Get(path)
	if err != nil {
		return make(map[string]event.ConnectionStatusAttr), err
	}

	if err := json.Unmarshal(resp.Body, &connections); err != nil {
		return make(map[string]event.ConnectionStatusAttr), err
	}
	return connections, nil
}
