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
const (
	defaultSrcPolicy = "defaultPolicy"
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
	ServiceStateMap map[string]map[string]*ServiceState        // Per destination service
	defaultPolicy   PolicyLoadBalancer
}

func (lB *LoadBalancer) Init() {
	lB.ServiceMap = make(map[string]*[]string)
	lB.Policy = make(map[string](map[string]PolicyLoadBalancer))
	lB.ServiceStateMap = make(map[string]map[string]*ServiceState)
	lB.defaultPolicy = Random
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
		lB.setPolicy2Service(defaultSrcPolicy, serviceDst, lB.defaultPolicy, "") //set default random policy

	}
	llog.Infof("Remote serviceDst added %v->[%+v]", serviceDst, *(lB.ServiceMap[serviceDst]))
}

func (lB *LoadBalancer) SetPolicy(serviceSrc, serviceDst string, policy PolicyLoadBalancer, defaultMbg string) {
	if lB.Policy == nil {
		lB.Init()
	}

	if serviceDst == event.Wildcard {
		for d, val := range lB.Policy {
			if serviceSrc == event.Wildcard {
				for s, _ := range val {
					lB.setPolicy2Service(s, d, policy, defaultMbg)
				}
			} else {
				lB.setPolicy2Service(serviceSrc, d, policy, defaultMbg)
			}
		}
	} else if serviceSrc == event.Wildcard { //&& serviceDst != event.Wildcard
		if _, ok := lB.Policy[serviceDst]; !ok { //for case the destService is not exist
			lB.setPolicy2Service(serviceSrc, serviceDst, policy, defaultMbg)
		} else {
			for s, _ := range lB.Policy[serviceDst] {
				lB.setPolicy2Service(s, serviceDst, policy, defaultMbg)
			}
		}
	} else { //serviceSrc != event.Wildcard && serviceDst != event.Wildcard
		lB.setPolicy2Service(serviceSrc, serviceDst, policy, defaultMbg)
	}
}

func (lB *LoadBalancer) setPolicy2Service(serviceSrc, serviceDst string, policy PolicyLoadBalancer, defaultMbg string) {
	plog.Infof("Set LB policy %v for serviceSrc %+v serviceDst %+v defaultMbg %+v", policy, serviceSrc, serviceDst, defaultMbg)

	if policy == Static && !lB.checkMbgExist(serviceDst, defaultMbg) {
		llog.Errorf("Remote service  %v is not exist in [%+v]", serviceDst, defaultMbg)
		defaultMbg = ""
	}

	plog.Infof("Set LB policy %v for serviceSrc %+v serviceDst %+v defaultMbg %+v", policy, serviceSrc, serviceDst, defaultMbg)
	if _, ok := lB.Policy[serviceDst]; !ok { //Create default service if destination service is not exist
		lB.Policy[serviceDst] = make(map[string]PolicyLoadBalancer)
		lB.Policy[serviceDst][defaultSrcPolicy] = policy
		lB.ServiceStateMap[serviceDst] = make(map[string]*ServiceState)
		lB.ServiceStateMap[serviceDst][defaultSrcPolicy] = &ServiceState{totalConnections: 0, defaultMbg: defaultMbg}
	}

	if serviceSrc == event.Wildcard { //Can happen only if the default is not exist
		return
	}

	//start to update policy
	lB.Policy[serviceDst][serviceSrc] = policy
	lB.ServiceStateMap[serviceDst][serviceSrc] = &ServiceState{totalConnections: 0, defaultMbg: defaultMbg}

}

func (lB *LoadBalancer) updateState(serviceSrc, serviceDst string) {
	lB.ServiceStateMap[serviceDst][defaultSrcPolicy].totalConnections = lB.ServiceStateMap[serviceDst][defaultSrcPolicy].totalConnections + 1
	if _, ok := lB.Policy[serviceDst][serviceSrc]; ok {
		lB.ServiceStateMap[serviceDst][serviceSrc].totalConnections = lB.ServiceStateMap[serviceDst][serviceSrc].totalConnections + 1
	}
}

/*********************  Policy functions ***************************************************/
func (lB *LoadBalancer) LookupRandom(service string, mbgs []string) (string, error) {
	index := rand.Intn(len(mbgs))
	plog.Infof("LoadBalancer selects index(%d) - target MBG %s", index, mbgs[index])
	return mbgs[index], nil
}

func (lB *LoadBalancer) LookupECMP(service string, mbgs []string) (string, error) {
	index := lB.ServiceStateMap[service][defaultSrcPolicy].totalConnections % len(mbgs)
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
	if _, ok := lB.Policy[serviceDst]; ok {
		if p, ok := lB.Policy[serviceDst][serviceSrc]; ok {
			return p
		} else {
			return lB.Policy[serviceDst][defaultSrcPolicy]
		}
	} else {
		plog.Errorf("Lookup policy for destination service (%s) that doesn't exist", serviceDst)
		return ""
	}
}
func (lB *LoadBalancer) getDefaultMbg(serviceSrc, serviceDst string) string {
	if _, ok := lB.Policy[serviceDst]; ok {
		if _, ok := lB.Policy[serviceDst][serviceSrc]; ok {
			return lB.ServiceStateMap[serviceDst][serviceSrc].defaultMbg
		} else {
			return lB.ServiceStateMap[serviceDst][defaultSrcPolicy].defaultMbg
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
