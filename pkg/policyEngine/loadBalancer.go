/**********************************************************/
/* Package Policy contain all Policies and data structure
/* related to Policy that can run in mbg
/**********************************************************/
package policyEngine

import (
	"encoding/json"
	"fmt"
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
	ServiceSrc string
	ServiceDst string
	Policy     PolicyLoadBalancer
	DefaultMbg string
}

type ServiceState struct {
	totalConnections int
	defaultMbg       string
}
type LoadBalancer struct {
	ServiceMap      map[string]*[]string                       // Service to MBGs
	Policy          map[string](map[string]PolicyLoadBalancer) // PolicyMap [serviceDst][serviceSrc]Policy
	ServiceStateMap map[string]map[string]*ServiceState        // Per source and destination service
}

func (lB *LoadBalancer) Init() {
	lB.ServiceMap = make(map[string]*[]string)
	lB.Policy = make(map[string](map[string]PolicyLoadBalancer))
	lB.ServiceStateMap = make(map[string](map[string]*ServiceState))
	lB.Policy[event.Wildcard] = make(map[string]PolicyLoadBalancer)
	lB.Policy[event.Wildcard][event.Wildcard] = Random //default policy
}

/*********************  HTTP functions ***************************************************/
func (lB *LoadBalancer) SetPolicyReq(w http.ResponseWriter, r *http.Request) {
	var requestAttr LoadBalancerRule
	err := json.NewDecoder(r.Body).Decode(&requestAttr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	plog.Infof("Set LB Policy request : %+v", requestAttr)

	lB.SetPolicy(requestAttr.ServiceSrc, requestAttr.ServiceDst, requestAttr.Policy, requestAttr.DefaultMbg)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func (lB *LoadBalancer) DeletePolicyReq(w http.ResponseWriter, r *http.Request) {
	var requestAttr LoadBalancerRule
	err := json.NewDecoder(r.Body).Decode(&requestAttr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	plog.Infof("Delete LB Policy request : %+v", requestAttr)

	lB.deletePolicy(requestAttr.ServiceSrc, requestAttr.ServiceDst, requestAttr.Policy, requestAttr.DefaultMbg)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func (lB *LoadBalancer) GetPolicyReq(w http.ResponseWriter, r *http.Request) {
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

/*********************  LodBalancer functions ***************************************************/
func (lB *LoadBalancer) AddToServiceMap(serviceDst string, mbg string) {
	if mbgs, ok := lB.ServiceMap[serviceDst]; ok {
		*mbgs = append(*mbgs, mbg)
		lB.ServiceMap[serviceDst] = mbgs
	} else {
		lB.ServiceMap[serviceDst] = &([]string{mbg})
		lB.ServiceStateMap[serviceDst] = make(map[string]*ServiceState)
		lB.ServiceStateMap[serviceDst][event.Wildcard] = &ServiceState{totalConnections: 0, defaultMbg: mbg}
	}
	llog.Infof("Remote serviceDst added %v->[%+v]", serviceDst, *(lB.ServiceMap[serviceDst]))
}

<<<<<<< HEAD
func (lB *LoadBalancer) RemoveMbgFromServiceMap(mbg string) {
	for svc, mbgs := range lB.ServiceMap {
		index := -1
		for i, mbgVal := range *mbgs {
			if mbg == mbgVal {
				index = i
				break
			}
		}
		if index == -1 {
			continue
		}
		*mbgs = append((*mbgs)[:index], (*mbgs)[index+1:]...)
		llog.Infof("MBG removed from service %v->[%+v]", svc, *(lB.ServiceMap[svc]))
	}
}

func (lB *LoadBalancer) SetPolicy(serviceSrc, serviceDst string, policy PolicyLoadBalancer, defaultMbg string) {
	plog.Infof("Set LB policy %v for serviceSrc %+v serviceDst %+v defaultMbg %+v", policy, serviceSrc, serviceDst, defaultMbg)

	if policy == Static && !lB.checkMbgExist(serviceDst, defaultMbg) {
		llog.Errorf("Remote service  %v is not exist in [%+v]", serviceDst, defaultMbg)
		defaultMbg = ""
	}

	if _, ok := lB.Policy[serviceDst]; !ok { //Create default service if destination service is not exist
		lB.Policy[serviceDst] = make(map[string]PolicyLoadBalancer)
	}
	//start to update policy
	lB.Policy[serviceDst][serviceSrc] = policy
	if serviceDst != event.Wildcard { //ServiceStateMap[dst][*] is created only when the remote service is exposed
		lB.ServiceStateMap[serviceDst][serviceSrc] = &ServiceState{totalConnections: 0, defaultMbg: defaultMbg}
	}
	if serviceDst != event.Wildcard && serviceSrc == event.Wildcard { //for [dst][*] update only defaultMbg
		lB.ServiceStateMap[serviceDst][serviceSrc].defaultMbg = defaultMbg
	}
}

func (lB *LoadBalancer) deletePolicy(serviceSrc, serviceDst string, policy PolicyLoadBalancer, defaultMbg string) {
	plog.Infof("Delete LB policy %v for serviceSrc %+v serviceDst %+v defaultMbg %+v", policy, serviceSrc, serviceDst, defaultMbg)
	if _, ok := lB.Policy[serviceDst][serviceSrc]; ok {
		delete(lB.Policy[serviceDst], serviceSrc)
		if len(lB.Policy[serviceDst]) == 0 {
			delete(lB.Policy, serviceDst)
		}
	}

	if serviceDst != event.Wildcard && serviceSrc != event.Wildcard { //ServiceStateMap apply only we set policy for specific serviceSrc and serviceDst
		delete(lB.ServiceStateMap[serviceDst], serviceSrc)
	}
}

func (lB *LoadBalancer) updateState(serviceSrc, serviceDst string) {
	if _, ok := lB.Policy[serviceDst][serviceSrc]; ok {
		lB.ServiceStateMap[serviceDst][serviceSrc].totalConnections = lB.ServiceStateMap[serviceDst][serviceSrc].totalConnections + 1
	}
	if _, ok := lB.Policy[event.Wildcard][serviceSrc]; ok {
		lB.ServiceStateMap[event.Wildcard][serviceSrc].totalConnections = lB.ServiceStateMap[event.Wildcard][serviceSrc].totalConnections + 1
	}
	lB.ServiceStateMap[serviceDst][event.Wildcard].totalConnections = lB.ServiceStateMap[serviceDst][event.Wildcard].totalConnections + 1 //always exist
}

/*********************  Policy functions ***************************************************/
func (lB *LoadBalancer) LookupRandom(service string, mbgs []string) (string, error) {
	index := rand.Intn(len(mbgs))
	plog.Infof("LoadBalancer selects index(%d) - target MBG %s", index, mbgs[index])
	return mbgs[index], nil
}

func (lB *LoadBalancer) LookupECMP(service string, mbgs []string) (string, error) {
	index := lB.ServiceStateMap[service][event.Wildcard].totalConnections % len(mbgs)
	plog.Infof("LoadBalancer selects index(%d) - target MBG %s", index, mbgs[index])
	return mbgs[index], nil
}

func (lB *LoadBalancer) LookupStatic(serviceSrc, serviceDst string, mbgs []string) (string, error) {
	mbg := lB.getDefaultMbg(serviceSrc, serviceDst)
	plog.Infof("LookupStatic: serviceSrc %s serviceDst %s selects defaultMbg %s - target MBG %s", serviceSrc, serviceDst, mbg, mbgs)
	for _, m := range mbgs {
		if m == mbg {
			plog.Infof("LoadBalancer selects - target MBG %s", mbg)
			return mbg, nil
		}
	}
	return "", fmt.Errorf("No available target MBG")
}

func (lB *LoadBalancer) LookupWith(serviceSrc, serviceDst string, mbgs []string) (string, error) {
	policy := lB.getPolicy(serviceSrc, serviceDst)

	lB.updateState(serviceSrc, serviceDst)
	plog.Infof("LoadBalancer lookup for serviceSrc %s serviceDst %s with policy %s with %+v", serviceSrc, serviceDst, policy, mbgs)

	if len(mbgs) == 0 {
		return "", fmt.Errorf("No available target MBG")
	}

	switch policy {
	case Random:
		return lB.LookupRandom(serviceDst, mbgs)
	case Ecmp:
		return lB.LookupECMP(serviceDst, mbgs)
	case Static:
		return lB.LookupStatic(serviceSrc, serviceDst, mbgs)
	default:
		return lB.LookupRandom(serviceDst, mbgs)
	}
}
func (lB *LoadBalancer) getPolicy(serviceSrc, serviceDst string) PolicyLoadBalancer {
	if p, ok := lB.Policy[serviceDst][serviceSrc]; ok {
		return p
	} else if p, ok := lB.Policy[event.Wildcard][serviceSrc]; ok {
		return p
	} else if p, ok := lB.Policy[serviceDst][event.Wildcard]; ok {
		return p
	} else {
		return lB.Policy[event.Wildcard][event.Wildcard]
	}
}

func (lB *LoadBalancer) getDefaultMbg(serviceSrc, serviceDst string) string {
	if _, ok := lB.Policy[serviceDst]; ok {
		if _, ok := lB.Policy[serviceDst][serviceSrc]; ok {
			return lB.ServiceStateMap[serviceDst][serviceSrc].defaultMbg
		} else {
			return lB.ServiceStateMap[serviceDst][event.Wildcard].defaultMbg
		}
	} else {
		plog.Errorf("Lookup policy for destination service (%s) that doesn't exist", serviceDst)
		return ""
	}
}

func (lB *LoadBalancer) GetTargetMbgs(service string) ([]string, error) {
	mbgList := lB.ServiceMap[service]
	if mbgList == nil {
		plog.Errorf("Unable to find MBG for %s", service)
		return []string{}, fmt.Errorf("No available target MBG")
	}
	return *mbgList, nil
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
