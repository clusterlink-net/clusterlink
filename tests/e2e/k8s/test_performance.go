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
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/util"
)

func (s *TestSuite) TestPerformance() {
	// measure baseline performance
	baseBPS, err := runIperfClient(s.clusters[0], &iperfService)
	require.Nil(s.T(), err)
	s.exportLogs()

	fmt.Printf("Baseline performance: %.2f Gbit/s\n", baseBPS/(1024*1024*1024))

	s.RunOnAllDataplaneTypes(func(cfg *util.PeerConfig) {
		// iperf is expected to generate many MBs of traffic
		cfg.ExpectLargeDataplaneTraffic = true

		cl, err := s.fabric.DeployClusterlinks(2, cfg)
		require.Nil(s.T(), err)

		require.Nil(s.T(), cl[0].CreateExport("iperf3", &iperfService))
		require.Nil(s.T(), cl[0].CreatePolicy(util.PolicyAllowAll))
		require.Nil(s.T(), cl[1].CreatePeer(cl[0]))

		importedService := &util.Service{
			Name:      "iperf3-" + cfg.DataplaneType,
			Namespace: cl[1].Namespace(),
			Port:      80,
		}
		require.Nil(s.T(), cl[1].CreateImport("iperf3", importedService))

		require.Nil(s.T(), cl[1].CreateBinding("iperf3", cl[0]))
		require.Nil(s.T(), cl[1].CreatePolicy(util.PolicyAllowAll))

		bps, err := runIperfClient(cl[1].Cluster(), importedService)
		require.Nil(s.T(), err)

		fmt.Printf("Performance drop: %.2f\n", baseBPS/bps)
	})
}

// returns bits/second.
func runIperfClient(cluster *util.KindCluster, server *util.Service) (float64, error) {
	type iperfOutput struct {
		End struct {
			SumSent struct {
				BitsPerSecond float64 `json:"bits_per_second"`
			} `json:"sum_sent"`
			SumReceived struct {
				BitsPerSecond float64 `json:"bits_per_second"`
			} `json:"sum_received"`
		}
	}

	var output string
	var err error
	for t := time.Now(); time.Since(t) < time.Second*60; time.Sleep(time.Millisecond * 500) {
		output, err = cluster.RunPod(&util.Pod{
			Name:      "iperf3-client",
			Namespace: server.Namespace,
			Image:     "networkstatic/iperf3",
			Args:      []string{"-J", "-c", server.Name, "-p", strconv.Itoa(int(server.Port))},
		})
		if err == nil {
			break
		}
	}
	if err != nil {
		return 0, err
	}

	var result iperfOutput
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return 0, fmt.Errorf("cannot decode iperf results: %w", err)
	}

	return (result.End.SumReceived.BitsPerSecond + result.End.SumSent.BitsPerSecond) / 2, nil
}
