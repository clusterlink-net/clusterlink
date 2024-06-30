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
	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/services/httpecho"
	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/util"
	"github.com/stretchr/testify/require"
)

func (s *TestSuite) TestRedundancy() {
	s.RunOnAllDataplaneTypes(func(cfg *util.PeerConfig) {
		cfg.Controlplanes = 3
		cfg.Dataplanes = 3
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

		data, err := cl[1].AccessService(httpecho.GetEchoValue, importedService, true, nil)
		require.Nil(s.T(), err)
		require.Equal(s.T(), cl[0].Name(), data)

		for i := 0; i < 100; i++ {
			data, err := cl[1].AccessService(httpecho.GetEchoValue, importedService, false, nil)
			require.Nil(s.T(), err)
			require.Equal(s.T(), cl[0].Name(), data)
		}
	})
}
