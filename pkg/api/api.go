package api

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.ibm.com/mbg-agent/pkg/controlplane/store"

	event "github.ibm.com/mbg-agent/pkg/controlplane/eventManager"
	"github.ibm.com/mbg-agent/pkg/policyEngine"
	"github.ibm.com/mbg-agent/pkg/protocol"
	httpUtils "github.ibm.com/mbg-agent/pkg/utils/http"
)

type Gwctl struct {
	Id string
}

const (
	Add int = iota
	Del
)

const (
	acl     = "acl"
	acl_add = "acl_add"
	acl_del = "acl_del"
	lb      = "lb"
	lb_add  = "lb_add"
	lb_del  = "lb_del"
	show    = "show"
)

func CreateGwctl(id, mbgIP, caFile, certificateFile, keyFile, dataplane string) (Gwctl, error) {
	gwctl := Gwctl{Id: id}
	c := GwctlConfig{}
	err := c.Set(id, mbgIP, caFile, certificateFile, keyFile, dataplane)
	if err != nil {
		return Gwctl{}, err
	}
	c.SetPolicyDispatcher(id, gwctl.GetAddrStart(dataplane)+mbgIP+"/policy")
	return gwctl, nil
}

func (g *Gwctl) AddPeer(id, target, peerCport string) error {
	d, err := GetConfig(g.Id)
	if err != nil {
		return err
	}
	mbgIP := d.GetMbgIP()
	address := g.GetAddrStart(d.GetDataplane()) + mbgIP + "/peer/" + id
	j, err := json.Marshal(protocol.PeerRequest{Id: id, Ip: target, Cport: ":" + peerCport})
	if err != nil {
		return err
	}
	_, err = httpUtils.HttpPost(address, j, g.GetHttpClient())
	return err
}

func (g *Gwctl) AddPolicyEngine(target string) error {
	d, err := GetConfig(g.Id)
	if err != nil {
		return err
	}
	return d.SetPolicyDispatcher(g.Id, g.GetAddrStart(d.GetDataplane())+target+"/policy")
}

func (g *Gwctl) AddService(id, target, port, description string) error {
	d, err := GetConfig(g.Id)
	mbgIP := d.GetMbgIP()

	address := g.GetAddrStart(d.GetDataplane()) + mbgIP + "/service"
	j, err := json.Marshal(protocol.ServiceRequest{Id: id, Ip: target, Port: port, Description: description})
	if err != nil {
		return err
	}
	_, err = httpUtils.HttpPost(address, j, g.GetHttpClient())
	return err
}

func (g *Gwctl) ExposeService(svcId, peer string) error {
	d, _ := GetConfig(g.Id)

	mbgIP := d.GetMbgIP()

	address := g.GetAddrStart(d.GetDataplane()) + mbgIP + "/expose"
	j, err := json.Marshal(protocol.ExposeRequest{Id: svcId, Ip: "", MbgID: peer})
	if err != nil {
		return err
	}
	//send expose
	_, err = httpUtils.HttpPost(address, j, g.GetHttpClient())
	return err
}

func (g *Gwctl) SendHello(peer ...string) error {
	d, _ := GetConfig(g.Id)
	mbgIP := d.GetMbgIP()
	j := []byte{}
	if len(peer) != 0 {
		address := g.GetAddrStart(d.GetDataplane()) + mbgIP + "/hello/" + peer[0]
		_, err := httpUtils.HttpPost(address, j, g.GetHttpClient())
		return err
	}
	address := g.GetAddrStart(d.GetDataplane()) + mbgIP + "/hello/"

	_, err := httpUtils.HttpPost(address, j, g.GetHttpClient())
	return err
}

func (g *Gwctl) GetPeer(peer string) (string, error) {
	d, _ := GetConfig(g.Id)
	mbgIP := d.GetMbgIP()
	address := g.GetAddrStart(d.GetDataplane()) + mbgIP + "/peer/" + peer

	resp, err := httpUtils.HttpGet(address, g.GetHttpClient())
	if err != nil {
		return "", err
	}
	var p protocol.PeerRequest
	if err := json.Unmarshal(resp, &p); err != nil {
		return "", err
	}
	return p.Ip + ":" + p.Cport, nil
}

func (g *Gwctl) GetPeers() ([]string, error) {
	d, _ := GetConfig(g.Id)
	mbgIP := d.GetMbgIP()

	address := g.GetAddrStart(d.GetDataplane()) + mbgIP + "/peer/"

	resp, err := httpUtils.HttpGet(address, g.GetHttpClient())
	if err != nil {
		return []string{}, err
	}
	pArr := make(map[string]protocol.PeerRequest)
	if err := json.Unmarshal(resp, &pArr); err != nil {
		return []string{}, err
	}
	var peers []string
	for _, p := range pArr {
		peers = append(peers, p.Id)
	}
	return peers, nil
}

func (g *Gwctl) GetLocalServices() ([]store.LocalService, error) {
	d, _ := GetConfig(g.Id)
	mbgIP := d.GetMbgIP()
	address := g.GetAddrStart(d.GetDataplane()) + mbgIP + "/service/"
	resp, err := httpUtils.HttpGet(address, g.GetHttpClient())
	if err != nil {
		return []store.LocalService{}, err
	}
	sArr := make(map[string]protocol.ServiceRequest)
	if err := json.Unmarshal(resp, &sArr); err != nil {
		return []store.LocalService{}, err
	}
	var serviceArr []store.LocalService
	for _, s := range sArr {
		serviceArr = append(serviceArr, store.LocalService{Id: s.Id, Ip: s.Ip, Port: s.Port, Description: s.Description})
	}
	return serviceArr, nil
}

func (g *Gwctl) GetLocalService(id string) (store.LocalService, error) {
	d, _ := GetConfig(g.Id)
	mbgIP := d.GetMbgIP()
	address := g.GetAddrStart(d.GetDataplane()) + mbgIP + "/service/" + id
	resp, err := httpUtils.HttpGet(address, g.GetHttpClient())
	if err != nil {
		return store.LocalService{}, err
	}
	var s protocol.ServiceRequest
	if err := json.Unmarshal(resp, &s); err != nil {
		return store.LocalService{}, err
	}
	return store.LocalService{Id: s.Id, Ip: s.Ip, Port: s.Port, Description: s.Description}, nil
}

func (g *Gwctl) GetRemoteService(id string) ([]protocol.ServiceRequest, error) {
	d, _ := GetConfig(g.Id)
	mbgIP := d.GetMbgIP()

	address := g.GetAddrStart(d.GetDataplane()) + mbgIP + "/remoteservice/" + id
	resp, err := httpUtils.HttpGet(address, g.GetHttpClient())
	if err != nil {
		return []protocol.ServiceRequest{}, err
	}
	var sArr []protocol.ServiceRequest
	if err := json.Unmarshal(resp, &sArr); err != nil {
		return []protocol.ServiceRequest{}, err
	}
	for i, s := range sArr {
		ip := strings.Split(mbgIP, ":")[0] + s.Ip
		sArr[i].Ip = ip
	}
	return sArr, nil
}

func (g *Gwctl) GetRemoteServices() (map[string][]protocol.ServiceRequest, error) {
	d, _ := GetConfig(g.Id)
	mbgIP := d.GetMbgIP()

	address := g.GetAddrStart(d.GetDataplane()) + mbgIP + "/remoteservice/"
	resp, err := httpUtils.HttpGet(address, g.GetHttpClient())
	if err != nil {
		return nil, err
	}
	sArr := make(map[string][]protocol.ServiceRequest)
	if err := json.Unmarshal(resp, &sArr); err != nil {
		return nil, err
	}
	for i, sA := range sArr {
		for j, s := range sA {
			ip := strings.Split(mbgIP, ":")[0] + s.Ip
			sArr[i][j].Ip = ip
		}
	}
	return sArr, nil
}

func (g *Gwctl) RemovePeer(id string) error {
	d, err := GetConfig(g.Id)
	if err != nil {
		return err
	}
	// Remove peer in local MBG
	mbgIP := d.GetMbgIP()
	address := g.GetAddrStart(d.GetDataplane()) + mbgIP + "/peer/" + id
	j, err := json.Marshal(protocol.PeerRemoveRequest{Id: id, Propagate: true})
	if err != nil {
		return err
	}
	_, err = httpUtils.HttpDelete(address, j, g.GetHttpClient())
	return err
}

func (g *Gwctl) RemoveLocalService(serviceId string) {
	d, _ := GetConfig(g.Id)
	mbgIP := d.GetMbgIP()
	address := g.GetAddrStart(d.GetDataplane()) + mbgIP + "/service/" + serviceId
	resp, _ := httpUtils.HttpDelete(address, nil, g.GetHttpClient())
	fmt.Printf("Response message for deleting service [%s]:%s \n", serviceId, string(resp))
}

func (g *Gwctl) RemoveLocalServiceFromPeer(serviceId, peer string) {
	d, _ := GetConfig(g.Id)
	mbgIP := d.GetMbgIP()
	address := g.GetAddrStart(d.GetDataplane()) + mbgIP + "/service/" + serviceId + "/peer"
	j, err := json.Marshal(protocol.ServiceDeleteRequest{Id: serviceId, Peer: peer})
	if err != nil {
		fmt.Printf("Unable to marshal json: %v", err)
	}
	resp, _ := httpUtils.HttpDelete(address, j, g.GetHttpClient())
	fmt.Printf("Response message for deleting service [%s]:%s \n", serviceId, string(resp))
}

func (g *Gwctl) RemoveRemoteService(serviceId, serviceMbg string) {
	d, _ := GetConfig(g.Id)
	mbgIP := d.GetMbgIP()
	address := g.GetAddrStart(d.GetDataplane()) + mbgIP + "/remoteservice/" + serviceId
	j, err := json.Marshal(protocol.ServiceRequest{Id: serviceId, MbgID: serviceMbg})
	if err != nil {
		fmt.Printf("Unable to marshal json: %v", err)
	}

	resp, _ := httpUtils.HttpDelete(address, j, g.GetHttpClient())
	fmt.Printf("Response message for deleting service [%s]:%s \n", serviceId, string(resp))
}

func (g *Gwctl) SendACLPolicy(serviceSrc string, serviceDst string, mbgDest string, priority int, action event.Action, command int) error {
	d, _ := GetConfig(g.Id)
	url := d.GetPolicyDispatcher() + "/" + acl
	switch command {
	case Add:
		url += "/add"
	case Del:
		url += "/delete"
	default:
		return fmt.Errorf("unknown command")
	}
	jsonReq, err := json.Marshal(policyEngine.AclRule{ServiceSrc: serviceSrc, ServiceDst: serviceDst, MbgDest: mbgDest, Priority: priority, Action: action})
	if err != nil {
		return err
	}
	_, err = httpUtils.HttpPost(url, jsonReq, g.GetHttpClient())
	return err
}

func (g *Gwctl) SendLBPolicy(serviceSrc, serviceDst string, policy policyEngine.PolicyLoadBalancer, mbgDest string, command int) error {
	d, _ := GetConfig(g.Id)
	url := d.GetPolicyDispatcher() + "/" + lb
	switch command {
	case Add:
		url += "/add"
	case Del:
		url += "/delete"
	default:
		return fmt.Errorf("unknow command")
	}
	jsonReq, err := json.Marshal(policyEngine.LoadBalancerRule{ServiceSrc: serviceSrc, ServiceDst: serviceDst, Policy: policy, DefaultMbg: mbgDest})
	if err != nil {
		return err
	}
	_, err = httpUtils.HttpPost(url, jsonReq, g.GetHttpClient())
	return err
}

func (g *Gwctl) GetACLPolicies() (policyEngine.ACL, error) {
	d, _ := GetConfig(g.Id)
	var rules policyEngine.ACL
	url := d.GetPolicyDispatcher() + "/" + acl
	resp, err := httpUtils.HttpGet(url, g.GetHttpClient())
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

func (g *Gwctl) GetLBPolicies() (map[string]map[string]policyEngine.PolicyLoadBalancer, error) {
	d, _ := GetConfig(g.Id)
	var policies map[string]map[string]policyEngine.PolicyLoadBalancer
	url := d.GetPolicyDispatcher() + "/" + lb
	resp, err := httpUtils.HttpGet(url, g.GetHttpClient())
	if err != nil {
		return make(map[string]map[string]policyEngine.PolicyLoadBalancer), err
	}

	if err := json.Unmarshal(resp, &policies); err != nil {
		return make(map[string]map[string]policyEngine.PolicyLoadBalancer), err
	}
	return policies, nil
}

func (g *Gwctl) CreateServiceEndpoint(serviceId string, port int, name, namespace, mbgAppName string) error {
	d, _ := GetConfig(g.Id)

	mbgIP := d.GetMbgIP()
	address := g.GetAddrStart(d.GetDataplane()) + mbgIP + "/binding"
	j, err := json.Marshal(protocol.BindingRequest{Id: serviceId, Port: port, Name: name, Namespace: namespace, MbgApp: mbgAppName})
	if err != nil {
		return err
	}
	//send Binding request
	_, err = httpUtils.HttpPost(address, j, g.GetHttpClient())
	return err
}

func (g *Gwctl) DeleteServiceEndpoint(serviceId string) error {
	d, err := GetConfig(g.Id)
	if err != nil {
		return err
	}
	mbgIP := d.GetMbgIP()
	address := g.GetAddrStart(d.GetDataplane()) + mbgIP + "/binding/" + serviceId

	_, err = httpUtils.HttpDelete(address, []byte{}, g.GetHttpClient())
	return err
}

/**** Http functions ***/
func (g *Gwctl) GetAddrStart(dataplane string) string {
	if dataplane == "mtls" {
		return "https://"
	} else {
		return "http://"
	}
}

func (g *Gwctl) GetHttpClient() http.Client {
	d, _ := GetConfig(g.Id)
	if d.GetDataplane() == "mtls" {
		cert, err := ioutil.ReadFile(d.GetCaFile())
		if err != nil {
			log.Fatalf("could not open certificate file: %v", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(cert)

		certificate, err := tls.LoadX509KeyPair(d.GetCertificate(), d.GetKeyFile())
		if err != nil {
			log.Fatalf("could not load certificate: %v", err)
		}

		client := http.Client{
			Timeout: time.Minute * 3,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:      caCertPool,
					Certificates: []tls.Certificate{certificate},
					ServerName:   d.GetId(),
				},
			},
		}
		return client
	} else {
		return http.Client{}
	}
}

/***** config *****/
func (g *Gwctl) ConfigCurrentContext() (GwctlConfig, error) {
	return GetConfig(g.Id)
}

func (g *Gwctl) ConfigUseContext() error {
	return SetDefaultLink(g.Id)
}
