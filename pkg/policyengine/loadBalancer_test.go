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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	crds "github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/policyengine"
)

const (
	svc1  = "svc1"
	svc2  = "svc2"
	svc3  = "svc3"
	peer1 = "peer1"
	peer2 = "peer2"
	peer3 = "peer3"
	ns1   = "ns1"
	ns2   = "ns2"
)

var (
	svc1NS1 = types.NamespacedName{Namespace: ns1, Name: svc1}
	svc2NS1 = types.NamespacedName{Namespace: ns1, Name: svc2}
	svc3NS2 = types.NamespacedName{Namespace: ns2, Name: svc3}
)

func makeSimpleImport(impName, impNs string, peers []string) *crds.Import {
	srcs := []crds.ImportSource{}
	for _, peer := range peers {
		srcs = append(srcs, crds.ImportSource{Peer: peer, ExportName: impName, ExportNamespace: impNs})
	}
	return &crds.Import{
		ObjectMeta: v1.ObjectMeta{Name: impName, Namespace: impNs},
		Spec:       crds.ImportSpec{Sources: srcs},
	}
}

func addImports(lb *policyengine.LoadBalancer) {
	// svc1 is imported from peer1 and peer2
	// svc2 is imported only from peer1
	// svc3 is imported from peer1 and peer2
	lb.AddImport(makeSimpleImport(svc1, ns1, []string{peer1, peer2}))
	lb.AddImport(makeSimpleImport(svc2, ns1, []string{peer1}))
	lb.AddImport(makeSimpleImport(svc3, ns2, []string{peer1, peer2}))
}

// We repeat lookups enough times to make sure we get all the peers allowed by the relevant policy.
func repeatLookups(t *testing.T, lb *policyengine.LoadBalancer,
	svc types.NamespacedName, targets []crds.ImportSource, breakEarly bool,
) map[string]int {
	t.Helper()
	res := map[string]int{}
	for i := 0; i < 100; i++ {
		target, err := lb.LookupWith(svc, targets)
		require.Nil(t, err)

		entry := target.Peer + "/" + target.ExportNamespace + "/" + target.ExportName
		res[entry]++
		if breakEarly && len(res) == len(targets) { // All legal peers appeared in lookup
			break
		}
	}
	return res
}

func TestAddAndDeleteImports(t *testing.T) {
	lb := policyengine.NewLoadBalancer()
	addImports(lb)

	svc1Tgts, err := lb.GetTargetPeers(svc1NS1)
	require.Nil(t, err)
	require.Len(t, svc1Tgts, 2)

	svc2Tgts, err := lb.GetTargetPeers(svc2NS1)
	require.Nil(t, err)
	require.Len(t, svc2Tgts, 1)

	svc3Tgts, err := lb.GetTargetPeers(svc3NS2)
	require.Nil(t, err)
	require.Len(t, svc3Tgts, 2)

	lb.DeleteImport(svc2NS1)
	_, err = lb.GetTargetPeers(svc2NS1)
	require.NotNil(t, err)
	_, err = lb.LookupWith(svc2NS1, svc2Tgts)
	require.NotNil(t, err)
}

func TestLookupWithNoPeers(t *testing.T) {
	lb := policyengine.NewLoadBalancer()
	addImports(lb)

	_, err := lb.LookupWith(svc1NS1, nil)
	require.NotNil(t, err)
}

func TestRandomLookUp(t *testing.T) {
	lb := policyengine.NewLoadBalancer()
	addImports(lb)

	svc1Tgts, err := lb.GetTargetPeers(svc1NS1)
	require.Nil(t, err)

	tgt := repeatLookups(t, lb, svc1NS1, svc1Tgts, true)
	require.Len(t, tgt, 2)
}

func TestFixedPeer(t *testing.T) {
	lb := policyengine.NewLoadBalancer()
	addImports(lb)

	lbPolicy := policyengine.LBPolicy{
		ServiceDst: svc1NS1.String(),
		Scheme:     policyengine.Static,
	}
	err := lb.SetPolicy(&lbPolicy)
	require.Nil(t, err)

	svc1Tgts, err := lb.GetTargetPeers(svc1NS1)
	require.Nil(t, err)

	tgt := repeatLookups(t, lb, svc1NS1, svc1Tgts, true)
	require.Len(t, tgt, 1)
}

func TestRoundRobin(t *testing.T) {
	lb := policyengine.NewLoadBalancer()
	addImports(lb)

	lbPolicy := policyengine.LBPolicy{
		ServiceDst: svc1NS1.String(),
		Scheme:     policyengine.ECMP,
	}
	err := lb.SetPolicy(&lbPolicy)
	require.Nil(t, err)

	svc1Tgts, err := lb.GetTargetPeers(svc1NS1)
	require.Nil(t, err)

	tgts := repeatLookups(t, lb, svc1NS1, svc1Tgts, false)
	require.Len(t, tgts, 2)
	for _, occurrences := range tgts {
		require.Equal(t, 50, occurrences)
	}
}
