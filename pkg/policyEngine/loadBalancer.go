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
)

type LoadBalancerRule struct {
	Service string
	Policy  PolicyLoadBalancer
}

type ServiceState struct {
	totalConnections int
}
type LoadBalancer struct {
	ServiceMap      map[string]*[]string          //Service to MBGs
	Policy          map[string]PolicyLoadBalancer // PolicyType like ecmp(Round-robin)/Random/etc
	ServiceStateMap map[string]*ServiceState
	ServiceCounter  map[string]uint //count number of calls for the service
	defaultPolicy   PolicyLoadBalancer
}

func (lB *LoadBalancer) Init() {
	lB.ServiceMap = make(map[string]*[]string)
	lB.Policy = make(map[string]PolicyLoadBalancer)
	lB.ServiceStateMap = make(map[string]*ServiceState)
	lB.ServiceCounter = make(map[string]uint)
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

	lB.SetPolicy(requestAttr.Service, requestAttr.Policy)

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
		lB.ServiceMap = make(map[string]*[]string)
		lB.ServiceMap[service] = &([]string{mbg})
		lB.Policy[service] = lB.defaultPolicy
	}
	llog.Infof("Remote service added %v->[%+v]", service, *(lB.ServiceMap[service]))
}

func (lB *LoadBalancer) SetPolicy(service string, policy PolicyLoadBalancer) {
	if lB.Policy == nil {
		lB.Init()
	}

	if service == event.Wildcard {
		for key, _ := range lB.Policy {
			plog.Infof("Set LB policy %v for service %+v", policy, key)
			lB.Policy[key] = policy
		}
	} else {
		lB.Policy[service] = policy
	}
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

func (lB *LoadBalancer) updateState(service string) {
	if _, ok := lB.ServiceStateMap[service]; !ok {
		lB.ServiceStateMap[service] = &ServiceState{totalConnections: 1}
	} else {
		lB.ServiceStateMap[service].totalConnections = lB.ServiceStateMap[service].totalConnections + 1
	}
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

func (lB *LoadBalancer) Lookup(service string) string {
	policy := lB.Policy[service]

	lB.updateState(service)
	plog.Infof("LoadBalancer lookup for %s with policy %s", service, policy)
	switch policy {
	case Random:
		return lB.LookupRandom(service)
	case Ecmp:
		return lB.LookupEcmp(service)
	default:
		return lB.LookupRandom(service)
	}
}
