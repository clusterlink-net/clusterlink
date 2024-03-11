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

	"github.com/clusterlink-net/clusterlink/pkg/api"
	"github.com/clusterlink-net/clusterlink/pkg/policyengine/connectivitypdp"
	"github.com/clusterlink-net/clusterlink/pkg/policyengine/policytypes"
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

	AuthorizeAndRouteConnection(connReq *policytypes.ConnectionRequest) (policytypes.ConnectionResponse, error)

	AddPeer(name string)
	DeletePeer(name string)

	AddImport(imp *api.Import) policytypes.PolicyAction
	DeleteImport(imp *api.Import)

	AddExport(exp *api.Export) ([]string, error) // Returns a list of peers to which export is allowed
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

func getServiceAttrs(serviceName, peer string) policytypes.WorkloadAttrs {
	ret := policytypes.WorkloadAttrs{ServiceNameLabel: serviceName}
	if len(peer) > 0 {
		ret[GatewayNameLabel] = peer
	}
	return ret
}

func getServiceAttrsForMultiplePeers(serviceName string, peers []string) []policytypes.WorkloadAttrs {
	res := []policytypes.WorkloadAttrs{}
	for _, peer := range peers {
		res = append(res, getServiceAttrs(serviceName, peer))
	}
	return res
}

func (pH *PolicyHandler) filterOutDisabledPeers(peers []string) []string {
	res := []string{}
	for _, peer := range peers {
		if pH.enabledPeers[peer] {
			res = append(res, peer)
		}
	}
	return res
}

func (pH *PolicyHandler) decideIncomingConnection(req *policytypes.ConnectionRequest) (policytypes.ConnectionResponse, error) {
	dest := getServiceAttrs(req.DstSvcName, "")
	decisions, err := pH.connectivityPDP.Decide(req.SrcWorkloadAttrs, []policytypes.WorkloadAttrs{dest})
	if err != nil {
		plog.Errorf("error deciding on a connection: %v", err)
		return policytypes.ConnectionResponse{Action: policytypes.ActionDeny}, err
	}
	if decisions[0].Decision == policytypes.DecisionAllow {
		return policytypes.ConnectionResponse{Action: policytypes.ActionAllow}, nil
	}
	return policytypes.ConnectionResponse{Action: policytypes.ActionDeny}, nil
}

func (pH *PolicyHandler) decideOutgoingConnection(req *policytypes.ConnectionRequest) (policytypes.ConnectionResponse, error) {
	// Get a list of peers for the service
	peerList, err := pH.loadBalancer.GetTargetPeers(req.DstSvcName)
	if err != nil || len(peerList) == 0 {
		plog.Errorf("error getting target peers for service %s: %v", req.DstSvcName, err)
		// this can be caused by a user typo - so only log this error
		return policytypes.ConnectionResponse{Action: policytypes.ActionDeny}, nil
	}

	peerList = pH.filterOutDisabledPeers(peerList)

	dsts := getServiceAttrsForMultiplePeers(req.DstSvcName, peerList)
	decisions, err := pH.connectivityPDP.Decide(req.SrcWorkloadAttrs, dsts)
	if err != nil {
		plog.Errorf("error deciding on a connection: %v", err)
		return policytypes.ConnectionResponse{Action: policytypes.ActionDeny}, err
	}

	allowedPeers := []string{}
	for _, decision := range decisions {
		dstPeer := decision.Destination[GatewayNameLabel]
		if decision.Decision == policytypes.DecisionAllow {
			allowedPeers = append(allowedPeers, dstPeer)
		}
	}

	if len(allowedPeers) == 0 {
		plog.Infof("access policies deny connections to service %s in all peers", req.DstSvcName)
		return policytypes.ConnectionResponse{Action: policytypes.ActionDeny}, nil
	}

	// Perform load-balancing using the filtered peer list
	srcSvcName := req.SrcWorkloadAttrs[ServiceNameLabel]
	targetPeer, err := pH.loadBalancer.LookupWith(srcSvcName, req.DstSvcName, allowedPeers)
	if err != nil {
		return policytypes.ConnectionResponse{Action: policytypes.ActionDeny}, err
	}
	return policytypes.ConnectionResponse{Action: policytypes.ActionAllow, DstPeer: targetPeer}, nil
}

func (pH *PolicyHandler) AuthorizeAndRouteConnection(req *policytypes.ConnectionRequest) (
	policytypes.ConnectionResponse,
	error,
) {
	plog.Infof("New connection request : %+v", req)

	var resp policytypes.ConnectionResponse
	var err error
	if req.Direction == policytypes.Incoming {
		resp, err = pH.decideIncomingConnection(req)
	} else if req.Direction == policytypes.Outgoing {
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

func (pH *PolicyHandler) AddImport(imp *api.Import) policytypes.PolicyAction {
	for _, pr := range imp.Spec.Peers {
		pH.loadBalancer.AddToServiceMap(imp.Name, pr)
	}
	return policytypes.ActionAllow
}

func (pH *PolicyHandler) DeleteImport(imp *api.Import) {
	for _, pr := range imp.Spec.Peers {
		pH.loadBalancer.RemoveDestService(imp.Name, pr)
	}
}

func (pH *PolicyHandler) AddExport(_ *api.Export) ([]string, error) {
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
func connPolicyFromBlob(blob []byte) (*policytypes.ConnectivityPolicy, error) {
	bReader := bytes.NewReader(blob)
	connPolicy := &policytypes.ConnectivityPolicy{}
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
	return pH.connectivityPDP.DeletePolicy(connPolicy.Name, connPolicy.Privileged)
}
