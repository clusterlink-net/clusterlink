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
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	crds "github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/policyengine"
	"github.com/clusterlink-net/clusterlink/pkg/policyengine/connectivitypdp"
)

const (
	svcName    = "svc"
	badSvcName = "sv"
	defaultNS  = "default"
)

var (
	selectAllSelector = metav1.LabelSelector{}
	simpleSelector    = metav1.LabelSelector{
		MatchLabels: connectivitypdp.WorkloadAttrs{policyengine.ServiceNameLabel: svcName},
	}
	simpleWorkloadSet = crds.WorkloadSetOrSelector{
		WorkloadSelector: &simpleSelector,
	}
	policy = crds.AccessPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-policy",
			Namespace: defaultNS,
		},
		Spec: crds.AccessPolicySpec{
			Action: crds.AccessPolicyActionAllow,
			From:   []crds.WorkloadSetOrSelector{simpleWorkloadSet},
			To:     []crds.WorkloadSetOrSelector{simpleWorkloadSet},
		},
	}

	pdpPolicy = connectivitypdp.PolicyFromCR(&policy)
)

func TestAddAndDeleteConnectivityPolicy(t *testing.T) {
	ph := policyengine.NewPolicyHandler()
	err := ph.AddAccessPolicy(pdpPolicy)
	require.Nil(t, err)

	polName := types.NamespacedName{Namespace: policy.Namespace, Name: policy.Name}
	err = ph.DeleteAccessPolicy(polName, false)
	require.Nil(t, err)

	// deleting the same policy again should result in a not-found error
	err = ph.DeleteAccessPolicy(polName, false)
	require.NotNil(t, err)
}

func TestAddBadPolicy(t *testing.T) {
	ph := policyengine.NewPolicyHandler()
	badPolicy := crds.AccessPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "bad-policy",
		},
	}
	err := ph.AddAccessPolicy(connectivitypdp.PolicyFromCR(&badPolicy))
	require.NotNil(t, err)
}

func TestIncomingConnectionRequests(t *testing.T) {
	ph := policyengine.NewPolicyHandler()
	policy2 := policy
	policy2.Spec.To = []crds.WorkloadSetOrSelector{{WorkloadSelector: &selectAllSelector}}
	err := ph.AddAccessPolicy(connectivitypdp.PolicyFromCR(&policy2))
	require.Nil(t, err)

	srcAttrs := connectivitypdp.WorkloadAttrs{policyengine.ServiceNameLabel: svcName}
	connReq := connectivitypdp.ConnectionRequest{
		SrcWorkloadAttrs: srcAttrs,
		Direction:        connectivitypdp.Incoming,
		DstSvcNamespace:  defaultNS,
	}
	connReqResp, err := ph.AuthorizeAndRouteConnection(&connReq)
	require.Equal(t, crds.AccessPolicyActionAllow, connReqResp.Action)
	require.Nil(t, err)

	srcAttrs[policyengine.ServiceNameLabel] = badSvcName
	connReq = connectivitypdp.ConnectionRequest{SrcWorkloadAttrs: srcAttrs, Direction: connectivitypdp.Incoming}
	connReqResp, err = ph.AuthorizeAndRouteConnection(&connReq)
	require.Equal(t, crds.AccessPolicyActionDeny, connReqResp.Action)
	require.Nil(t, err)
}

func TestOutgoingConnectionRequests(t *testing.T) {
	ph := policyengine.NewPolicyHandler()
	simpleSelector2 := metav1.LabelSelector{MatchLabels: connectivitypdp.WorkloadAttrs{
		policyengine.ServiceNameLabel: svcName,
		policyengine.GatewayNameLabel: peer2,
	}}
	simpleWorkloadSet2 := crds.WorkloadSetOrSelector{WorkloadSelector: &simpleSelector2}
	policy2 := policy
	policy2.Spec.To = []crds.WorkloadSetOrSelector{simpleWorkloadSet2}
	err := ph.AddAccessPolicy(connectivitypdp.PolicyFromCR(&policy2))
	require.Nil(t, err)

	addRemoteSvc(t, svcName, []string{peer1, peer2}, "", ph)

	// Should choose between peer1 and peer2, but only peer2 is allowed by the single access policy
	srcAttrs := connectivitypdp.WorkloadAttrs{policyengine.ServiceNameLabel: svcName}
	badSrcAttrs := connectivitypdp.WorkloadAttrs{policyengine.ServiceNameLabel: badSvcName}
	requestAttr := connectivitypdp.ConnectionRequest{
		SrcWorkloadAttrs: srcAttrs,
		DstSvcName:       svcName,
		DstSvcNamespace:  defaultNS,
		Direction:        connectivitypdp.Outgoing,
	}
	connReqResp, err := ph.AuthorizeAndRouteConnection(&requestAttr)
	require.Equal(t, crds.AccessPolicyActionAllow, connReqResp.Action)
	require.Equal(t, peer2, connReqResp.DstPeer)
	require.Nil(t, err)

	// Src service does not match the spec of the single access policy
	requestAttr = connectivitypdp.ConnectionRequest{
		SrcWorkloadAttrs: badSrcAttrs,
		DstSvcName:       svcName,
		Direction:        connectivitypdp.Outgoing,
	}
	connReqResp, err = ph.AuthorizeAndRouteConnection(&requestAttr)
	require.Equal(t, crds.AccessPolicyActionDeny, connReqResp.Action)
	require.Nil(t, err)

	// Dst service does not match the spec of the single access policy
	requestAttr = connectivitypdp.ConnectionRequest{
		SrcWorkloadAttrs: srcAttrs,
		DstSvcName:       badSvcName,
		Direction:        connectivitypdp.Outgoing,
	}
	connReqResp, err = ph.AuthorizeAndRouteConnection(&requestAttr)
	require.Equal(t, crds.AccessPolicyActionDeny, connReqResp.Action)
	require.Nil(t, err)

	// peer2 is removed as a remote for the requested service,
	// so now the single allow policy does not allow the remaining peers
	ph.DeletePeer(peer2)
	requestAttr = connectivitypdp.ConnectionRequest{
		SrcWorkloadAttrs: srcAttrs,
		DstSvcName:       svcName,
		Direction:        connectivitypdp.Outgoing,
	}
	connReqResp, err = ph.AuthorizeAndRouteConnection(&requestAttr)
	require.Equal(t, crds.AccessPolicyActionDeny, connReqResp.Action)
	require.Nil(t, err)
}

func TestLoadBalancer(t *testing.T) {
	ph := policyengine.NewPolicyHandler()
	addRemoteSvc(t, svcName, []string{peer1, peer2}, "", ph)
	require.Nil(t, ph.AddAccessPolicy(pdpPolicy))

	addRemoteSvc(t, svcName, []string{peer1, peer2}, policyengine.Static, ph)

	srcAttrs := connectivitypdp.WorkloadAttrs{policyengine.ServiceNameLabel: svcName}
	requestAttr := connectivitypdp.ConnectionRequest{
		SrcWorkloadAttrs: srcAttrs,
		DstSvcName:       svcName,
		DstSvcNamespace:  defaultNS,
		Direction:        connectivitypdp.Outgoing,
	}
	connReqResp, err := ph.AuthorizeAndRouteConnection(&requestAttr)
	require.Nil(t, err)
	require.Equal(t, crds.AccessPolicyActionAllow, connReqResp.Action)
	require.Equal(t, peer1, connReqResp.DstPeer) // LB policy requires this request to be served by peer1

	addRemoteSvc(t, svcName, []string{peer1, peer2}, "", ph)
	// LB policy is deleted - the random default policy now takes effect
	connReqResp, err = ph.AuthorizeAndRouteConnection(&requestAttr)
	require.Nil(t, err)
	require.Equal(t, crds.AccessPolicyActionAllow, connReqResp.Action)
	require.Contains(t, []string{peer1, peer2}, connReqResp.DstPeer)

	ph.DeletePeer(peer1) // peer1 is deleted, so all requests should go to peer2
	connReqResp, err = ph.AuthorizeAndRouteConnection(&requestAttr)
	require.Nil(t, err)
	require.Equal(t, crds.AccessPolicyActionAllow, connReqResp.Action)
	require.Equal(t, peer2, connReqResp.DstPeer)

	ph.DeletePeer(peer1) // deleting peer1 again should make no change
	connReqResp, err = ph.AuthorizeAndRouteConnection(&requestAttr)
	require.Nil(t, err)
	require.Equal(t, crds.AccessPolicyActionAllow, connReqResp.Action)
	require.Equal(t, peer2, connReqResp.DstPeer)

	ph.DeletePeer(peer2) // deleting peer2 should result in an deny, as there are no available peers left
	connReqResp, err = ph.AuthorizeAndRouteConnection(&requestAttr)
	require.Nil(t, err)
	require.Equal(t, crds.AccessPolicyActionDeny, connReqResp.Action)
}

func TestDisableEnablePeers(t *testing.T) {
	ph := policyengine.NewPolicyHandler()
	addRemoteSvc(t, svcName, []string{peer1, peer2}, "", ph)
	require.Nil(t, ph.AddAccessPolicy(pdpPolicy))

	addRemoteSvc(t, svcName, []string{peer1, peer2}, policyengine.Static, ph)

	srcAttrs := connectivitypdp.WorkloadAttrs{policyengine.ServiceNameLabel: svcName}
	requestAttr := connectivitypdp.ConnectionRequest{
		SrcWorkloadAttrs: srcAttrs,
		DstSvcName:       svcName,
		DstSvcNamespace:  defaultNS,
		Direction:        connectivitypdp.Outgoing,
	}
	connReqResp, err := ph.AuthorizeAndRouteConnection(&requestAttr)
	require.Nil(t, err)
	require.Equal(t, crds.AccessPolicyActionAllow, connReqResp.Action)
	require.Equal(t, peer1, connReqResp.DstPeer) // LB policy defaults this request to be served by peer1

	ph.DeletePeer(peer1)

	connReqResp, err = ph.AuthorizeAndRouteConnection(&requestAttr)
	require.Nil(t, err)
	require.Equal(t, crds.AccessPolicyActionAllow, connReqResp.Action)
	require.Equal(t, peer2, connReqResp.DstPeer) // peer1 is now disabled, so peer2 must be used

	ph.DeletePeer(peer2)

	connReqResp, err = ph.AuthorizeAndRouteConnection(&requestAttr)
	require.Nil(t, err)
	require.Equal(t, crds.AccessPolicyActionDeny, connReqResp.Action) // no enabled peers - a Deny is returned
	require.Equal(t, "", connReqResp.DstPeer)

	ph.AddPeer(peer1)
	ph.AddPeer(peer2)

	connReqResp, err = ph.AuthorizeAndRouteConnection(&requestAttr)
	require.Nil(t, err)
	require.Equal(t, crds.AccessPolicyActionAllow, connReqResp.Action)
	require.Equal(t, peer1, connReqResp.DstPeer) // peer1 was re-enabled, so it is now chosen again
}

//nolint:unparam // `svc` always receives `svcName` (allow passing other names in future)
func addRemoteSvc(
	t *testing.T,
	svc string,
	peers []string,
	lbScheme policyengine.LBScheme,
	ph policyengine.PolicyDecider,
) {
	t.Helper()

	srcs := []crds.ImportSource{}
	for _, peer := range peers {
		ph.AddPeer(peer)
		srcs = append(srcs, crds.ImportSource{Peer: peer, ExportName: svc})
	}

	imp := crds.Import{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: defaultNS,
		},
		Spec: crds.ImportSpec{
			Sources:  srcs,
			LBScheme: string(lbScheme),
		},
	}
	ph.AddImport(&imp)
}
