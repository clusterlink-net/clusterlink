package admin

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.ibm.com/mbg-agent/cmd/gwctl/config"
	"github.ibm.com/mbg-agent/pkg/api"

	event "github.ibm.com/mbg-agent/pkg/controlplane/eventManager"
	"github.ibm.com/mbg-agent/pkg/policyEngine"
	"github.ibm.com/mbg-agent/pkg/utils/httputils"
)

// Client struct allow to send REST API commands to GW
type Client struct {
	ID  string
	Cfg config.ClientConfig
}

// Const for Add or Del policy- TODO remove it when the new policy engine is integrated
const (
	Add int = iota
	Del
)

const (
	acl    = "acl"
	aclAdd = "aclAdd"
	aclDel = "aclDel"
	lb     = "lb"
	lbAdd  = "lbAdd"
	lbDel  = "lbDel"
	show   = "show"
)

// NewClient creates a new Client object.
func NewClient(cfg config.ClientConfig) (*Client, error) {
	gwctl := Client{ID: cfg.ID}
	if cfg.PolicyEngineIP == "" {
		cfg.PolicyEngineIP = gwctl.getProtocolPrefix(cfg.Dataplane) + cfg.GwIP + "/policy"
	}

	if cfg.MetricsManagerIP == "" {
		cfg.MetricsManagerIP = gwctl.getProtocolPrefix(cfg.Dataplane) + cfg.GwIP + "/metrics"
	}

	c, err := config.NewClientConfig(cfg)
	gwctl.Cfg = *c
	if err != nil {
		return &Client{}, err
	}

	return &gwctl, nil
}

// GetClientFromID loads Client from file according to the id.
func GetClientFromID(id string) (*Client, error) {
	gwctl := Client{ID: id}
	c, err := config.GetConfigFromID(id)
	gwctl.Cfg = *c
	if err != nil {
		return &Client{}, err
	}
	return &gwctl, nil
}

/* Peer functions*/

// CreatePeer sends a create peer request to the GW.
func (g *Client) CreatePeer(peer api.Peer) error {
	gwIP := g.Cfg.GetGwIP()
	address := g.getProtocolPrefix(g.Cfg.GetDataplane()) + gwIP + "/peers/"
	j, err := json.Marshal(peer)
	if err != nil {
		return err
	}
	_, err = httputils.HttpPost(address, j, g.GetHTTPClient())
	return err
}

// DeletePeer sends a delete peer request to the GW.
func (g *Client) DeletePeer(p api.Peer) error {
	// Delete peer in local MBG
	gwIP := g.Cfg.GetGwIP()
	address := g.getProtocolPrefix(g.Cfg.GetDataplane()) + gwIP + "/peers/" + p.Name
	j, err := json.Marshal(api.Peer{Name: p.Name})
	if err != nil {
		return err
	}
	_, err = httputils.HttpDelete(address, j, g.GetHTTPClient())
	return err
}

// GetPeers sends a get-all peers request to the GW.
func (g *Client) GetPeers() ([]api.Peer, error) {
	gwIP := g.Cfg.GetGwIP()
	address := g.getProtocolPrefix(g.Cfg.GetDataplane()) + gwIP + "/peers/"
	resp, err := httputils.HttpGet(address, g.GetHTTPClient())
	if err != nil {
		return nil, err
	}
	p := []api.Peer{}
	if err := json.Unmarshal(resp, &p); err != nil {
		return nil, err
	}
	return p, nil
}

// GetPeer sends a get-specific peer request to the GW.
func (g *Client) GetPeer(p api.Peer) (api.Peer, error) {
	var pRtn api.Peer
	gwIP := g.Cfg.GetGwIP()
	address := g.getProtocolPrefix(g.Cfg.GetDataplane()) + gwIP + "/peers/" + p.Name

	resp, err := httputils.HttpGet(address, g.GetHTTPClient())
	if err != nil {
		return p, err
	}

	if err := json.Unmarshal(resp, &pRtn); err != nil {
		return pRtn, err
	}
	return pRtn, nil
}

/* Export functions*/

// CreateExportService sends a create export service request to the GW.
func (g *Client) CreateExportService(e api.Export) error {
	gwIP := g.Cfg.GetGwIP()

	address := g.getProtocolPrefix(g.Cfg.GetDataplane()) + gwIP + "/exports/"
	j, err := json.Marshal(e)
	if err != nil {
		return err
	}
	_, err = httputils.HttpPost(address, j, g.GetHTTPClient())
	return err
}

// DeleteExportService sends a delete export service request to the GW.
func (g *Client) DeleteExportService(e api.Export) error {
	gwIP := g.Cfg.GetGwIP()
	address := g.getProtocolPrefix(g.Cfg.GetDataplane()) + gwIP + "/exports/" + e.Name
	resp, _ := httputils.HttpDelete(address, nil, g.GetHTTPClient())
	fmt.Printf("Response message for deleting service [%s]:%s \n", e.Name, string(resp))
	return nil
}

// GetExportServices sends a get-all export services request to the GW.
func (g *Client) GetExportServices() ([]api.Export, error) {
	gwIP := g.Cfg.GetGwIP()
	address := g.getProtocolPrefix(g.Cfg.GetDataplane()) + gwIP + "/exports/"
	resp, err := httputils.HttpGet(address, g.GetHTTPClient())
	if err != nil {
		return []api.Export{}, err
	}
	sArr := []api.Export{}
	if err := json.Unmarshal(resp, &sArr); err != nil {
		return []api.Export{}, err
	}
	return sArr, nil
}

// GetExportService sends a get export service request to the GW.
func (g *Client) GetExportService(e api.Export) (api.Export, error) {
	gwIP := g.Cfg.GetGwIP()
	address := g.getProtocolPrefix(g.Cfg.GetDataplane()) + gwIP + "/exports/" + e.Name
	resp, err := httputils.HttpGet(address, g.GetHTTPClient())
	if err != nil {
		return api.Export{}, err
	}
	var s api.Export
	if err := json.Unmarshal(resp, &s); err != nil {
		return api.Export{}, err
	}
	return s, nil
}

/* Import functions*/

// CreateImportService sends a create import service request to the GW.
func (g *Client) CreateImportService(svcImport api.Import) error {
	gwIP := g.Cfg.GetGwIP()

	address := g.getProtocolPrefix(g.Cfg.GetDataplane()) + gwIP + "/imports/"
	j, err := json.Marshal(svcImport)
	if err != nil {
		return err
	}
	_, err = httputils.HttpPost(address, j, g.GetHTTPClient())
	return err
}

// DeleteImportService sends a delete import service request to the GW.
func (g *Client) DeleteImportService(i api.Import) error {
	gwIP := g.Cfg.GetGwIP()
	address := g.getProtocolPrefix(g.Cfg.GetDataplane()) + gwIP + "/imports/" + i.Name
	j, err := json.Marshal(i)
	if err != nil {
		fmt.Printf("Unable to marshal json: %v", err)
		return err
	}
	resp, _ := httputils.HttpDelete(address, j, g.GetHTTPClient())
	fmt.Printf("Response message for deleting service [%s]:%s \n", i.Name, string(resp))
	return nil
}

// GetImportService sends a get import service request to the GW.
func (g *Client) GetImportService(i api.Import) (api.Import, error) {
	gwIP := g.Cfg.GetGwIP()
	var iRtn api.Import
	address := g.getProtocolPrefix(g.Cfg.GetDataplane()) + gwIP + "/imports/" + i.Name
	resp, err := httputils.HttpGet(address, g.GetHTTPClient())
	if err != nil {
		return i, err
	}
	if err := json.Unmarshal(resp, &iRtn); err != nil {
		return iRtn, err
	}
	return iRtn, nil
}

// GetImportServices sends a get-all import services request to the GW.
func (g *Client) GetImportServices() ([]api.Import, error) {
	gwIP := g.Cfg.GetGwIP()

	address := g.getProtocolPrefix(g.Cfg.GetDataplane()) + gwIP + "/imports/"
	resp, err := httputils.HttpGet(address, g.GetHTTPClient())
	if err != nil {
		return nil, err
	}
	iArr := []api.Import{}
	if err := json.Unmarshal(resp, &iArr); err != nil {
		return nil, err
	}

	return iArr, nil
}

/* Binding functions*/

// CreateBinding sends a create binding request to the GW.
func (g *Client) CreateBinding(b api.Binding) error {
	gwIP := g.Cfg.GetGwIP()

	address := g.getProtocolPrefix(g.Cfg.GetDataplane()) + gwIP + "/bindings/"
	j, err := json.Marshal(b)
	if err != nil {
		return err
	}
	_, err = httputils.HttpPost(address, j, g.GetHTTPClient())
	return err
}

// DeleteBinding sends a delete binding request to the GW.
func (g *Client) DeleteBinding(b api.Binding) error {
	gwIP := g.Cfg.GetGwIP()

	address := g.getProtocolPrefix(g.Cfg.GetDataplane()) + gwIP + "/bindings/" + b.Spec.Import
	j, err := json.Marshal(b)
	if err != nil {
		return err
	}
	_, err = httputils.HttpDelete(address, j, g.GetHTTPClient())
	return err
}

// GetBinding sends a get binding request to the GW.
func (g *Client) GetBinding(importID string) ([]api.Binding, error) {
	gwIP := g.Cfg.GetGwIP()

	address := g.getProtocolPrefix(g.Cfg.GetDataplane()) + gwIP + "/bindings/" + importID
	resp, err := httputils.HttpGet(address, g.GetHTTPClient())
	if err != nil {
		return nil, err
	}
	bArr := []api.Binding{}
	if err := json.Unmarshal(resp, &bArr); err != nil {
		return nil, err
	}
	return bArr, nil
}

/* Policy functions*/

// SendACLPolicy sends an ACL request to the GW.
func (g *Client) SendACLPolicy(serviceSrc string, serviceDst string, gwDest string, priority int, action event.Action, command int) error {
	url := g.Cfg.GetPolicyEngineIP() + "/" + acl
	switch command {
	case Add:
		url += "/add"
	case Del:
		url += "/delete"
	default:
		return fmt.Errorf("unknown command")
	}
	jsonReq, err := json.Marshal(policyEngine.AclRule{ServiceSrc: serviceSrc, ServiceDst: serviceDst, MbgDest: gwDest, Priority: priority, Action: action})
	if err != nil {
		return err
	}
	_, err = httputils.HttpPost(url, jsonReq, g.GetHTTPClient())
	return err
}

// SendLBPolicy sends an LB request to the GW.
func (g *Client) SendLBPolicy(serviceSrc, serviceDst string, policy policyEngine.PolicyLoadBalancer, gwDest string, command int) error {
	url := g.Cfg.GetPolicyEngineIP() + "/" + lb
	switch command {
	case Add:
		url += "/add"
	case Del:
		url += "/delete"
	default:
		return fmt.Errorf("unknow command")
	}
	jsonReq, err := json.Marshal(policyEngine.LoadBalancerRule{ServiceSrc: serviceSrc, ServiceDst: serviceDst, Policy: policy, DefaultMbg: gwDest})
	if err != nil {
		return err
	}
	_, err = httputils.HttpPost(url, jsonReq, g.GetHTTPClient())
	return err
}

// GetACLPolicies sends an ACL get request to the GW.
func (g *Client) GetACLPolicies() (policyEngine.ACL, error) {
	var rules policyEngine.ACL
	url := g.Cfg.GetPolicyEngineIP() + "/" + acl
	resp, err := httputils.HttpGet(url, g.GetHTTPClient())
	if err != nil {
		return make(policyEngine.ACL), err
	}
	err = json.NewDecoder(bytes.NewBuffer(resp)).Decode(&rules)
	if err != nil {
		fmt.Printf("Unable to decode response %v\n", err)
		return make(policyEngine.ACL), err
	}
	return rules, nil
}

// GetLBPolicies sends an LB get request to the GW.
func (g *Client) GetLBPolicies() (map[string]map[string]policyEngine.PolicyLoadBalancer, error) {
	var policies map[string]map[string]policyEngine.PolicyLoadBalancer
	url := g.Cfg.GetPolicyEngineIP() + "/" + lb
	resp, err := httputils.HttpGet(url, g.GetHTTPClient())
	if err != nil {
		return make(map[string]map[string]policyEngine.PolicyLoadBalancer), err
	}

	if err := json.Unmarshal(resp, &policies); err != nil {
		return make(map[string]map[string]policyEngine.PolicyLoadBalancer), err
	}
	return policies, nil
}

func (g *Client) GetMetrics() (map[string]event.ConnectionStatusAttr, error) {
	var connections map[string]event.ConnectionStatusAttr
	url := g.Cfg.GetMetricsManagerIP() + "/" + event.ConnectionStatus
	resp, err := httputils.HttpGet(url, g.GetHTTPClient())
	if err != nil {
		return make(map[string]event.ConnectionStatusAttr), err
	}

	if err := json.Unmarshal(resp, &connections); err != nil {
		return make(map[string]event.ConnectionStatusAttr), err
	}
	return connections, nil
}

/* Http functions */
//TODO use the common HTTP and TLS utils

// getProtocolPrefix -
func (g *Client) getProtocolPrefix(dataplane string) string {
	prefix := "http://"
	if dataplane == "mtls" {
		prefix = "https://"
	}
	return prefix

}

// GetHTTPClient - get HTTP client object
func (g *Client) GetHTTPClient() http.Client {
	client := http.Client{}
	if g.Cfg.GetDataplane() == "mtls" {
		cert, err := ioutil.ReadFile(g.Cfg.GetCaFile())
		if err != nil {
			log.Fatalf("could not open certificate file: %v", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(cert)

		certificate, err := tls.LoadX509KeyPair(g.Cfg.GetCert(), g.Cfg.GetKeyFile())
		if err != nil {
			log.Fatalf("could not load certificate: %v", err)
		}

		client = http.Client{
			Timeout: time.Minute * 3,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:      caCertPool,
					Certificates: []tls.Certificate{certificate},
					ServerName:   g.Cfg.GetID(),
				},
			},
		}
	}

	return client

}

// ConfigCurrentContext -set the current config context
func (g *Client) ConfigCurrentContext() (*config.ClientConfig, error) {
	return config.GetConfigFromID(g.ID)
}

// ConfigUseContext - get the current config context
func (g *Client) ConfigUseContext() error {
	return g.Cfg.SetDefaultClient(g.ID)
}
