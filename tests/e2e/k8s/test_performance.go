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
	"fmt"

	"github.com/stretchr/testify/require"

	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/services/iperf3"
	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/util"
)

func (s *TestSuite) TestPerformance() {
	// measure baseline performance
	baseBPS, err := iperf3.RunClient(s.clusters[0], &iperf3Service)
	require.Nil(s.T(), err)
	s.exportLogs()

	fmt.Printf("Baseline performance: %.2f Gbit/s\n", baseBPS/(1024*1024*1024))

	s.RunOnAllDataplaneTypes(func(cfg *util.PeerConfig) {
		// iperf is expected to generate many MBs of traffic
		cfg.ExpectLargeDataplaneTraffic = true

		cl, err := s.fabric.DeployClusterlinks(2, cfg)
		require.Nil(s.T(), err)

		require.Nil(s.T(), cl[0].CreateExport("iperf3", &iperf3Service))
		require.Nil(s.T(), cl[0].CreatePolicy(util.PolicyAllowAll))
		require.Nil(s.T(), cl[1].CreatePeer(cl[0]))

		importedService := &util.Service{
			Name:      "iperf3",
			Namespace: cl[1].Namespace(),
			Port:      80,
		}
		require.Nil(s.T(), cl[1].CreateImport(importedService, cl[0], httpEchoService.Name))

		require.Nil(s.T(), cl[1].CreatePolicy(util.PolicyAllowAll))

		bps, err := iperf3.RunClient(cl[1].Cluster(), importedService)
		require.Nil(s.T(), err)

		fmt.Printf("Performance drop: %.2f\n", baseBPS/bps)
	})
}
