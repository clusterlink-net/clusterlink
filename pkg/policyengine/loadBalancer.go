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

// LBPolicy is being used by the CRUD interface, and is now deprecated.
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
	services map[string]*serviceState // Keeps a state for each imported service
}

// NewLoadBalancer returns a new instance of a LoadBalancer object.
func NewLoadBalancer() *LoadBalancer {
	return &LoadBalancer{services: map[string]*serviceState{}}
}

// AddImport adds a new remote service for the load balancer to take decisions on.
func (lb *LoadBalancer) AddImport(imp *crds.Import) {
	fullName := types.NamespacedName{Namespace: imp.Namespace, Name: imp.Name}.String()
	state, exists := lb.services[fullName]
	if !exists {
		lb.services[fullName] = &serviceState{
			scheme:           LBScheme(imp.Spec.LBScheme),
			impSources:       imp.Spec.Sources,
			totalConnections: 0,
		}
	} else {
		state.impSources = imp.Spec.Sources
		if imp.Spec.LBScheme != "" {
			state.scheme = LBScheme(imp.Spec.LBScheme)
		}
	}
}

// DeleteImport removes a remote service from the list of services the load balancer reasons about.
func (lb *LoadBalancer) DeleteImport(impName types.NamespacedName) {
	delete(lb.services, impName.String())
}

// AddToServiceMap is being used by the CRUD interface and is now deprecated.
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

// SetPolicy is being used by the CRUD interface and is now deprecated.
func (lb *LoadBalancer) SetPolicy(lbPolicy *LBPolicy) error {
	plog.Infof("Set LB policy %+v", lbPolicy)

	state, ok := lb.services[lbPolicy.ServiceDst]
	if !ok {
		return fmt.Errorf("service %s was not imported yet", lbPolicy.ServiceDst)
	}
	state.scheme = lbPolicy.Scheme

	return nil
}

// DeletePolicy is being used by the CRUD interface and is now deprecated.
func (lb *LoadBalancer) DeletePolicy(lbPolicy *LBPolicy) error {
	plog.Infof("Delete LB policy %+v", lbPolicy)

	state, ok := lb.services[lbPolicy.ServiceDst]
	if !ok {
		return fmt.Errorf("service %s was not imported yet", lbPolicy.ServiceDst)
	}
	state.scheme = Random // back to default

	return nil
}

// RemoveDestService is being used by the CRUD interface and is now deprecated.
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

func (lb *LoadBalancer) lookupRandom(svcFullName string, svcSrcs []crds.ImportSource) *crds.ImportSource {
	index := rand.Intn(len(svcSrcs)) //nolint:gosec // G404: use of weak random is fine for load balancing
	plog.Infof("LoadBalancer selects index(%d) - source %v for service %s", index, svcSrcs[index], svcFullName)
	return &svcSrcs[index]
}

func (lb *LoadBalancer) lookupECMP(svcFullName string, svcSrcs []crds.ImportSource) *crds.ImportSource {
	index := lb.services[svcFullName].totalConnections % len(svcSrcs)
	plog.Infof("LoadBalancer selects index(%d) - service source %v", index, svcSrcs[index])
	return &svcSrcs[index]
}

func (lb *LoadBalancer) lookupStatic(svcFullName string, svcSrcs []crds.ImportSource) *crds.ImportSource {
	srcs := lb.services[svcFullName].impSources
	if len(srcs) == 0 { // shouldn't happen
		plog.Errorf("No sources for service %s. Resorting to random.", svcFullName)
		return lb.lookupRandom(svcFullName, svcSrcs)
	}

	defaultSrc := srcs[0]
	for i := range svcSrcs { // ensure default is in the list
		tgt := &svcSrcs[i]
		if tgt.ExportNamespace == defaultSrc.ExportNamespace && tgt.ExportName == defaultSrc.ExportName &&
			tgt.Peer == defaultSrc.Peer {
			plog.Infof("LoadBalancer selected default service source %v", defaultSrc)
			return tgt
		}
	}

	plog.Errorf("Default source for service %s does not exist. "+
		"Falling back to other sources due to unavailability of default source", svcFullName)
	return lb.lookupRandom(svcFullName, svcSrcs)
}

// LookupWith decides which service-source to use for a given outgoing-connection request.
// The decision is based on the policy set for the service, and on its locally stored state.
func (lb *LoadBalancer) LookupWith(svc types.NamespacedName, svcSrcs []crds.ImportSource) (*crds.ImportSource, error) {
	if len(svcSrcs) == 0 {
		return nil, fmt.Errorf("no available sources for service %s", svc.String())
	}

	svcFullName := svc.String()
	svcState, ok := lb.services[svcFullName]
	if !ok {
		return nil, fmt.Errorf("unknown target service %s", svc.String())
	}

	svcState.totalConnections++

	switch svcState.scheme {
	case Random:
		return lb.lookupRandom(svcFullName, svcSrcs), nil
	case ECMP:
		return lb.lookupECMP(svcFullName, svcSrcs), nil
	case Static:
		return lb.lookupStatic(svcFullName, svcSrcs), nil
	default:
		return lb.lookupRandom(svcFullName, svcSrcs), nil
	}
}

// GetSvcSources returns all known sources for a given service in a slice of ImportSource objects.
func (lb *LoadBalancer) GetSvcSources(svc types.NamespacedName) ([]crds.ImportSource, error) {
	svcState, ok := lb.services[svc.String()]
	if !ok || len(svcState.impSources) == 0 {
		err := fmt.Errorf("no available sources for service %s", svc.String())
		plog.Error(err.Error())
		return nil, err
	}
	return svcState.impSources, nil
}
