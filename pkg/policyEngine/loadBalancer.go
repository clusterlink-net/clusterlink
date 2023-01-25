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
)

var llog = logrus.WithField("component", "LoadBalancer")

const (
	random = 0
	ecmp   = 2
)

type LoadBalancerRule struct {
	Service string
	Policy  int
}

type ServiceState struct {
	totalConnections int
}
type LoadBalancer struct {
	ServiceMap      map[string]*[]string //Service to MBGs
	Policy          map[string]int       // PolicyType like RoundRobin/Random/etc
	ServiceStateMap map[string]*ServiceState
}

func (lB *LoadBalancer) Init() {
	lB.ServiceMap = make(map[string]*[]string)
	lB.Policy = make(map[string]int)
	lB.ServiceStateMap = make(map[string]*ServiceState)
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

func (lB *LoadBalancer) AddToServiceMap(service string, mbg string) {
	if mbgs, ok := lB.ServiceMap[service]; ok {
		*mbgs = append(*mbgs, mbg)
		lB.ServiceMap[service] = mbgs
	} else {
		lB.ServiceMap = make(map[string]*[]string)
		lB.ServiceMap[service] = &([]string{mbg})
	}
	llog.Infof("Remote service added %v->[%+v]", service, *(lB.ServiceMap[service]))
}

func (lB *LoadBalancer) SetPolicy(service string, policy int) {
	if lB.Policy == nil {
		lB.Init()
	}
	lB.Policy[service] = policy
}

func (lB *LoadBalancer) LookupRandom(service string) string {
	plog.Infof("service Map %+v", lB.ServiceMap)
	mbgList := lB.ServiceMap[service]
	if mbgList != nil {
		mbgs := *mbgList
		plog.Infof("mbgList for service %s -> %+v", service, mbgs)
		index := rand.Intn(len(*mbgList))
		plog.Infof("LoadBalancer (%d)target MBG %s", index, mbgs[index])
		return mbgs[index]
	} else {
		plog.Infof("mbgList is nil")
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
	plog.Infof("LoadBalancer lookup for %s", service)
	switch policy {
	case random:
		return lB.LookupRandom(service)
	case ecmp:
		return lB.LookupEcmp(service)
	default:
		return lB.LookupRandom(service)
	}
}
