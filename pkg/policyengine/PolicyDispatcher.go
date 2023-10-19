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

/**********************************************************/
/* Package Policy contain all Policies and data structure
/* related to Policy that can run in mbg
/**********************************************************/
package policyengine

import (
	"bytes"
	"encoding/json"
	"io"

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
	MbgNameLabel     = "clusterlink/metadata.gatewayName"
)

var plog = logrus.WithField("component", "PolicyEngine")

type PolicyDecider interface {
	AddLBPolicy(lbPolicy *LBPolicy) error
	DeleteLBPolicy(lbPolicy *LBPolicy) error

	AddAccessPolicy(policy *api.Policy) error
	DeleteAccessPolicy(policy *api.Policy) error

	AuthorizeAndRouteConnection(connReq *event.ConnectionRequestAttr) (event.ConnectionRequestResp, error)

	AddPeer(peer *api.Peer)
	DeletePeer(name string)

	AddBinding(imp *api.Binding) (event.Action, error)
	DeleteBinding(imp *api.Binding)

	AddExport(exp *api.Export) (event.ExposeRequestResp, error)
	DeleteExport(name string)
}

type MbgState struct {
	mbgPeers []string
}

type PolicyHandler struct {
	loadBalancer    *LoadBalancer
	connectivityPDP *connectivitypdp.PDP
	mbgState        MbgState
}

func NewPolicyHandler() PolicyDecider {
	return &PolicyHandler{
		loadBalancer:    NewLoadBalancer(),
		connectivityPDP: connectivitypdp.NewPDP(),
	}
}

func exists(slice []string, entry string) (int, bool) {
	for i, e := range slice {
		if e == entry {
			return i, true
		}
	}
	return -1, false
}

func getServiceAttrs(serviceName, peer string) policytypes.WorkloadAttrs {
	ret := policytypes.WorkloadAttrs{ServiceNameLabel: serviceName}
	if len(peer) > 0 {
		ret[MbgNameLabel] = peer
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

func (pH *PolicyHandler) decideIncomingConnection(requestAttr *event.ConnectionRequestAttr) (event.ConnectionRequestResp, error) {
	src := getServiceAttrs(requestAttr.SrcService, requestAttr.OtherMbg)
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

func (pH *PolicyHandler) decideOutgoingConnection(requestAttr *event.ConnectionRequestAttr) (event.ConnectionRequestResp, error) {
	// Get a list of MBGs for the service
	mbgList, err := pH.loadBalancer.GetTargetMbgs(requestAttr.DstService)
	if err != nil || len(mbgList) == 0 {
		plog.Errorf("error getting target peers for service %s: %v", requestAttr.DstService, err)
		return event.ConnectionRequestResp{Action: event.Deny}, nil // this can be caused by a user typo - so only log this error
	}

	src := getServiceAttrs(requestAttr.SrcService, "")
	dsts := getServiceAttrsForMultiplePeers(requestAttr.DstService, mbgList)
	decisions, err := pH.connectivityPDP.Decide(src, dsts)
	if err != nil {
		plog.Errorf("error deciding on a connection: %v", err)
		return event.ConnectionRequestResp{Action: event.Deny}, err
	}

	allowedMbgs := []string{}
	for _, decision := range decisions {
		if decision.Decision == policytypes.PolicyDecisionAllow {
			allowedMbgs = append(allowedMbgs, decision.Destination[MbgNameLabel])
		}
	}

	if len(allowedMbgs) == 0 {
		plog.Infof("access policies deny connections to service %s in all peers", requestAttr.DstService)
		return event.ConnectionRequestResp{Action: event.Deny}, nil
	}

	// Perform load-balancing using the filtered mbgList
	targetMbg, err := pH.loadBalancer.LookupWith(requestAttr.SrcService, requestAttr.DstService, allowedMbgs)
	if err != nil {
		return event.ConnectionRequestResp{Action: event.Deny}, err
	}
	return event.ConnectionRequestResp{Action: event.Allow, TargetMbg: targetMbg}, nil
}

func (pH *PolicyHandler) AuthorizeAndRouteConnection(connReq *event.ConnectionRequestAttr) (event.ConnectionRequestResp, error) {
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

func (pH *PolicyHandler) AddPeer(peer *api.Peer) {
	_, exist := exists(pH.mbgState.mbgPeers, peer.Name)
	if exist {
		return
	}
	pH.mbgState.mbgPeers = append(pH.mbgState.mbgPeers, peer.Name)
	plog.Infof("Added Peer %+v", pH.mbgState.mbgPeers)
}

func (pH *PolicyHandler) DeletePeer(name string) {
	pH.loadBalancer.RemoveMbgFromServiceMap(name)

	index, exist := exists(pH.mbgState.mbgPeers, name)
	if !exist {
		return
	}
	pH.mbgState.mbgPeers = append(pH.mbgState.mbgPeers[:index], pH.mbgState.mbgPeers[index+1:]...)
	plog.Infof("Removed Peer(%s, %d) %+v", name, index, pH.mbgState.mbgPeers)

}

func (pH *PolicyHandler) AddBinding(binding *api.Binding) (event.Action, error) {
	pH.loadBalancer.AddToServiceMap(binding.Spec.Import, binding.Spec.Peer)
	return event.Allow, nil
}

func (pH *PolicyHandler) DeleteBinding(binding *api.Binding) {
	pH.loadBalancer.RemoveDestService(binding.Spec.Import, binding.Spec.Peer)
}

func (pH *PolicyHandler) AddExport(_ *api.Export) (event.ExposeRequestResp, error) {
	return event.ExposeRequestResp{Action: event.AllowAll, TargetMbgs: pH.mbgState.mbgPeers}, nil
}

func (pH *PolicyHandler) DeleteExport(_ string) {
}

func connPolicyFromBlob(blob io.Reader) (*policytypes.ConnectivityPolicy, error) {
	connPolicy := &policytypes.ConnectivityPolicy{}
	err := json.NewDecoder(blob).Decode(connPolicy)
	if err != nil {
		plog.Errorf("failed decoding connectivity policy: %v", err)
		return nil, err
	}
	return connPolicy, nil
}

func (pH *PolicyHandler) AddLBPolicy(lbPolicy *LBPolicy) error {
	pH.loadBalancer.SetPolicy(lbPolicy.ServiceSrc, lbPolicy.ServiceDst, lbPolicy.Scheme, lbPolicy.DefaultMbg)
	return nil
}

func (pH *PolicyHandler) DeleteLBPolicy(lbPolicy *LBPolicy) error {
	pH.loadBalancer.deletePolicy(lbPolicy.ServiceSrc, lbPolicy.ServiceDst, lbPolicy.Scheme, lbPolicy.DefaultMbg)
	return nil
}

func (pH *PolicyHandler) AddAccessPolicy(policy *api.Policy) error {
	connPolicy, err := connPolicyFromBlob(bytes.NewReader(policy.Spec.Blob))
	if err != nil {
		return err
	}
	return pH.connectivityPDP.AddOrUpdatePolicy(*connPolicy)
}

func (pH *PolicyHandler) DeleteAccessPolicy(policy *api.Policy) error {
	connPolicy, err := connPolicyFromBlob(bytes.NewReader(policy.Spec.Blob))
	if err != nil {
		return err
	}
	return pH.connectivityPDP.DeletePolicy(connPolicy.Name, connPolicy.Privileged)
}
