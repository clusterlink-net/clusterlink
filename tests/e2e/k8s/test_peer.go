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

package k8s

import (
	"context"

	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/services/httpecho"
	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/util"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
)

func (s *TestSuite) TestPeerStatus() {
	cl, err := s.fabric.DeployClusterlinks(1, nil)
	require.Nil(s.T(), err)

	peer := &v1alpha1.Peer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cl[0].Name(),
			Namespace: cl[0].Namespace(),
		},
		Spec: v1alpha1.PeerSpec{
			Gateways: []v1alpha1.Endpoint{{
				Host: cl[0].IP(),
				Port: cl[0].Port(),
			}},
		},
	}

	require.Nil(s.T(), cl[0].Cluster().Resources().Create(context.Background(), peer))
	require.Nil(s.T(), cl[0].WaitForPeerCondition(peer, v1alpha1.PeerReachable, true))

	require.Nil(s.T(), cl[0].ScaleDataplane(0))
	require.Nil(s.T(), cl[0].WaitForPeerCondition(peer, v1alpha1.PeerReachable, false))

	require.Nil(s.T(), cl[0].ScaleDataplane(1))
	require.Nil(s.T(), cl[0].WaitForPeerCondition(peer, v1alpha1.PeerReachable, true))
}

func (s *TestSuite) TestPeerMultipleGateways() {
	cl, err := s.fabric.DeployClusterlinks(1, nil)
	require.Nil(s.T(), err)

	importedService := &util.Service{
		Name: "import",
		Port: 80,
	}
	require.Nil(s.T(), cl[0].CreateImport(importedService, cl[0], httpEchoService.Name))

	require.Nil(s.T(), cl[0].CreateService(&httpEchoService))
	require.Nil(s.T(), cl[0].CreateExport(&httpEchoService))
	require.Nil(s.T(), cl[0].CreatePolicy(util.PolicyAllowAll))

	peer := &v1alpha1.Peer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cl[0].Name(),
			Namespace: cl[0].Namespace(),
		},
		Spec: v1alpha1.PeerSpec{
			Gateways: []v1alpha1.Endpoint{
				{
					Host: "bad-host",
					Port: 1234,
				},
				{
					Host: cl[0].IP(),
					Port: cl[0].Port(),
				},
			},
		},
	}

	require.Nil(s.T(), cl[0].Cluster().Resources().Create(context.Background(), peer))
	require.Nil(s.T(), cl[0].WaitForPeerCondition(peer, v1alpha1.PeerReachable, true))

	data, err := cl[0].AccessService(httpecho.GetEchoValue, importedService, true, nil)
	require.Nil(s.T(), err)
	require.Equal(s.T(), cl[0].Name(), data)

	for i := 0; i < 100; i++ {
		data, err := cl[0].AccessService(httpecho.GetEchoValue, importedService, false, nil)
		require.Nil(s.T(), err)
		require.Equal(s.T(), cl[0].Name(), data)
	}
}
