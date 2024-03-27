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
	"bytes"
	"encoding/json"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"

	"github.com/clusterlink-net/clusterlink/pkg/api"
	crds "github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/policyengine/connectivitypdp"
)

const (
	LbType     = "lb"     // Type for load-balancing policies
	AccessType = "access" // Type for access policies

	ServiceNameLabel = "clusterlink/metadata.serviceName"
	GatewayNameLabel = "clusterlink/metadata.gatewayName"
)

var plog = logrus.WithField("component", "PolicyEngine")

// PolicyDecider is an interface for entities that make policy-based decisions on various ClusterLink operations.
type PolicyDecider interface {
	AddLBPolicy(policy *api.Policy) error
	DeleteLBPolicy(policy *api.Policy) error

	AddAccessPolicy(policy *api.Policy) error
	DeleteAccessPolicy(policy *api.Policy) error

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

func (pH *PolicyHandler) filterOutDisabledPeers(dsts []crds.ImportSource) []crds.ImportSource {
	res := []crds.ImportSource{}
	for _, dst := range dsts {
		if pH.enabledPeers[dst.Peer] {
			res = append(res, dst)
		}
	}
	return res
}

func (pH *PolicyHandler) decideIncomingConnection(
	req *connectivitypdp.ConnectionRequest,
) (connectivitypdp.ConnectionResponse, error) {
	dest := getServiceAttrs(req.DstSvcName, "")
	decisions, err := pH.connectivityPDP.Decide(req.SrcWorkloadAttrs, []connectivitypdp.WorkloadAttrs{dest})
	if err != nil {
		plog.Errorf("error deciding on a connection: %v", err)
		return connectivitypdp.ConnectionResponse{Action: crds.AccessPolicyActionDeny}, err
	}
	if decisions[0].Decision == connectivitypdp.DecisionAllow {
		return connectivitypdp.ConnectionResponse{Action: crds.AccessPolicyActionAllow}, nil
	}
	return connectivitypdp.ConnectionResponse{Action: crds.AccessPolicyActionDeny}, nil
}

func (pH *PolicyHandler) decideOutgoingConnection(
	req *connectivitypdp.ConnectionRequest,
) (connectivitypdp.ConnectionResponse, error) {
	// Get a list of possible destinations for the service (a.k.a. service sources)
	dstSvcNsName := types.NamespacedName{Namespace: req.DstSvcNamespace, Name: req.DstSvcName}
	svcSourceList, err := pH.loadBalancer.GetSvcSources(dstSvcNsName)
	if err != nil {
		plog.Errorf("error getting sources for service %s: %v", req.DstSvcName, err)
		// this can be caused by a user typo - so only log this error
		return connectivitypdp.ConnectionResponse{Action: crds.AccessPolicyActionDeny}, nil
	}

	svcSourceList = pH.filterOutDisabledPeers(svcSourceList)

	dsts := getServiceAttrsForMultipleDsts(req.DstSvcName, svcSourceList)
	decisions, err := pH.connectivityPDP.Decide(req.SrcWorkloadAttrs, dsts)
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
	tgt, err := pH.loadBalancer.LookupWith(dstSvcNsName, allowedSvcSources)
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

func (pH *PolicyHandler) AuthorizeAndRouteConnection(req *connectivitypdp.ConnectionRequest) (
	connectivitypdp.ConnectionResponse,
	error,
) {
	plog.Infof("New connection request : %+v", req)

	var resp connectivitypdp.ConnectionResponse
	var err error
	if req.Direction == connectivitypdp.Incoming {
		resp, err = pH.decideIncomingConnection(req)
	} else if req.Direction == connectivitypdp.Outgoing {
		resp, err = pH.decideOutgoingConnection(req)
	}

	plog.Infof("Response : %+v", resp)
	return resp, err
}

func (pH *PolicyHandler) AddPeer(name string) {
	pH.enabledPeers[name] = true
	plog.Infof("Added Peer %s", name)
}

func (pH *PolicyHandler) DeletePeer(name string) {
	delete(pH.enabledPeers, name)
	plog.Infof("Removed Peer %s", name)
}

func (pH *PolicyHandler) AddImport(imp *crds.Import) {
	pH.loadBalancer.AddImport(imp)
}

func (pH *PolicyHandler) DeleteImport(name types.NamespacedName) {
	pH.loadBalancer.DeleteImport(name)
}

func (pH *PolicyHandler) AddExport(_ *crds.Export) ([]string, error) {
	retPeers := []string{}
	for peer, enabled := range pH.enabledPeers {
		if enabled {
			retPeers = append(retPeers, peer)
		}
	}
	return retPeers, nil
}

func (pH *PolicyHandler) DeleteExport(_ string) {
}

// connPolicyFromBlob unmarshals a ConnectivityPolicy object encoded as json in a byte array.
func connPolicyFromBlob(blob []byte) (*crds.AccessPolicy, error) {
	bReader := bytes.NewReader(blob)
	connPolicy := &crds.AccessPolicy{}
	err := json.NewDecoder(bReader).Decode(connPolicy)
	if err != nil {
		plog.Errorf("failed decoding connectivity policy: %v", err)
		return nil, err
	}
	return connPolicy, nil
}

// lbPolicyFromBlob unmarshals an LBPolicy object encoded as json in a byte array.
func lbPolicyFromBlob(blob []byte) (*LBPolicy, error) {
	bReader := bytes.NewReader(blob)
	lbPolicy := &LBPolicy{}
	err := json.NewDecoder(bReader).Decode(lbPolicy)
	if err != nil {
		plog.Errorf("failed decoding load-balancing policy: %v", err)
		return nil, err
	}
	return lbPolicy, nil
}

func (pH *PolicyHandler) AddLBPolicy(policy *api.Policy) error {
	lbPolicy, err := lbPolicyFromBlob(policy.Spec.Blob)
	if err != nil {
		return err
	}
	return pH.loadBalancer.SetPolicy(lbPolicy)
}

func (pH *PolicyHandler) DeleteLBPolicy(policy *api.Policy) error {
	lbPolicy, err := lbPolicyFromBlob(policy.Spec.Blob)
	if err != nil {
		return err
	}
	return pH.loadBalancer.DeletePolicy(lbPolicy)
}

func (pH *PolicyHandler) AddAccessPolicy(policy *api.Policy) error {
	connPolicy, err := connPolicyFromBlob(policy.Spec.Blob)
	if err != nil {
		return err
	}
	return pH.connectivityPDP.AddOrUpdatePolicy(connPolicy)
}

func (pH *PolicyHandler) DeleteAccessPolicy(policy *api.Policy) error {
	connPolicy, err := connPolicyFromBlob(policy.Spec.Blob)
	if err != nil {
		return err
	}
	return pH.connectivityPDP.DeletePolicy(connPolicy.Name, connPolicy.Spec.Privileged)
}
