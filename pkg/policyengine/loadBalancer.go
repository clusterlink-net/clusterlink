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
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"

	crds "github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
)

var llog = logrus.WithField("component", "LoadBalancer")

type LBScheme string

const (
	Random     LBScheme = "random"
	RoundRobin LBScheme = "round-robin"
	Static     LBScheme = "static"
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
	services map[types.NamespacedName]*serviceState // Keeps a state for each imported service
}

// NewLoadBalancer returns a new instance of a LoadBalancer object.
func NewLoadBalancer() *LoadBalancer {
	return &LoadBalancer{services: map[types.NamespacedName]*serviceState{}}
}

// AddImport adds a new remote service for the load balancer to take decisions on.
func (lb *LoadBalancer) AddImport(imp *crds.Import) {
	namespacedName := types.NamespacedName{Namespace: imp.Namespace, Name: imp.Name}
	state, exists := lb.services[namespacedName]
	if !exists {
		lb.services[namespacedName] = &serviceState{
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
	delete(lb.services, impName)
}

func nsNameFromFullName(fullName string) types.NamespacedName {
	parts := strings.SplitN(fullName, string(types.Separator), 2)
	if len(parts) == 2 {
		return types.NamespacedName{Namespace: parts[0], Name: parts[1]}
	}
	return types.NamespacedName{Name: fullName}
}

// SetPolicy is being used by the CRUD interface and is now deprecated.
func (lb *LoadBalancer) SetPolicy(lbPolicy *LBPolicy) error {
	plog.Infof("Set LB policy %+v", lbPolicy)

	state, ok := lb.services[nsNameFromFullName(lbPolicy.ServiceDst)]
	if !ok {
		return fmt.Errorf("service %s was not imported yet", lbPolicy.ServiceDst)
	}
	state.scheme = lbPolicy.Scheme

	return nil
}

// DeletePolicy is being used by the CRUD interface and is now deprecated.
func (lb *LoadBalancer) DeletePolicy(lbPolicy *LBPolicy) error {
	plog.Infof("Delete LB policy %+v", lbPolicy)

	state, ok := lb.services[nsNameFromFullName(lbPolicy.ServiceDst)]
	if !ok {
		return fmt.Errorf("service %s was not imported yet", lbPolicy.ServiceDst)
	}
	state.scheme = Random // back to default

	return nil
}

func (lb *LoadBalancer) lookupRandom(svc types.NamespacedName, svcSrcs []crds.ImportSource) *crds.ImportSource {
	index := rand.Intn(len(svcSrcs)) //nolint:gosec // G404: use of weak random is fine for load balancing
	plog.Infof("LoadBalancer selects index(%d) - source %v for service %s", index, svcSrcs[index], svc)
	return &svcSrcs[index]
}

func (lb *LoadBalancer) lookupRoundRobin(svc types.NamespacedName, svcSrcs []crds.ImportSource) *crds.ImportSource {
	index := lb.services[svc].totalConnections % len(svcSrcs)
	plog.Infof("LoadBalancer selects index(%d) - service source %v", index, svcSrcs[index])
	return &svcSrcs[index]
}

func (lb *LoadBalancer) lookupStatic(svc types.NamespacedName, svcSrcs []crds.ImportSource) *crds.ImportSource {
	srcs := lb.services[svc].impSources
	if len(srcs) == 0 { // shouldn't happen
		plog.Errorf("No sources for service %s. Resorting to random.", svc)
		return lb.lookupRandom(svc, svcSrcs)
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
		"Falling back to other sources due to unavailability of default source", svc)
	return lb.lookupRandom(svc, svcSrcs)
}

// LookupWith decides which service-source to use for a given outgoing-connection request.
// The decision is based on the policy set for the service, and on its locally stored state.
func (lb *LoadBalancer) LookupWith(svc types.NamespacedName, svcSrcs []crds.ImportSource) (*crds.ImportSource, error) {
	if len(svcSrcs) == 0 {
		return nil, fmt.Errorf("no available sources for service %s", svc.String())
	}

	svcState, ok := lb.services[svc]
	if !ok {
		return nil, fmt.Errorf("unknown target service %s", svc.String())
	}

	svcState.totalConnections++

	switch svcState.scheme {
	case Random:
		return lb.lookupRandom(svc, svcSrcs), nil
	case RoundRobin:
		return lb.lookupRoundRobin(svc, svcSrcs), nil
	case Static:
		return lb.lookupStatic(svc, svcSrcs), nil
	default:
		return lb.lookupRandom(svc, svcSrcs), nil
	}
}

// GetSvcSources returns all known sources for a given service in a slice of ImportSource objects.
func (lb *LoadBalancer) GetSvcSources(svc types.NamespacedName) ([]crds.ImportSource, error) {
	svcState, ok := lb.services[svc]
	if !ok || len(svcState.impSources) == 0 {
		err := fmt.Errorf("no available sources for service %s", svc.String())
		plog.Error(err.Error())
		return nil, err
	}
	return svcState.impSources, nil
}
