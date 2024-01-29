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

package policyengine_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/clusterlink-net/clusterlink/pkg/api"
	"github.com/clusterlink-net/clusterlink/pkg/policyengine"
	"github.com/clusterlink-net/clusterlink/pkg/policyengine/policytypes"
)

const (
	svcName    = "svc"
	badSvcName = "sv"
)

var (
	selectAllSelector = metav1.LabelSelector{}
	simpleSelector    = metav1.LabelSelector{
		MatchLabels: policytypes.WorkloadAttrs{policyengine.ServiceNameLabel: svcName},
	}
	simpleWorkloadSet = policytypes.WorkloadSetOrSelector{
		WorkloadSelector: &simpleSelector,
	}
	policy = policytypes.ConnectivityPolicy{
		Name:       "test-policy",
		Privileged: false,
		Action:     policytypes.ActionAllow,
		From:       []policytypes.WorkloadSetOrSelector{simpleWorkloadSet},
		To:         []policytypes.WorkloadSetOrSelector{simpleWorkloadSet},
	}
)

func TestAddAndDeleteConnectivityPolicy(t *testing.T) {
	ph := policyengine.NewPolicyHandler()
	policyBuf, err := json.Marshal(policy)
	require.Nil(t, err)
	apiPolicy := api.Policy{Name: "test", Spec: api.PolicySpec{Blob: policyBuf}}

	err = ph.AddAccessPolicy(&apiPolicy)
	require.Nil(t, err)

	err = ph.DeleteAccessPolicy(&apiPolicy)
	require.Nil(t, err)

	// deleting the same policy again should result in a not-found error
	err = ph.DeleteAccessPolicy(&apiPolicy)
	require.NotNil(t, err)
}

func TestAddBadPolicy(t *testing.T) {
	ph := policyengine.NewPolicyHandler()
	badPolicy := policytypes.ConnectivityPolicy{Name: "bad-policy"}
	policyBuf, err := json.Marshal(badPolicy)
	require.Nil(t, err)
	apiPolicy := api.Policy{Name: "bad-policy", Spec: api.PolicySpec{Blob: policyBuf}}

	err = ph.AddAccessPolicy(&apiPolicy)
	require.NotNil(t, err)

	notEvenAPolicy := []byte{'{'} // a malformed json
	apiPolicy = api.Policy{Name: "bad-json", Spec: api.PolicySpec{Blob: notEvenAPolicy}}

	err = ph.AddAccessPolicy(&apiPolicy)
	require.NotNil(t, err)
}

func TestDeleteMalformedPolicy(t *testing.T) {
	ph := policyengine.NewPolicyHandler()
	notEvenAPolicy := []byte{'{'}
	apiPolicy := api.Policy{Name: "bad-json", Spec: api.PolicySpec{Blob: notEvenAPolicy}}

	err := ph.DeleteAccessPolicy(&apiPolicy)
	require.NotNil(t, err)
}

func TestIncomingConnectionRequests(t *testing.T) {
	ph := policyengine.NewPolicyHandler()
	policy2 := policy
	policy2.To = []policytypes.WorkloadSetOrSelector{{WorkloadSelector: &selectAllSelector}}
	addPolicy(t, &policy2, ph)

	srcAttrs := policytypes.WorkloadAttrs{policyengine.ServiceNameLabel: svcName}
	connReq := policytypes.ConnectionRequest{SrcWorkloadAttrs: srcAttrs, Direction: policytypes.Incoming}
	connReqResp, err := ph.AuthorizeAndRouteConnection(&connReq)
	require.Equal(t, policytypes.ActionAllow, connReqResp.Action)
	require.Nil(t, err)

	srcAttrs[policyengine.ServiceNameLabel] = badSvcName
	connReq = policytypes.ConnectionRequest{SrcWorkloadAttrs: srcAttrs, Direction: policytypes.Incoming}
	connReqResp, err = ph.AuthorizeAndRouteConnection(&connReq)
	require.Equal(t, policytypes.ActionDeny, connReqResp.Action)
	require.Nil(t, err)
}

func TestOutgoingConnectionRequests(t *testing.T) {
	ph := policyengine.NewPolicyHandler()
	simpleSelector2 := metav1.LabelSelector{MatchLabels: policytypes.WorkloadAttrs{
		policyengine.ServiceNameLabel: svcName,
		policyengine.GatewayNameLabel: peer2,
	}}
	simpleWorkloadSet2 := policytypes.WorkloadSetOrSelector{WorkloadSelector: &simpleSelector2}
	policy2 := policy
	policy2.To = []policytypes.WorkloadSetOrSelector{simpleWorkloadSet2}
	addPolicy(t, &policy2, ph)
	addRemoteSvc(t, svcName, peer1, ph)
	addRemoteSvc(t, svcName, peer2, ph)

	// Should choose between peer1 and peer2, but only peer2 is allowed by the single access policy
	srcAttrs := policytypes.WorkloadAttrs{policyengine.ServiceNameLabel: svcName}
	badSrcAttrs := policytypes.WorkloadAttrs{policyengine.ServiceNameLabel: badSvcName}
	requestAttr := policytypes.ConnectionRequest{SrcWorkloadAttrs: srcAttrs, DstSvcName: svcName, Direction: policytypes.Outgoing}
	connReqResp, err := ph.AuthorizeAndRouteConnection(&requestAttr)
	require.Equal(t, policytypes.ActionAllow, connReqResp.Action)
	require.Equal(t, peer2, connReqResp.DstPeer)
	require.Nil(t, err)

	// Src service does not match the spec of the single access policy
	requestAttr = policytypes.ConnectionRequest{SrcWorkloadAttrs: badSrcAttrs, DstSvcName: svcName, Direction: policytypes.Outgoing}
	connReqResp, err = ph.AuthorizeAndRouteConnection(&requestAttr)
	require.Equal(t, policytypes.ActionDeny, connReqResp.Action)
	require.Nil(t, err)

	// Dst service does not match the spec of the single access policy
	requestAttr = policytypes.ConnectionRequest{SrcWorkloadAttrs: srcAttrs, DstSvcName: badSvcName, Direction: policytypes.Outgoing}
	connReqResp, err = ph.AuthorizeAndRouteConnection(&requestAttr)
	require.Equal(t, policytypes.ActionDeny, connReqResp.Action)
	require.Nil(t, err)

	// peer2 is removed as a remote for the requested service,
	// so now the single allow policy does not allow the remaining peers
	removeRemoteSvc(svcName, peer2, ph)
	requestAttr = policytypes.ConnectionRequest{SrcWorkloadAttrs: srcAttrs, DstSvcName: svcName, Direction: policytypes.Outgoing}
	connReqResp, err = ph.AuthorizeAndRouteConnection(&requestAttr)
	require.Equal(t, policytypes.ActionDeny, connReqResp.Action)
	require.Nil(t, err)
}

func TestLoadBalancer(t *testing.T) {
	ph := policyengine.NewPolicyHandler()
	addRemoteSvc(t, svcName, peer1, ph)
	addRemoteSvc(t, svcName, peer2, ph)
	addPolicy(t, &policy, ph)

	lbPolicy := policyengine.LBPolicy{
		ServiceSrc:  svcName,
		ServiceDst:  svcName,
		Scheme:      policyengine.Static,
		DefaultPeer: peer1,
	}
	policyBuf, err := json.Marshal(lbPolicy)
	require.Nil(t, err)
	apiLBPolicy := api.Policy{Name: policy.Name, Spec: api.PolicySpec{Blob: policyBuf}}
	err = ph.AddLBPolicy(&apiLBPolicy)
	require.Nil(t, err)

	srcAttrs := policytypes.WorkloadAttrs{policyengine.ServiceNameLabel: svcName}
	requestAttr := policytypes.ConnectionRequest{SrcWorkloadAttrs: srcAttrs, DstSvcName: svcName, Direction: policytypes.Outgoing}
	connReqResp, err := ph.AuthorizeAndRouteConnection(&requestAttr)
	require.Nil(t, err)
	require.Equal(t, policytypes.ActionAllow, connReqResp.Action)
	require.Equal(t, peer1, connReqResp.DstPeer) // LB policy requires this request to be served by peer1

	err = ph.DeleteLBPolicy(&apiLBPolicy) // LB policy is deleted - the random default policy now takes effect
	require.Nil(t, err)
	connReqResp, err = ph.AuthorizeAndRouteConnection(&requestAttr)
	require.Nil(t, err)
	require.Equal(t, policytypes.ActionAllow, connReqResp.Action)
	require.Contains(t, []string{peer1, peer2}, connReqResp.DstPeer)

	ph.DeletePeer(peer1) // peer1 is deleted, so all requests should go to peer2
	connReqResp, err = ph.AuthorizeAndRouteConnection(&requestAttr)
	require.Nil(t, err)
	require.Equal(t, policytypes.ActionAllow, connReqResp.Action)
	require.Equal(t, peer2, connReqResp.DstPeer)

	ph.DeletePeer(peer1) // deleting peer1 again should make no change
	connReqResp, err = ph.AuthorizeAndRouteConnection(&requestAttr)
	require.Nil(t, err)
	require.Equal(t, policytypes.ActionAllow, connReqResp.Action)
	require.Equal(t, peer2, connReqResp.DstPeer)

	ph.DeletePeer(peer2) // deleting peer2 should result in an deny, as there are no available peers left
	connReqResp, err = ph.AuthorizeAndRouteConnection(&requestAttr)
	require.Nil(t, err)
	require.Equal(t, policytypes.ActionDeny, connReqResp.Action)
}

func TestBadLBPolicy(t *testing.T) {
	ph := policyengine.NewPolicyHandler()
	notEvenAPolicy := []byte{'{'}
	apiPolicy := api.Policy{Name: "bad-json", Spec: api.PolicySpec{Blob: notEvenAPolicy}}

	err := ph.AddLBPolicy(&apiPolicy)
	require.NotNil(t, err)

	err = ph.DeleteLBPolicy(&apiPolicy)
	require.NotNil(t, err)
}

func TestDisableEnablePeers(t *testing.T) {
	ph := policyengine.NewPolicyHandler()
	addRemoteSvc(t, svcName, peer1, ph)
	addRemoteSvc(t, svcName, peer2, ph)
	addPolicy(t, &policy, ph)

	lbPolicy := policyengine.LBPolicy{
		ServiceSrc:  svcName,
		ServiceDst:  svcName,
		Scheme:      policyengine.Static,
		DefaultPeer: peer1,
	}
	policyBuf, err := json.Marshal(lbPolicy)
	require.Nil(t, err)
	apiLBPolicy := api.Policy{Name: policy.Name, Spec: api.PolicySpec{Blob: policyBuf}}
	err = ph.AddLBPolicy(&apiLBPolicy)
	require.Nil(t, err)

	srcAttrs := policytypes.WorkloadAttrs{policyengine.ServiceNameLabel: svcName}
	requestAttr := policytypes.ConnectionRequest{SrcWorkloadAttrs: srcAttrs, DstSvcName: svcName, Direction: policytypes.Outgoing}
	connReqResp, err := ph.AuthorizeAndRouteConnection(&requestAttr)
	require.Nil(t, err)
	require.Equal(t, policytypes.ActionAllow, connReqResp.Action)
	require.Equal(t, peer1, connReqResp.DstPeer) // LB policy defaults this request to be served by peer1

	ph.DeletePeer(peer1)

	connReqResp, err = ph.AuthorizeAndRouteConnection(&requestAttr)
	require.Nil(t, err)
	require.Equal(t, policytypes.ActionAllow, connReqResp.Action)
	require.Equal(t, peer2, connReqResp.DstPeer) // peer1 is now disabled, so peer2 must be used

	ph.DeletePeer(peer2)

	connReqResp, err = ph.AuthorizeAndRouteConnection(&requestAttr)
	require.Nil(t, err)
	require.Equal(t, policytypes.ActionDeny, connReqResp.Action) // no enabled peers - a Deny is returned
	require.Equal(t, "", connReqResp.DstPeer)

	ph.AddPeer(peer1)
	ph.AddPeer(peer2)

	connReqResp, err = ph.AuthorizeAndRouteConnection(&requestAttr)
	require.Nil(t, err)
	require.Equal(t, policytypes.ActionAllow, connReqResp.Action)
	require.Equal(t, peer1, connReqResp.DstPeer) // peer1 was re-enabled, so it is now chosen again
}

//nolint:unparam // `svc` always receives `svcName` (allow passing other names in future)
func addRemoteSvc(t *testing.T, svc, peer string, ph policyengine.PolicyDecider) {
	t.Helper()
	ph.AddPeer(peer) // just in case it was not already added
	action, err := ph.AddBinding(&api.Binding{Spec: api.BindingSpec{Import: svc, Peer: peer}})
	require.Nil(t, err)
	require.Equal(t, policytypes.ActionAllow, action)
}

func removeRemoteSvc(svc, peer string, ph policyengine.PolicyDecider) {
	ph.DeleteBinding(&api.Binding{Spec: api.BindingSpec{Import: svc, Peer: peer}})
}

func addPolicy(t *testing.T, policy *policytypes.ConnectivityPolicy, ph policyengine.PolicyDecider) {
	t.Helper()
	policyBuf, err := json.Marshal(policy)
	require.Nil(t, err)
	apiPolicy := api.Policy{Name: policy.Name, Spec: api.PolicySpec{Blob: policyBuf}}
	err = ph.AddAccessPolicy(&apiPolicy)
	require.Nil(t, err)
}
