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

package iperf3

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/util"
)

// RunClient runs iperf3 client. Returns bits/second.
func RunClient(cluster *util.KindCluster, server *util.Service) (float64, error) {
	type iperfOutput struct {
		//nolint:tagliatelle // iperf output is out of our control
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
