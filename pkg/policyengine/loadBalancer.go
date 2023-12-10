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
	"fmt"
	"math/rand"

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
	ServiceMap      map[string][]string                 // Service to Peers
	Scheme          map[string](map[string]LBScheme)    // PolicyMap [serviceDst][serviceSrc]Policy
	ServiceStateMap map[string]map[string]*ServiceState // State of policy Per destination and source
}

func NewLoadBalancer() *LoadBalancer {
	lb := &LoadBalancer{
		ServiceMap:      make(map[string][]string),
		Scheme:          make(map[string](map[string]LBScheme)),
		ServiceStateMap: make(map[string](map[string]*ServiceState)),
	}

	lb.Scheme[event.Wildcard] = map[string]LBScheme{event.Wildcard: Random} // default policy
	lb.ServiceStateMap[event.Wildcard] = map[string]*ServiceState{event.Wildcard: {}}
	return lb
}

func (lB *LoadBalancer) AddToServiceMap(serviceDst string, peer string) {
	if peers, ok := lB.ServiceMap[serviceDst]; ok {
		_, exist := exists(peers, peer)
		if !exist {
			lB.ServiceMap[serviceDst] = append(peers, peer)
		}
	} else {
		lB.ServiceMap[serviceDst] = []string{peer}
		lB.ServiceStateMap[serviceDst] = make(map[string]*ServiceState)
		lB.ServiceStateMap[serviceDst][event.Wildcard] = &ServiceState{totalConnections: 0, defaultPeer: peer}
	}
	llog.Infof("Remote serviceDst added %v->[%+v]", serviceDst, lB.ServiceMap[serviceDst])
}

func (lB *LoadBalancer) RemovePeerFromServiceMap(peer string) {
	for svc := range lB.ServiceMap {
		lB.removePeerFromService(svc, peer)
	}
}

func (lB *LoadBalancer) removePeerFromService(svc, peer string) {
	if peers, ok := lB.ServiceMap[svc]; ok {
		index, exist := exists(peers, peer)
		if !exist {
			return
		}
		lB.ServiceMap[svc] = append(peers[:index], peers[index+1:]...)
		llog.Infof("Peer removed from service %v->[%+v]", svc, lB.ServiceMap[svc])
	}
}

func (lB *LoadBalancer) SetPolicy(lbPolicy *LBPolicy) error {
	plog.Infof("Set LB policy %+v", lbPolicy)

	defaultPeer := lbPolicy.DefaultPeer
	serviceSrc := lbPolicy.ServiceSrc
	serviceDst := lbPolicy.ServiceDst
	scheme := lbPolicy.Scheme
	if scheme == Static && !lB.checkPeerExist(serviceDst, defaultPeer) {
		err := fmt.Errorf("remote service  %v does not exist in [%+v]", serviceDst, defaultPeer)
		llog.Errorf(err.Error())
		return err
	}

	if _, ok := lB.Scheme[serviceDst]; !ok {
		lB.Scheme[serviceDst] = make(map[string]LBScheme)
	}
	lB.Scheme[serviceDst][serviceSrc] = scheme

	if _, ok := lB.ServiceStateMap[serviceDst]; !ok {
		lB.ServiceStateMap[serviceDst] = make(map[string]*ServiceState)
	}
	lB.ServiceStateMap[serviceDst][serviceSrc] = &ServiceState{totalConnections: 0, defaultPeer: defaultPeer}

	return nil
}

func (lB *LoadBalancer) DeletePolicy(lbPolicy *LBPolicy) error {
	plog.Infof("Delete LB policy %+v", lbPolicy)

	serviceSrc := lbPolicy.ServiceSrc
	serviceDst := lbPolicy.ServiceDst

	if serviceSrc == event.Wildcard && serviceDst == event.Wildcard {
		return fmt.Errorf("default policy cannot be deleted")
	}

	if _, ok := lB.Scheme[serviceDst][serviceSrc]; ok {
		delete(lB.Scheme[serviceDst], serviceSrc)
		if len(lB.Scheme[serviceDst]) == 0 {
			delete(lB.Scheme, serviceDst)
		}
	} else {
		return fmt.Errorf("failed to delete a non-existing load-balancing policy")
	}

	if serviceDst != event.Wildcard && serviceSrc != event.Wildcard { // ServiceStateMap apply only we set policy for specific serviceSrc and serviceDst
		delete(lB.ServiceStateMap[serviceDst], serviceSrc)
	}
	return nil
}

func (lB *LoadBalancer) RemoveDestService(serviceDst, peer string) {
	if peer != "" {
		lB.removePeerFromService(serviceDst, peer)
	} else {
		delete(lB.ServiceMap, serviceDst)
	}
}

func (lB *LoadBalancer) updateState(serviceSrc, serviceDst string) {
	if _, ok := lB.ServiceStateMap[serviceDst][serviceSrc]; ok {
		lB.ServiceStateMap[serviceDst][serviceSrc].totalConnections++
	}
	if _, ok := lB.ServiceStateMap[serviceDst][event.Wildcard]; ok {
		lB.ServiceStateMap[serviceDst][event.Wildcard].totalConnections++ // may not exist if dst is not imported yet
	}
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
	policy := lB.getScheme(serviceSrc, serviceDst)

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

func (lB *LoadBalancer) getScheme(serviceSrc, serviceDst string) LBScheme {
	if p, ok := lB.Scheme[serviceDst][serviceSrc]; ok {
		return p
	} else if p, ok := lB.Scheme[event.Wildcard][serviceSrc]; ok {
		return p
	} else if p, ok := lB.Scheme[serviceDst][event.Wildcard]; ok {
		return p
	} else {
		return lB.Scheme[event.Wildcard][event.Wildcard]
	}
}

func (lB *LoadBalancer) getDefaultPeer(serviceSrc, serviceDst string) string {
	if _, ok := lB.ServiceStateMap[serviceDst]; ok {
		if _, ok := lB.ServiceStateMap[serviceDst][serviceSrc]; ok {
			return lB.ServiceStateMap[serviceDst][serviceSrc].defaultPeer
		}
		return lB.ServiceStateMap[serviceDst][event.Wildcard].defaultPeer
	}
	plog.Errorf("Lookup policy for destination service (%s) that doesn't exist", serviceDst)
	return ""
}

func (lB *LoadBalancer) GetTargetPeers(service string) ([]string, error) {
	peerList := lB.ServiceMap[service]
	if len(peerList) == 0 {
		plog.Errorf("Unable to find peer for %s", service)
		return []string{}, fmt.Errorf("no available target peer")
	}
	return peerList, nil
}

func (lB *LoadBalancer) checkPeerExist(service, peer string) bool {
	peerList := lB.ServiceMap[service]
	_, exist := exists(peerList, peer)
	return exist
}

func exists(slice []string, entry string) (int, bool) {
	for i, e := range slice {
		if e == entry {
			return i, true
		}
	}
	return -1, false
}
