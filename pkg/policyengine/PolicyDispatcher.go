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

// policyengine package handles policies that govern ClusterLink behavior
package policyengine

import (
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"

	crds "github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/policyengine/connectivitypdp"
)

const (
	ServiceNameLabel = "clusterlink/metadata.serviceName"
	GatewayNameLabel = "clusterlink/metadata.gatewayName"
)

var plog = logrus.WithField("component", "PolicyEngine")

// PolicyDecider is an interface for entities that make policy-based decisions on various ClusterLink operations.
type PolicyDecider interface {
	AddAccessPolicy(policy *connectivitypdp.AccessPolicy) error
	DeleteAccessPolicy(name types.NamespacedName, privileged bool) error

	AuthorizeAndRouteConnection(connReq *connectivitypdp.ConnectionRequest) (connectivitypdp.ConnectionResponse, error)

	AddPeer(name string)
	DeletePeer(name string)

	AddImport(imp *crds.Import)
	DeleteImport(name types.NamespacedName)

	AddExport(exp *crds.Export) ([]string, error) // Returns a list of peers to which export is allowed
	DeleteExport(name string)
}

// PolicyHandler implements PolicyDecider using Connectivity Policies and Load-Balancing Policies.
type PolicyHandler struct {
	loadBalancer    *LoadBalancer
	connectivityPDP *connectivitypdp.PDP
	enabledPeers    map[string]bool
}

func NewPolicyHandler() PolicyDecider {
	return &PolicyHandler{
		loadBalancer:    NewLoadBalancer(),
		connectivityPDP: connectivitypdp.NewPDP(),
		enabledPeers:    map[string]bool{},
	}
}

func getServiceAttrs(serviceName, peer string) connectivitypdp.WorkloadAttrs {
	ret := connectivitypdp.WorkloadAttrs{ServiceNameLabel: serviceName}
	if len(peer) > 0 {
		ret[GatewayNameLabel] = peer
	}
	return ret
}

func getServiceAttrsForMultipleDsts(serviceName string, dsts []crds.ImportSource) []connectivitypdp.WorkloadAttrs {
	res := []connectivitypdp.WorkloadAttrs{}
	for _, dst := range dsts {
		res = append(res, getServiceAttrs(serviceName, dst.Peer))
	}
	return res
}

func (ph *PolicyHandler) filterOutDisabledPeers(dsts []crds.ImportSource) []crds.ImportSource {
	res := []crds.ImportSource{}
	for _, dst := range dsts {
		if ph.enabledPeers[dst.Peer] {
			res = append(res, dst)
		}
	}
	return res
}

func (ph *PolicyHandler) decideIncomingConnection(
	req *connectivitypdp.ConnectionRequest,
) (connectivitypdp.ConnectionResponse, error) {
	dest := getServiceAttrs(req.DstSvcName, "")
	decisions, err := ph.connectivityPDP.Decide(req.SrcWorkloadAttrs, []connectivitypdp.WorkloadAttrs{dest},
		req.DstSvcNamespace)
	if err != nil {
		plog.Errorf("error deciding on a connection: %v", err)
		return connectivitypdp.ConnectionResponse{Action: crds.AccessPolicyActionDeny}, err
	}
	if decisions[0].Decision == connectivitypdp.DecisionAllow {
		return connectivitypdp.ConnectionResponse{Action: crds.AccessPolicyActionAllow}, nil
	}
	return connectivitypdp.ConnectionResponse{Action: crds.AccessPolicyActionDeny}, nil
}

func (ph *PolicyHandler) decideOutgoingConnection(
	req *connectivitypdp.ConnectionRequest,
) (connectivitypdp.ConnectionResponse, error) {
	// Get a list of possible destinations for the service (a.k.a. service sources)
	dstSvcNsName := types.NamespacedName{Namespace: req.DstSvcNamespace, Name: req.DstSvcName}
	svcSourceList, err := ph.loadBalancer.GetSvcSources(dstSvcNsName)
	if err != nil {
		plog.Errorf("error getting sources for service %s: %v", req.DstSvcName, err)
		// this can be caused by a user typo - so only log this error
		return connectivitypdp.ConnectionResponse{Action: crds.AccessPolicyActionDeny}, nil
	}

	svcSourceList = ph.filterOutDisabledPeers(svcSourceList)

	dsts := getServiceAttrsForMultipleDsts(req.DstSvcName, svcSourceList)
	decisions, err := ph.connectivityPDP.Decide(req.SrcWorkloadAttrs, dsts, req.DstSvcNamespace)
	if err != nil {
		plog.Errorf("error deciding on a connection: %v", err)
		return connectivitypdp.ConnectionResponse{Action: crds.AccessPolicyActionDeny}, err
	}

	allowedSvcSources := []crds.ImportSource{}
	for idx, decision := range decisions {
		if decision.Decision == connectivitypdp.DecisionAllow {
			allowedSvcSources = append(allowedSvcSources, svcSourceList[idx])
		}
	}

	if len(allowedSvcSources) == 0 {
		plog.Infof("access policies deny connections to service %s for all its sources", req.DstSvcName)
		return connectivitypdp.ConnectionResponse{Action: crds.AccessPolicyActionDeny}, nil
	}

	// Perform load-balancing using the filtered peer list
	tgt, err := ph.loadBalancer.LookupWith(dstSvcNsName, allowedSvcSources)
	if err != nil {
		return connectivitypdp.ConnectionResponse{Action: crds.AccessPolicyActionDeny}, err
	}
	return connectivitypdp.ConnectionResponse{
		Action:       crds.AccessPolicyActionAllow,
		DstPeer:      tgt.Peer,
		DstName:      tgt.ExportName,
		DstNamespace: tgt.ExportNamespace,
	}, nil
}

func (ph *PolicyHandler) AuthorizeAndRouteConnection(req *connectivitypdp.ConnectionRequest) (
	connectivitypdp.ConnectionResponse,
	error,
) {
	plog.Infof("New connection request : %+v", req)

	var resp connectivitypdp.ConnectionResponse
	var err error
	if req.Direction == connectivitypdp.Incoming {
		resp, err = ph.decideIncomingConnection(req)
	} else if req.Direction == connectivitypdp.Outgoing {
		resp, err = ph.decideOutgoingConnection(req)
	}

	plog.Infof("Response : %+v", resp)
	return resp, err
}

func (ph *PolicyHandler) AddPeer(name string) {
	ph.enabledPeers[name] = true
	plog.Infof("Added Peer %s", name)
}

func (ph *PolicyHandler) DeletePeer(name string) {
	delete(ph.enabledPeers, name)
	plog.Infof("Removed Peer %s", name)
}

func (ph *PolicyHandler) AddImport(imp *crds.Import) {
	ph.loadBalancer.AddImport(imp)
}

func (ph *PolicyHandler) DeleteImport(name types.NamespacedName) {
	ph.loadBalancer.DeleteImport(name)
}

func (ph *PolicyHandler) AddExport(_ *crds.Export) ([]string, error) {
	retPeers := []string{}
	for peer, enabled := range ph.enabledPeers {
		if enabled {
			retPeers = append(retPeers, peer)
		}
	}
	return retPeers, nil
}

func (ph *PolicyHandler) DeleteExport(_ string) {
}

func (ph *PolicyHandler) AddAccessPolicy(policy *connectivitypdp.AccessPolicy) error {
	return ph.connectivityPDP.AddOrUpdatePolicy(policy)
}

func (ph *PolicyHandler) DeleteAccessPolicy(name types.NamespacedName, privileged bool) error {
	return ph.connectivityPDP.DeletePolicy(name, privileged)
}
