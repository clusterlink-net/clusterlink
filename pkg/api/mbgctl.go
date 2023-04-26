package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	mbg "github.ibm.com/mbg-agent/cmd/mbg/state"
	"github.ibm.com/mbg-agent/cmd/mbgctl/state"

	event "github.ibm.com/mbg-agent/pkg/eventManager"
	"github.ibm.com/mbg-agent/pkg/policyEngine"
	"github.ibm.com/mbg-agent/pkg/protocol"
	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
)

type Mbgctl struct {
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

func CreateMbgctl(id, mbgIP, caFile, certificateFile, keyFile, dataplane string) (Mbgctl, error) {
	err := state.SetState(id, mbgIP, caFile, certificateFile, keyFile, dataplane)
	if err != nil {
		return Mbgctl{}, err
	}
	return Mbgctl{id}, nil
}

func (m *Mbgctl) AddPeer(id, target, peerCport string) error {
	err := state.UpdateState(m.Id)
	if err != nil {
		return err
	}
	mbgIP := state.GetMbgIP()
	address := state.GetAddrStart() + mbgIP + "/peer/" + id
	j, err := json.Marshal(protocol.PeerRequest{Id: id, Ip: target, Cport: ":" + peerCport})
	if err != nil {
		return err
	}
	_, err = httpAux.HttpPost(address, j, state.GetHttpClient())
	return err
}

func (m *Mbgctl) AddPolicyEngine(target string) error {
	err := state.UpdateState(m.Id)
	if err != nil {
		return err
	}
	return state.AssignPolicyDispatcher(m.Id, state.GetAddrStart()+target+"/policy")
}

func (m *Mbgctl) AddService(id, target, port, description string) error {
	state.UpdateState(m.Id)
	state.AddService(m.Id, id, target, port, description)
	mbgIP := state.GetMbgIP()

	address := state.GetAddrStart() + mbgIP + "/service"
	j, err := json.Marshal(protocol.ServiceRequest{Id: id, Ip: target, Port: port, Description: description})
	if err != nil {
		return err
	}
	_, err = httpAux.HttpPost(address, j, state.GetHttpClient())
	return err
}

func (m *Mbgctl) ExposeService(id, peer string) error {
	state.UpdateState(m.Id)

	mbgIP := state.GetMbgIP()
	s := state.GetService(id)
	svcExp := s.Service

	address := state.GetAddrStart() + mbgIP + "/expose"
	j, err := json.Marshal(protocol.ExposeRequest{Id: svcExp.Id, Ip: svcExp.Ip, MbgID: peer})
	if err != nil {
		return err
	}
	//send expose
	_, err = httpAux.HttpPost(address, j, state.GetHttpClient())
	return err
}

func (m *Mbgctl) SendHello(peer ...string) error {
	state.UpdateState(m.Id)
	mbgIP := state.GetMbgIP()
	j := []byte{}
	if len(peer) != 0 {
		address := state.GetAddrStart() + mbgIP + "/hello/" + peer[0]
		_, err := httpAux.HttpPost(address, j, state.GetHttpClient())
		return err
	}
	address := state.GetAddrStart() + mbgIP + "/hello/"

	_, err := httpAux.HttpPost(address, j, state.GetHttpClient())
	return err
}

func (m *Mbgctl) GetPeer(peer string) (string, error) {
	state.UpdateState(m.Id)
	mbgIP := state.GetMbgIP()
	address := state.GetAddrStart() + mbgIP + "/peer/" + peer

	resp, err := httpAux.HttpGet(address, state.GetHttpClient())
	if err != nil {
		return "", err
	}
	var p protocol.PeerRequest
	if err := json.Unmarshal(resp, &p); err != nil {
		return "", err
	}
	return p.Ip + ":" + p.Cport, nil
}

func (m *Mbgctl) GetPeers() ([]string, error) {
	state.UpdateState(m.Id)
	mbgIP := state.GetMbgIP()

	address := state.GetAddrStart() + mbgIP + "/peer/"

	resp, err := httpAux.HttpGet(address, state.GetHttpClient())
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

func (m *Mbgctl) GetLocalServices() ([]mbg.LocalService, error) {
	state.UpdateState(m.Id)
	mbgIP := state.GetMbgIP()
	address := state.GetAddrStart() + mbgIP + "/service/"
	resp, err := httpAux.HttpGet(address, state.GetHttpClient())
	if err != nil {
		return []mbg.LocalService{}, err
	}
	sArr := make(map[string]protocol.ServiceRequest)
	if err := json.Unmarshal(resp, &sArr); err != nil {
		return []mbg.LocalService{}, err
	}
	var serviceArr []mbg.LocalService
	for _, s := range sArr {
		serviceArr = append(serviceArr, mbg.LocalService{Id: s.Id, Ip: s.Ip, Port: s.Port, Description: s.Description})
	}
	return serviceArr, nil
}

func (m *Mbgctl) GetLocalService(id string) (mbg.LocalService, error) {
	state.UpdateState(m.Id)
	mbgIP := state.GetMbgIP()
	address := state.GetAddrStart() + mbgIP + "/service/" + id
	resp, err := httpAux.HttpGet(address, state.GetHttpClient())
	if err != nil {
		return mbg.LocalService{}, err
	}
	var s protocol.ServiceRequest
	if err := json.Unmarshal(resp, &s); err != nil {
		return mbg.LocalService{}, err
	}
	return mbg.LocalService{Id: s.Id, Ip: s.Ip, Port: s.Port, Description: s.Description}, nil
}

func (m *Mbgctl) GetRemoteService(id string) ([]protocol.ServiceRequest, error) {
	state.UpdateState(m.Id)
	mbgIP := state.GetMbgIP()

	address := state.GetAddrStart() + mbgIP + "/remoteservice/" + id
	resp, err := httpAux.HttpGet(address, state.GetHttpClient())
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

func (m *Mbgctl) GetRemoteServices() (map[string][]protocol.ServiceRequest, error) {
	state.UpdateState(m.Id)
	mbgIP := state.GetMbgIP()

	address := state.GetAddrStart() + mbgIP + "/remoteservice/"
	resp, err := httpAux.HttpGet(address, state.GetHttpClient())
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

func (m *Mbgctl) RemovePeer(id string) error {
	err := state.UpdateState(m.Id)
	if err != nil {
		return err
	}
	// Remove peer in local MBG
	mbgIP := state.GetMbgIP()
	address := state.GetAddrStart() + mbgIP + "/peer/" + id
	j, err := json.Marshal(protocol.PeerRemoveRequest{Id: id, Propagate: true})
	if err != nil {
		return err
	}
	_, err = httpAux.HttpDelete(address, j, state.GetHttpClient())
	return err
}

func (m *Mbgctl) RemoveLocalService(serviceId string) {
	state.UpdateState(m.Id)
	state.DelService(m.Id, serviceId)
	mbgIP := state.GetMbgIP()
	address := state.GetAddrStart() + mbgIP + "/service/" + serviceId
	resp, _ := httpAux.HttpDelete(address, nil, state.GetHttpClient())
	fmt.Printf("Response message for deleting service [%s]:%s \n", serviceId, string(resp))
}
func (m *Mbgctl) RemoveLocalServiceFromPeer(serviceId, peer string) {
	state.UpdateState(m.Id)
	mbgIP := state.GetMbgIP()
	address := state.GetAddrStart() + mbgIP + "/service/" + serviceId + "/peer"
	j, err := json.Marshal(protocol.ServiceDeleteRequest{Id: serviceId, Peer: peer})
	if err != nil {
		fmt.Printf("Unable to marshal json: %v", err)
	}
	resp, _ := httpAux.HttpDelete(address, j, state.GetHttpClient())
	fmt.Printf("Response message for deleting service [%s]:%s \n", serviceId, string(resp))
}

func (m *Mbgctl) RemoveRemoteService(serviceId, serviceMbg string) {
	state.UpdateState(m.Id)
	state.DelService(m.Id, serviceId)
	mbgIP := state.GetMbgIP()
	address := state.GetAddrStart() + mbgIP + "/remoteservice/" + serviceId
	j, err := json.Marshal(protocol.ServiceRequest{Id: serviceId, MbgID: serviceMbg})
	if err != nil {
		fmt.Printf("Unable to marshal json: %v", err)
	}

	resp, _ := httpAux.HttpDelete(address, j, state.GetHttpClient())
	fmt.Printf("Response message for deleting service [%s]:%s \n", serviceId, string(resp))
}

func (m *Mbgctl) SendACLPolicy(serviceSrc string, serviceDst string, mbgDest string, priority int, action event.Action, command int) error {
	state.UpdateState(m.Id)
	url := state.GetPolicyDispatcher() + "/" + acl
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
	_, err = httpAux.HttpPost(url, jsonReq, state.GetHttpClient())
	return err
}

func (m *Mbgctl) SendLBPolicy(serviceSrc, serviceDst string, policy policyEngine.PolicyLoadBalancer, mbgDest string, command int) error {
	state.UpdateState(m.Id)
	url := state.GetPolicyDispatcher() + "/" + lb
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
	_, err = httpAux.HttpPost(url, jsonReq, state.GetHttpClient())
	return err
}

func (m *Mbgctl) GetACLPolicies() (policyEngine.ACL, error) {
	state.UpdateState(m.Id)
	var rules policyEngine.ACL
	url := state.GetPolicyDispatcher() + "/" + acl
	resp, err := httpAux.HttpGet(url, state.GetHttpClient())
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

func (m *Mbgctl) GetLBPolicies() (map[string]map[string]policyEngine.PolicyLoadBalancer, error) {
	state.UpdateState(m.Id)
	var policies map[string]map[string]policyEngine.PolicyLoadBalancer
	url := state.GetPolicyDispatcher() + "/" + lb
	resp, err := httpAux.HttpGet(url, state.GetHttpClient())
	if err != nil {
		return make(map[string]map[string]policyEngine.PolicyLoadBalancer), err
	}

	if err := json.Unmarshal(resp, &policies); err != nil {
		return make(map[string]map[string]policyEngine.PolicyLoadBalancer), err
	}
	return policies, nil
}

/***** config *****/
func (m *Mbgctl) ConfigCurrentContext() (state.MbgctlState, error) {
	return state.GetState("")
}

func (m *Mbgctl) ConfigUseContext() error {
	return state.SetDefaultLink(m.Id)
}
