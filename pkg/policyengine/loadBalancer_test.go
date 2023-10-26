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

	event "github.com/clusterlink-net/clusterlink/pkg/controlplane/eventmanager"
	"github.com/clusterlink-net/clusterlink/pkg/policyengine"
)

const (
	svc1  = "svc1"
	svc2  = "svc2"
	svc3  = "svc3"
	peer1 = "peer1"
	peer2 = "peer2"
	peer3 = "peer3"
)

func addImports(lb *policyengine.LoadBalancer) {
	// svc1 is imported from peer1 and peer2
	// svc2 is imported only from peer1
	// svc3 is imported from peer1 and peer2
	lb.AddToServiceMap(svc1, peer1)
	lb.AddToServiceMap(svc1, peer2)
	lb.AddToServiceMap(svc2, peer1)
	lb.AddToServiceMap(svc3, peer1)
	lb.AddToServiceMap(svc3, peer2)
}

// We repeat lookups enough times to make sure we get all the peers allowed by the relevant policy
func repeatLookups(t *testing.T, lb *policyengine.LoadBalancer, srcSvc, dstSvc string, peers []string) map[string]bool {
	res := map[string]bool{}
	for i := 0; i < 100; i++ {
		targetPeer, err := lb.LookupWith(srcSvc, dstSvc, peers)
		require.Nil(t, err)
		res[targetPeer] = true
		if len(res) == len(peers) { // All legal peers appeared in lookup
			break
		}
	}
	return res
}

func TestAddAndRemoveImportsAndPeers(t *testing.T) {
	lb := policyengine.NewLoadBalancer()
	addImports(lb)

	svc1Peers, err := lb.GetTargetPeers(svc1)
	require.Nil(t, err)
	require.Equal(t, []string{peer1, peer2}, svc1Peers)

	svc2Peers, err := lb.GetTargetPeers(svc2)
	require.Nil(t, err)
	require.Equal(t, []string{peer1}, svc2Peers)

	svc3Peers, err := lb.GetTargetPeers(svc3)
	require.Nil(t, err)
	require.Equal(t, []string{peer1, peer2}, svc3Peers)

	// peer2 no longer exports svc1 - now only peer1 exports svc1
	lb.RemoveDestService(svc1, peer2)

	svc1Peers, err = lb.GetTargetPeers(svc1)
	require.Nil(t, err)
	require.Equal(t, []string{peer1}, svc1Peers)

	// test deleting the same import twice
	lb.RemoveDestService(svc1, peer2)

	svc1Peers, err = lb.GetTargetPeers(svc1)
	require.Nil(t, err)
	require.Equal(t, []string{peer1}, svc1Peers)

	// remove peer1 - no target peers for both svc1 and svc2
	lb.RemovePeerFromServiceMap(peer1)

	_, err = lb.GetTargetPeers(svc1)
	require.NotNil(t, err)

	_, err = lb.GetTargetPeers(svc2)
	require.NotNil(t, err)

	svc3Peers, err = lb.GetTargetPeers(svc3)
	require.Nil(t, err)
	require.Equal(t, []string{peer2}, svc3Peers)

	// remove all peers for svc3
	lb.RemoveDestService(svc3, "")

	_, err = lb.GetTargetPeers(svc3)
	require.NotNil(t, err)
}

func TestAddAndRemovePolicy(t *testing.T) {
	lb := policyengine.NewLoadBalancer()
	addImports(lb)
	svc1Peers, err := lb.GetTargetPeers(svc1)
	require.Nil(t, err)
	require.Equal(t, []string{peer1, peer2}, svc1Peers)

	lbPolicy := policyengine.LBPolicy{ServiceSrc: svc2, ServiceDst: svc1, Scheme: policyengine.Static, DefaultPeer: peer1}
	err = lb.SetPolicy(&lbPolicy) // static policy: svc2 asking to connect to svc1 should always choose peer1
	require.Nil(t, err)

	peers := repeatLookups(t, lb, svc2, svc1, svc1Peers)
	require.Len(t, peers, 1)
	require.Equal(t, true, peers[peer1])

	peers = repeatLookups(t, lb, svc3, svc1, svc1Peers) // using a different source service - should default to random
	require.Len(t, peers, 2)
	require.Equal(t, true, peers[peer1])
	require.Equal(t, true, peers[peer2])

	peers = repeatLookups(t, lb, svc2, svc3, svc1Peers) // using a different target service - should default to random
	require.Len(t, peers, 2)
	require.Equal(t, true, peers[peer1])
	require.Equal(t, true, peers[peer2])

	peers = repeatLookups(t, lb, svc2, svc1, []string{peer2}) // default peer is not available - fall back to random
	require.Len(t, peers, 1)
	require.Equal(t, true, peers[peer2])

	lbPolicy = policyengine.LBPolicy{ServiceSrc: svc2, ServiceDst: svc1, Scheme: policyengine.ECMP, DefaultPeer: peer1}
	err = lb.SetPolicy(&lbPolicy) // override above policy with a round-robin policy
	require.Nil(t, err)

	peers = repeatLookups(t, lb, svc2, svc1, svc1Peers)
	require.Len(t, peers, 2)
	require.Equal(t, true, peers[peer1])
	require.Equal(t, true, peers[peer2])

	err = lb.DeletePolicy(&lbPolicy) // delete policy - fall back to random policy
	require.Nil(t, err)

	peers = repeatLookups(t, lb, svc2, svc1, svc1Peers)
	require.Len(t, peers, 2)
	require.Equal(t, true, peers[peer1])
	require.Equal(t, true, peers[peer2])
}

func TestLookupWithNoPeers(t *testing.T) {
	lb := policyengine.NewLoadBalancer()
	addImports(lb)

	_, err := lb.LookupWith(svc1, svc2, []string{})
	require.NotNil(t, err)
}

func TestSetBadStaticPolicy(t *testing.T) {
	lb := policyengine.NewLoadBalancer()
	addImports(lb)

	badPolicy := policyengine.LBPolicy{ServiceSrc: svc2, ServiceDst: svc1, Scheme: policyengine.Static, DefaultPeer: peer3}
	err := lb.SetPolicy(&badPolicy)
	require.NotNil(t, err)
}

func TestDeletingNonExistingPolicy(t *testing.T) {
	lb := policyengine.NewLoadBalancer()
	addImports(lb)

	noSuchPolicy := policyengine.LBPolicy{ServiceSrc: svc2, ServiceDst: svc1, Scheme: policyengine.Static, DefaultPeer: peer3}
	err := lb.DeletePolicy(&noSuchPolicy)
	require.NotNil(t, err)
}

func TestPoliciesWithWildcards(t *testing.T) {
	lb := policyengine.NewLoadBalancer()
	addImports(lb)
	svc1Peers, err := lb.GetTargetPeers(svc1)
	require.Nil(t, err)
	svc2Peers, err := lb.GetTargetPeers(svc2)
	require.Nil(t, err)
	svc3Peers, err := lb.GetTargetPeers(svc3)
	require.Nil(t, err)

	policy := policyengine.LBPolicy{ServiceSrc: event.Wildcard, ServiceDst: svc1, Scheme: policyengine.Static, DefaultPeer: peer1}
	err = lb.SetPolicy(&policy)
	require.Nil(t, err)

	peers := repeatLookups(t, lb, svc2, svc1, svc1Peers)
	require.Len(t, peers, 1)
	require.Equal(t, true, peers[peer1])

	peers = repeatLookups(t, lb, svc3, svc1, svc1Peers)
	require.Len(t, peers, 1)
	require.Equal(t, true, peers[peer1])

	peers = repeatLookups(t, lb, event.Wildcard, svc1, svc1Peers)
	require.Len(t, peers, 1)
	require.Equal(t, true, peers[peer1])

	peers = repeatLookups(t, lb, svc2, event.Wildcard, svc1Peers)
	require.Len(t, peers, 2)
	require.Equal(t, true, peers[peer1])
	require.Equal(t, true, peers[peer2])

	err = lb.DeletePolicy(&policy)
	require.Nil(t, err)

	policy = policyengine.LBPolicy{ServiceSrc: svc1, ServiceDst: event.Wildcard, Scheme: policyengine.ECMP, DefaultPeer: peer1}
	err = lb.SetPolicy(&policy)
	require.Nil(t, err)

	peers = repeatLookups(t, lb, svc1, svc2, svc2Peers)
	require.Len(t, peers, 1)
	require.Equal(t, true, peers[peer1])

	peers = repeatLookups(t, lb, svc1, svc3, svc3Peers)
	require.Len(t, peers, 2)
	require.Equal(t, true, peers[peer1])
	require.Equal(t, true, peers[peer2])

	peers = repeatLookups(t, lb, svc1, event.Wildcard, svc1Peers)
	require.Len(t, peers, 2)
	require.Equal(t, true, peers[peer1])
	require.Equal(t, true, peers[peer2])

	peers = repeatLookups(t, lb, event.Wildcard, svc1, svc1Peers)
	require.Len(t, peers, 2)
	require.Equal(t, true, peers[peer1])
	require.Equal(t, true, peers[peer2])
}

func TestLookupBeforeImport(t *testing.T) {
	lb := policyengine.NewLoadBalancer()
	targetPeer, err := lb.LookupWith(svc1, svc2, []string{peer1, peer2})
	require.Nil(t, err)
	require.NotEmpty(t, targetPeer)

	targetPeer, err = lb.LookupWith(svc1, event.Wildcard, []string{peer1, peer2})
	require.Nil(t, err)
	require.NotEmpty(t, targetPeer)

	targetPeer, err = lb.LookupWith(event.Wildcard, svc1, []string{peer1, peer2})
	require.Nil(t, err)
	require.NotEmpty(t, targetPeer)
}

func TestAddPolicyBeforeImport(t *testing.T) {
	lb := policyengine.NewLoadBalancer()

	policy := policyengine.LBPolicy{ServiceSrc: event.Wildcard, ServiceDst: event.Wildcard, Scheme: policyengine.ECMP}
	err := lb.SetPolicy(&policy)
	require.Nil(t, err)

	policy = policyengine.LBPolicy{ServiceSrc: svc1, ServiceDst: event.Wildcard, Scheme: policyengine.ECMP}
	err = lb.SetPolicy(&policy)
	require.Nil(t, err)

	policy = policyengine.LBPolicy{ServiceSrc: event.Wildcard, ServiceDst: svc2, Scheme: policyengine.ECMP}
	err = lb.SetPolicy(&policy)
	require.Nil(t, err)

	policy = policyengine.LBPolicy{ServiceSrc: svc1, ServiceDst: svc2, Scheme: policyengine.ECMP}
	err = lb.SetPolicy(&policy)
	require.Nil(t, err)
}

func TestDeletingDefaultPolicy(t *testing.T) {
	lb := policyengine.NewLoadBalancer()
	policy := policyengine.LBPolicy{ServiceSrc: event.Wildcard, ServiceDst: event.Wildcard, Scheme: policyengine.ECMP}
	err := lb.DeletePolicy(&policy)
	require.NotNil(t, err)
}
