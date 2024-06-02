// Copyright (c) The ClusterLink Authors.
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
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/clusterlink-net/clusterlink/cmd/clusterlink/config"
	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/bootstrap"
	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/services"
	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/services/httpecho"
	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/util"
)

func (s *TestSuite) TestDynamicPeerCertificates() {
	s.RunOnAllDataplaneTypes(func(cfg *util.PeerConfig) {
		cl, err := s.fabric.DeployClusterlinks(2, cfg)
		require.Nil(s.T(), err)

		require.Nil(s.T(), cl[0].CreateService(&httpEchoService))
		require.Nil(s.T(), cl[0].CreateExport(&httpEchoService))
		require.Nil(s.T(), cl[0].CreatePolicy(util.PolicyAllowAll))
		require.Nil(s.T(), cl[1].CreatePeer(cl[0]))

		importedService := &util.Service{
			Name: httpEchoService.Name,
			Port: 80,
		}
		require.Nil(s.T(), cl[1].CreateImport(importedService, cl[0], httpEchoService.Name))

		require.Nil(s.T(), cl[1].CreatePolicy(util.PolicyAllowAll))

		_, err = cl[1].AccessService(httpecho.GetEchoValue, importedService, true, nil)
		require.Nil(s.T(), err)

		// create a new fabric certificate
		fabricCert, err := bootstrap.CreateFabricCertificate(config.DefaultFabric)
		require.Nil(s.T(), err)

		// create new peer certificates
		var peerCerts []*bootstrap.Certificate
		for i := 0; i < 2; i++ {
			peerCert, err := bootstrap.CreatePeerCertificate(cl[0].Name(), fabricCert)
			require.Nil(s.T(), err)
			peerCerts = append(peerCerts, peerCert)
		}

		// update peer certificates on cl[1]
		require.Nil(s.T(), cl[1].UpdatePeerCertificates(fabricCert, peerCerts[0]))

		// verify peer becomes unreachable
		require.Nil(s.T(), cl[1].WaitForPeerCondition(
			&v1alpha1.Peer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cl[0].Name(),
					Namespace: cl[0].Namespace(),
				},
			},
			v1alpha1.PeerReachable,
			false))

		// verify service is no longer accessible
		_, err = cl[1].AccessService(httpecho.GetEchoValue, importedService, false, &services.ConnectionResetError{})
		require.ErrorIs(s.T(), err, &services.ConnectionResetError{})

		// update peer certificates on cl[0]
		require.Nil(s.T(), cl[0].UpdatePeerCertificates(fabricCert, peerCerts[1]))

		// verify peer becomes reachable
		require.Nil(s.T(), cl[1].WaitForPeerCondition(
			&v1alpha1.Peer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cl[0].Name(),
					Namespace: cl[0].Namespace(),
				},
			},
			v1alpha1.PeerReachable,
			true))

		// verify access is back
		_, err = cl[1].AccessService(httpecho.GetEchoValue, importedService, false, nil)
		require.Nil(s.T(), err)
	})
}
