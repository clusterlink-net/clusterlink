package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"

	"github.ibm.com/mbg-agent/pkg/api"
	event "github.ibm.com/mbg-agent/pkg/controlplane/eventManager"
	"github.ibm.com/mbg-agent/pkg/policyEngine"
	"github.ibm.com/mbg-agent/pkg/util/jsonapi"
	"github.ibm.com/mbg-agent/pkg/util/rest"
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

/* Policy functions*/

const (
	acl    = "acl"
	aclAdd = "aclAdd"
	aclDel = "aclDel"
	lb     = "lb"
	lbAdd  = "lbAdd"
	lbDel  = "lbDel"
	show   = "show"
)

// Const for Add or Del policy- TODO remove it when the new policy engine is integrated
const (
	Add int = iota
	Del
)

// SendACLPolicy sends an ACL request to the GW.
func (c *Client) SendACLPolicy(serviceSrc string, serviceDst string, gwDest string, priority int, action event.Action, command int) error {
	path := "/policy/" + acl
	switch command {
	case Add:
		path += "/add"
	case Del:
		path += "/delete"
	default:
		return fmt.Errorf("unknown command")
	}
	jsonReq, err := json.Marshal(policyEngine.AclRule{ServiceSrc: serviceSrc, ServiceDst: serviceDst, MbgDest: gwDest, Priority: priority, Action: action})
	if err != nil {
		return err
	}

	_, err = c.client.Post(path, jsonReq)
	return err
}

// SendLBPolicy sends an LB request to the GW.
func (c *Client) SendLBPolicy(serviceSrc, serviceDst string, policy policyEngine.PolicyLoadBalancer, gwDest string, command int) error {
	path := "/policy/" + lb
	switch command {
	case Add:
		path += "/add"
	case Del:
		path += "/delete"
	default:
		return fmt.Errorf("unknow command")
	}
	jsonReq, err := json.Marshal(policyEngine.LoadBalancerRule{ServiceSrc: serviceSrc, ServiceDst: serviceDst, Policy: policy, DefaultMbg: gwDest})
	if err != nil {
		return err
	}
	_, err = c.client.Post(path, jsonReq)
	return err
}

// GetACLPolicies sends an ACL get request to the GW.
func (c *Client) GetACLPolicies() (policyEngine.ACL, error) {
	var rules policyEngine.ACL
	path := "/policy/" + acl
	resp, err := c.client.Get(path)
	if err != nil {
		return make(policyEngine.ACL), err
	}
	err = json.NewDecoder(bytes.NewBuffer(resp.Body)).Decode(&rules)
	if err != nil {
		fmt.Printf("Unable to decode response %v\n", err)
		return make(policyEngine.ACL), err
	}
	return rules, nil
}

// GetLBPolicies sends an LB get request to the GW.
func (c *Client) GetLBPolicies() (map[string]map[string]policyEngine.PolicyLoadBalancer, error) {
	var policies map[string]map[string]policyEngine.PolicyLoadBalancer
	path := "/policy/" + lb
	resp, err := c.client.Get(path)
	if err != nil {
		return make(map[string]map[string]policyEngine.PolicyLoadBalancer), err
	}

	if err := json.Unmarshal(resp.Body, &policies); err != nil {
		return make(map[string]map[string]policyEngine.PolicyLoadBalancer), err
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
