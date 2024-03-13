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
	"k8s.io/apimachinery/pkg/types"

	crds "github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
)

var llog = logrus.WithField("component", "LoadBalancer")

type LBScheme string

const (
	Random LBScheme = "random"
	ECMP   LBScheme = "ecmp"
	Static LBScheme = "static"
)

// deprecated.
type LBPolicy struct {
	ServiceSrc  string
	ServiceDst  string
	Scheme      LBScheme
	DefaultPeer string
}

type serviceState struct {
	scheme           LBScheme
	impSources       []crds.ImportSource
	totalConnections int
}

type LoadBalancer struct {
	services map[string]*serviceState // State of policy Per destination and source
}

func NewLoadBalancer() *LoadBalancer {
	return &LoadBalancer{services: map[string]*serviceState{}}
}

func (lb *LoadBalancer) AddImport(imp *crds.Import) {
	fullName := types.NamespacedName{Namespace: imp.Namespace, Name: imp.Name}.String()
	lb.services[fullName] = &serviceState{
		scheme:           LBScheme(imp.Spec.LBScheme),
		impSources:       imp.Spec.Sources,
		totalConnections: 0,
	}
}

func (lb *LoadBalancer) DeleteImport(impName types.NamespacedName) {
	delete(lb.services, impName.String())
}

// Deprecated.
func (lb *LoadBalancer) AddToServiceMap(serviceDst, peer string) {
	importSrc := crds.ImportSource{Peer: peer, ExportName: serviceDst}
	state, ok := lb.services[serviceDst]
	if ok {
		state.impSources = append(state.impSources, importSrc)
	} else {
		lb.services[serviceDst] = &serviceState{
			scheme:           Random,
			impSources:       []crds.ImportSource{importSrc},
			totalConnections: 0,
		}
	}

	llog.Infof("Remote serviceDst added %v->[%+v]", serviceDst, lb.services[serviceDst])
}

// deprecated.
func (lb *LoadBalancer) SetPolicy(lbPolicy *LBPolicy) error {
	plog.Infof("Set LB policy %+v", lbPolicy)

	state, ok := lb.services[lbPolicy.ServiceDst]
	if !ok {
		return fmt.Errorf("service %s was not imported yet", lbPolicy.ServiceDst)
	}
	state.scheme = lbPolicy.Scheme

	return nil
}

// deprecated.
func (lb *LoadBalancer) DeletePolicy(lbPolicy *LBPolicy) error {
	plog.Infof("Delete LB policy %+v", lbPolicy)

	state, ok := lb.services[lbPolicy.ServiceDst]
	if !ok {
		return fmt.Errorf("service %s was not imported yet", lbPolicy.ServiceDst)
	}
	state.scheme = Random // back to default

	return nil
}

// deprecated.
func (lb *LoadBalancer) RemoveDestService(serviceDst, peer string) {
	state, ok := lb.services[serviceDst]
	if !ok {
		return
	}

	newSrcs := []crds.ImportSource{}
	for _, src := range state.impSources {
		if src.Peer != peer {
			newSrcs = append(newSrcs, src)
		}
	}

	state.impSources = newSrcs
}

func (lb *LoadBalancer) LookupRandom(svcFullName string, targets []crds.ImportSource) (*crds.ImportSource, error) {
	index := rand.Intn(len(targets)) //nolint:gosec // G404: use of weak random is fine for load balancing
	plog.Infof("LoadBalancer selects index(%d) - target peer %v for service %s", index, targets[index], svcFullName)
	return &targets[index], nil
}

func (lb *LoadBalancer) LookupECMP(svcFullName string, targets []crds.ImportSource) (*crds.ImportSource, error) {
	index := lb.services[svcFullName].totalConnections % len(targets)
	plog.Infof("LoadBalancer selects index(%d) - target service %v", index, targets[index])
	return &targets[index], nil
}

func (lb *LoadBalancer) LookupWith(svc types.NamespacedName, targets []crds.ImportSource) (*crds.ImportSource, error) {
	if len(targets) == 0 {
		return nil, fmt.Errorf("no available targets for service %s", svc.String())
	}

	svcFullName := svc.String()
	svcState, ok := lb.services[svcFullName]
	if !ok {
		return nil, fmt.Errorf("unknown target service %s", svc.String())
	}

	svcState.totalConnections++

	switch svcState.scheme {
	case Random:
		return lb.LookupRandom(svcFullName, targets)
	case ECMP:
		return lb.LookupECMP(svcFullName, targets)
	case Static:
		return &targets[0], nil
	default:
		return lb.LookupRandom(svcFullName, targets)
	}
}

func (lb *LoadBalancer) GetTargetPeers(svc types.NamespacedName) ([]crds.ImportSource, error) {
	svcState, ok := lb.services[svc.String()]
	if !ok || len(svcState.impSources) == 0 {
		plog.Errorf("Unable to find peer for %s", svc.String())
		return nil, fmt.Errorf("no available target peers")
	}
	return svcState.impSources, nil
}
