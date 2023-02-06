/**********************************************************/
/* Package Policy contain all Policies and data structure
/* related to Policy that can run in mbg
/**********************************************************/
package policyEngine

import (
	"encoding/json"
	"math/rand"
	"net/http"

	"github.com/sirupsen/logrus"
	event "github.ibm.com/mbg-agent/pkg/eventManager"
)

var llog = logrus.WithField("component", "LoadBalancer")

type PolicyLoadBalancer string

const (
	Random PolicyLoadBalancer = "random"
	Ecmp                      = "ecmp"
	Static                    = "static"
)

type LoadBalancerRule struct {
	Service    string
	Policy     PolicyLoadBalancer
	DefaultMbg string
}

type ServiceState struct {
	totalConnections int
	defaultMbg       string
}
type LoadBalancer struct {
	ServiceMap      map[string]*[]string          //Service to MBGs
	Policy          map[string]PolicyLoadBalancer // PolicyType like ecmp(Round-robin)/Random/etc
	ServiceStateMap map[string]*ServiceState
	defaultPolicy   PolicyLoadBalancer
}

func (lB *LoadBalancer) Init() {
	lB.ServiceMap = make(map[string]*[]string)
	lB.Policy = make(map[string]PolicyLoadBalancer)
	lB.ServiceStateMap = make(map[string]*ServiceState)
	lB.defaultPolicy = Random
}

func (lB *LoadBalancer) SetPolicyReq(w http.ResponseWriter, r *http.Request) {
	var requestAttr LoadBalancerRule
	err := json.NewDecoder(r.Body).Decode(&requestAttr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	plog.Infof("Set LB Policy request : %+v", requestAttr)

	lB.SetPolicy(requestAttr.Service, requestAttr.Policy, requestAttr.DefaultMbg)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func (lB *LoadBalancer) GetPolicyReq(w http.ResponseWriter, r *http.Request) {
	plog.Infof("Get LB Policy request ")
	respJson, err := json.Marshal(lB.Policy)
	if err != nil {
		plog.Errorf("Unable to Marshal LB Policy")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, err = w.Write(respJson)
	if err != nil {
		plog.Errorf("Unable to write response %v", err)
	}
}

func (lB *LoadBalancer) AddToServiceMap(service string, mbg string) {
	if mbgs, ok := lB.ServiceMap[service]; ok {
		*mbgs = append(*mbgs, mbg)
		lB.ServiceMap[service] = mbgs
	} else {
		lB.ServiceMap[service] = &([]string{mbg})
		if _, ok := lB.Policy[service]; !ok { //set default random policy
			lB.setPolicy2Service(service, lB.defaultPolicy, "")
		}

	}
	llog.Infof("Remote service added %v->[%+v]", service, *(lB.ServiceMap[service]))
}

func (lB *LoadBalancer) SetPolicy(service string, policy PolicyLoadBalancer, defaultMbg string) {
	if lB.Policy == nil {
		lB.Init()
	}

	if service == event.Wildcard {
		for s, _ := range lB.Policy {
			lB.setPolicy2Service(s, policy, defaultMbg)
		}
	} else {
		lB.setPolicy2Service(service, policy, defaultMbg)
	}
}

func (lB *LoadBalancer) setPolicy2Service(service string, policy PolicyLoadBalancer, defaultMbg string) {
	plog.Infof("Set LB policy %v for service %+v", policy, service)
	if policy == Static && !lB.checkMbgExist(service, defaultMbg) {
		llog.Errorf("Remote service  %v is not exist in [%+v]", service, defaultMbg)
		defaultMbg = ""
	}
	lB.Policy[service] = policy
	lB.ServiceStateMap[service] = &ServiceState{totalConnections: 0, defaultMbg: defaultMbg}
}

func (lB *LoadBalancer) updateState(service string) {
	lB.ServiceStateMap[service].totalConnections = lB.ServiceStateMap[service].totalConnections + 1
}

func (lB *LoadBalancer) LookupRandom(service string) string {
	mbgList := lB.ServiceMap[service]
	if mbgList != nil {
		mbgs := *mbgList
		plog.Infof("mbgList for service %s -> %+v", service, mbgs)
		index := rand.Intn(len(*mbgList))
		plog.Infof("LoadBalancer selects index(%d) - target MBG %s", index, mbgs[index])
		return mbgs[index]
	}
	return ""
}

func (lB *LoadBalancer) LookupEcmp(service string) string {
	mbgList := lB.ServiceMap[service]
	if mbgList != nil {
		mbgs := *mbgList
		index := lB.ServiceStateMap[service].totalConnections % len(mbgs)
		return mbgs[index]
	}
	return ""
}

func (lB *LoadBalancer) LookupStatic(service string) string {
	mbgList := lB.ServiceMap[service]
	if mbgList != nil {
		mbg := lB.ServiceStateMap[service].defaultMbg
		plog.Infof("LoadBalancer selects - target MBG %s", mbg)
		return mbg
	}
	return ""
}

func (lB *LoadBalancer) Lookup(service string) string {
	policy := lB.Policy[service]

	lB.updateState(service)
	plog.Infof("LoadBalancer lookup for %s with policy %s", service, policy)
	switch policy {
	case Random:
		return lB.LookupRandom(service)
	case Ecmp:
		return lB.LookupEcmp(service)
	case Static:
		return lB.LookupStatic(service)
	default:
		return lB.LookupRandom(service)
	}
}

func (lB *LoadBalancer) checkMbgExist(service, mbg string) bool {
	mbgList := lB.ServiceMap[service]
	if mbgList != nil {
		for _, val := range *mbgList {
			if val == mbg {
				return true
			}
		}
	}
	return false
}
