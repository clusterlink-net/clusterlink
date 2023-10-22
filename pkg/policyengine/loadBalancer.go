// Copyright 2023 The ClusterLink Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package policyengine

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"

	"github.com/sirupsen/logrus"

	event "github.com/clusterlink-net/clusterlink/pkg/controlplane/eventmanager"
)

var llog = logrus.WithField("component", "LoadBalancer")

type LBScheme string

const (
	Random LBScheme = "random"
	ECMP   LBScheme = "ecmp"
	Static LBScheme = "static"
)

type LBPolicy struct {
	ServiceSrc  string
	ServiceDst  string
	Scheme      LBScheme
	DefaultPeer string
}

type ServiceState struct {
	totalConnections int
	defaultPeer      string
}

type LoadBalancer struct {
	ServiceMap      map[string]*[]string                // Service to Peers
	Policy          map[string](map[string]LBScheme)    // PolicyMap [serviceDst][serviceSrc]Policy
	ServiceStateMap map[string]map[string]*ServiceState // State of policy Per destination and source
}

func NewLoadBalancer() *LoadBalancer {
	lb := &LoadBalancer{
		ServiceMap:      make(map[string]*[]string),
		Policy:          make(map[string](map[string]LBScheme)),
		ServiceStateMap: make(map[string](map[string]*ServiceState)),
	}

	lb.ServiceStateMap[event.Wildcard] = make(map[string]*ServiceState)
	lb.Policy[event.Wildcard] = make(map[string]LBScheme)
	lb.Policy[event.Wildcard][event.Wildcard] = Random // default policy
	return lb
}

/*********************  HTTP functions ***************************************************/
func (lB *LoadBalancer) SetPolicyReq(w http.ResponseWriter, r *http.Request) {
	var requestAttr LBPolicy
	err := json.NewDecoder(r.Body).Decode(&requestAttr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	plog.Infof("Set LB Policy request : %+v", requestAttr)

	lB.SetPolicy(requestAttr.ServiceSrc, requestAttr.ServiceDst, requestAttr.Scheme, requestAttr.DefaultPeer)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func (lB *LoadBalancer) DeletePolicyReq(w http.ResponseWriter, r *http.Request) {
	var requestAttr LBPolicy
	err := json.NewDecoder(r.Body).Decode(&requestAttr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	plog.Infof("Delete LB Policy request : %+v", requestAttr)

	lB.deletePolicy(requestAttr.ServiceSrc, requestAttr.ServiceDst, requestAttr.Scheme, requestAttr.DefaultPeer)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func (lB *LoadBalancer) GetPolicyReq(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(lB.Policy); err != nil {
		plog.Errorf("Error happened in JSON encode. Err: %s", err)
		return
	}
}

/*********************  LodBalancer functions ***************************************************/

func (lB *LoadBalancer) AddToServiceMap(serviceDst string, peer string) {
	if peers, ok := lB.ServiceMap[serviceDst]; ok {
		_, exist := exists(*peers, peer)
		if !exist {
			*peers = append(*peers, peer)
			lB.ServiceMap[serviceDst] = peers
		}
	} else {
		lB.ServiceMap[serviceDst] = &([]string{peer})
		lB.ServiceStateMap[serviceDst] = make(map[string]*ServiceState)
		lB.ServiceStateMap[serviceDst][event.Wildcard] = &ServiceState{totalConnections: 0, defaultPeer: peer}
	}
	llog.Infof("Remote serviceDst added %v->[%+v]", serviceDst, *(lB.ServiceMap[serviceDst]))
}

func (lB *LoadBalancer) RemovePeerFromServiceMap(peer string) {
	for svc := range lB.ServiceMap {
		lB.RemovePeerFromService(svc, peer)
	}
}
func (lB *LoadBalancer) RemovePeerFromService(svc, peer string) {
	if peers, ok := lB.ServiceMap[svc]; ok {
		index, exist := exists(*peers, peer)
		if !exist {
			return
		}
		*peers = append((*peers)[:index], (*peers)[index+1:]...)
		llog.Infof("Peer removed from service %v->[%+v]", svc, *(lB.ServiceMap[svc]))
	}
}
func (lB *LoadBalancer) SetPolicy(serviceSrc, serviceDst string, policy LBScheme, defaultPeer string) {
	plog.Infof("Set LB policy %v for serviceSrc %+v serviceDst %+v defaultPeer %+v", policy, serviceSrc, serviceDst, defaultPeer)

	if policy == Static && !lB.checkPeerExist(serviceDst, defaultPeer) {
		llog.Errorf("Remote service  %v is not exist in [%+v]", serviceDst, defaultPeer)
		defaultPeer = ""
	}

	if _, ok := lB.Policy[serviceDst]; !ok { // Create default service if destination service is not exist
		lB.Policy[serviceDst] = make(map[string]LBScheme)
	}
	// start to update policy
	lB.Policy[serviceDst][serviceSrc] = policy
	if serviceDst != event.Wildcard { // ServiceStateMap[dst][*] is created only when the remote service is exposed
		lB.ServiceStateMap[serviceDst][serviceSrc] = &ServiceState{totalConnections: 0, defaultPeer: defaultPeer}
	}

	if serviceDst != event.Wildcard && serviceSrc == event.Wildcard { // for [dst][*] update only defaultPeer
		lB.ServiceStateMap[serviceDst][serviceSrc].defaultPeer = defaultPeer
	}
}

func (lB *LoadBalancer) deletePolicy(serviceSrc, serviceDst string, policy LBScheme, defaultPeer string) {
	plog.Infof("Delete LB policy %v for serviceSrc %+v serviceDst %+v defaultPeer %+v", policy, serviceSrc, serviceDst, defaultPeer)
	if _, ok := lB.Policy[serviceDst][serviceSrc]; ok {
		delete(lB.Policy[serviceDst], serviceSrc)
		if len(lB.Policy[serviceDst]) == 0 {
			delete(lB.Policy, serviceDst)
		}
	}

	if serviceDst != event.Wildcard && serviceSrc != event.Wildcard { // ServiceStateMap apply only we set policy for specific serviceSrc and serviceDst
		delete(lB.ServiceStateMap[serviceDst], serviceSrc)
	}
}

func (lB *LoadBalancer) RemoveDestService(serviceDst, peer string) {
	if peer != "" {
		lB.RemovePeerFromService(serviceDst, peer)
	} else {
		delete(lB.ServiceMap, serviceDst)
	}
}
func (lB *LoadBalancer) updateState(serviceSrc, serviceDst string) {
	if _, ok := lB.Policy[serviceDst][serviceSrc]; ok {
		lB.ServiceStateMap[serviceDst][serviceSrc].totalConnections++
	}
	if _, ok := lB.Policy[event.Wildcard][serviceSrc]; ok && serviceDst == event.Wildcard {
		lB.ServiceStateMap[event.Wildcard][serviceSrc].totalConnections++
	}
	lB.ServiceStateMap[serviceDst][event.Wildcard].totalConnections++ // always exist
}

/*********************  Policy functions ***************************************************/
func (lB *LoadBalancer) LookupRandom(service string, peers []string) (string, error) {
	index := rand.Intn(len(peers)) //nolint:gosec // G404: use of weak random is fine for load balancing
	plog.Infof("LoadBalancer selects index(%d) - target peer %s for service %s", index, peers[index], service)
	return peers[index], nil
}

func (lB *LoadBalancer) LookupECMP(service string, peers []string) (string, error) {
	index := lB.ServiceStateMap[service][event.Wildcard].totalConnections % len(peers)
	plog.Infof("LoadBalancer selects index(%d) - target peer %s", index, peers[index])
	return peers[index], nil
}

func (lB *LoadBalancer) LookupStatic(serviceSrc, serviceDst string, peers []string) (string, error) {
	peer := lB.getDefaultPeer(serviceSrc, serviceDst)
	plog.Infof("LookupStatic: serviceSrc %s serviceDst %s selects defaultPeer %s - target peer %s", serviceSrc, serviceDst, peer, peers)
	for _, m := range peers {
		if m == peer {
			plog.Infof("LoadBalancer selects - target peer %s", peer)
			return peer, nil
		}
	}
	plog.Errorf("Falling back to other peers due to unavailability of default peer")

	return lB.LookupRandom(serviceDst, peers)
}

func (lB *LoadBalancer) LookupWith(serviceSrc, serviceDst string, peers []string) (string, error) {
	policy := lB.getPolicy(serviceSrc, serviceDst)

	lB.updateState(serviceSrc, serviceDst)
	plog.Infof("LoadBalancer lookup for serviceSrc %s serviceDst %s with policy %s with %+v", serviceSrc, serviceDst, policy, peers)

	if len(peers) == 0 {
		return "", fmt.Errorf("no available target peer")
	}

	switch policy {
	case Random:
		return lB.LookupRandom(serviceDst, peers)
	case ECMP:
		return lB.LookupECMP(serviceDst, peers)
	case Static:
		return lB.LookupStatic(serviceSrc, serviceDst, peers)
	default:
		return lB.LookupRandom(serviceDst, peers)
	}
}
func (lB *LoadBalancer) getPolicy(serviceSrc, serviceDst string) LBScheme {
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

func (lB *LoadBalancer) getDefaultPeer(serviceSrc, serviceDst string) string {
	if _, ok := lB.Policy[serviceDst]; ok {
		if _, ok := lB.Policy[serviceDst][serviceSrc]; ok {
			return lB.ServiceStateMap[serviceDst][serviceSrc].defaultPeer
		}
		return lB.ServiceStateMap[serviceDst][event.Wildcard].defaultPeer
	}
	plog.Errorf("Lookup policy for destination service (%s) that doesn't exist", serviceDst)
	return ""
}

func (lB *LoadBalancer) GetTargetPeers(service string) ([]string, error) {
	peerList := lB.ServiceMap[service]
	if peerList == nil {
		plog.Errorf("Unable to find peer for %s", service)
		return []string{}, fmt.Errorf("no available target peer")
	}
	return *peerList, nil
}

func (lB *LoadBalancer) checkPeerExist(service, peer string) bool {
	peerList := lB.ServiceMap[service]
	if peerList != nil {
		for _, val := range *peerList {
			if val == peer {
				return true
			}
		}
	}
	return false
}
