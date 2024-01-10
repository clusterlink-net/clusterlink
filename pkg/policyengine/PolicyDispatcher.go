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
	event "github.com/clusterlink-net/clusterlink/pkg/controlplane/eventmanager"
	"github.com/clusterlink-net/clusterlink/pkg/policyengine/connectivitypdp"
	"github.com/clusterlink-net/clusterlink/pkg/policyengine/policytypes"
)

const (
	LbType     = "lb"     // Type for load-balancing policies
	AccessType = "access" // Type for access policies

	PolicyRoute = "/policy"        // Parent route for all kinds of policies
	LbRoute     = "/" + LbType     // Route for managing LoadBalancer policies
	AccessRoute = "/" + AccessType // Route for managing Access policies (Connectivity policies)

	GetRoute = "/"       // Route for getting a list of active policies
	AddRoute = "/add"    // Route for adding policies
	DelRoute = "/delete" // Route for deleting policies

	ServiceNameLabel = "clusterlink/metadata.serviceName"
	GatewayNameLabel = "clusterlink/metadata.gatewayName"
)

var plog = logrus.WithField("component", "PolicyEngine")

type PolicyDecider interface {
	AddLBPolicy(policy *api.Policy) error
	DeleteLBPolicy(policy *api.Policy) error

	AddAccessPolicy(policy *api.Policy) error
	DeleteAccessPolicy(policy *api.Policy) error

	AuthorizeAndRouteConnection(connReq *event.ConnectionRequestAttr) (event.ConnectionRequestResp, error)

	AddPeer(name string)
	DeletePeer(name string)

	AddBinding(imp *api.Binding) (event.Action, error)
	DeleteBinding(imp *api.Binding)

	AddExport(exp *api.Export) (event.ExposeRequestResp, error)
	DeleteExport(name string)
}

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

func (pH *PolicyHandler) decideIncomingConnection(
	requestAttr *event.ConnectionRequestAttr,
) (event.ConnectionRequestResp, error) {
	src := getServiceAttrs(requestAttr.SrcService, requestAttr.OtherPeer)
	dest := getServiceAttrs(requestAttr.DstService, "")
	decisions, err := pH.connectivityPDP.Decide(src, []policytypes.WorkloadAttrs{dest})
	if err != nil {
		plog.Errorf("error deciding on a connection: %v", err)
		return event.ConnectionRequestResp{Action: event.Deny}, err
	}
	if decisions[0].Decision == policytypes.PolicyDecisionAllow {
		return event.ConnectionRequestResp{Action: event.Allow}, nil
	}
	return event.ConnectionRequestResp{Action: event.Deny}, nil
}

func (pH *PolicyHandler) decideOutgoingConnection(
	requestAttr *event.ConnectionRequestAttr,
) (event.ConnectionRequestResp, error) {
	// Get a list of peers for the service
	peerList, err := pH.loadBalancer.GetTargetPeers(requestAttr.DstService)
	if err != nil || len(peerList) == 0 {
		plog.Errorf("error getting target peers for service %s: %v", requestAttr.DstService, err)
		// this can be caused by a user typo - so only log this error
		return event.ConnectionRequestResp{Action: event.Deny}, nil
	}

	peerList = pH.filterOutDisabledPeers(peerList)

	src := getServiceAttrs(requestAttr.SrcService, "")
	dsts := getServiceAttrsForMultiplePeers(requestAttr.DstService, peerList)
	decisions, err := pH.connectivityPDP.Decide(src, dsts)
	if err != nil {
		plog.Errorf("error deciding on a connection: %v", err)
		return event.ConnectionRequestResp{Action: event.Deny}, err
	}

	allowedPeers := []string{}
	for _, decision := range decisions {
		dstPeer := decision.Destination[GatewayNameLabel]
		if decision.Decision == policytypes.PolicyDecisionAllow {
			allowedPeers = append(allowedPeers, dstPeer)
		}
	}

	if len(allowedPeers) == 0 {
		plog.Infof("access policies deny connections to service %s in all peers", requestAttr.DstService)
		return event.ConnectionRequestResp{Action: event.Deny}, nil
	}

	// Perform load-balancing using the filtered peer list
	targetPeer, err := pH.loadBalancer.LookupWith(requestAttr.SrcService, requestAttr.DstService, allowedPeers)
	if err != nil {
		return event.ConnectionRequestResp{Action: event.Deny}, err
	}
	return event.ConnectionRequestResp{Action: event.Allow, TargetPeer: targetPeer}, nil
}

func (pH *PolicyHandler) AuthorizeAndRouteConnection(
	connReq *event.ConnectionRequestAttr,
) (event.ConnectionRequestResp, error) {
	plog.Infof("New connection request : %+v", connReq)

	var resp event.ConnectionRequestResp
	var err error
	if connReq.Direction == event.Incoming {
		resp, err = pH.decideIncomingConnection(connReq)
	} else if connReq.Direction == event.Outgoing {
		resp, err = pH.decideOutgoingConnection(connReq)
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

func (pH *PolicyHandler) AddBinding(binding *api.Binding) (event.Action, error) {
	pH.loadBalancer.AddToServiceMap(binding.Spec.Import, binding.Spec.Peer)
	return event.Allow, nil
}

func (pH *PolicyHandler) DeleteBinding(binding *api.Binding) {
	pH.loadBalancer.RemoveDestService(binding.Spec.Import, binding.Spec.Peer)
}

func (pH *PolicyHandler) AddExport(_ *api.Export) (event.ExposeRequestResp, error) {
	return event.ExposeRequestResp{Action: event.AllowAll}, nil
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
	return pH.connectivityPDP.AddOrUpdatePolicy(*connPolicy)
}

func (pH *PolicyHandler) DeleteAccessPolicy(policy *api.Policy) error {
	connPolicy, err := connPolicyFromBlob(policy.Spec.Blob)
	if err != nil {
		return err
	}
	return pH.connectivityPDP.DeletePolicy(connPolicy.Name, connPolicy.Privileged)
}
