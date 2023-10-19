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

// ###############################################################
// Name: Simple iperf3  test
// Desc: create 2 kind clusters :
// 1) MBG and iperf3 client
// 2) MBG and iperf3 server
// ##############################################################
package iperf3_test

import (
	"flag"
	"log"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/clusterlink-net/clusterlink/pkg/api"
	"github.com/clusterlink-net/clusterlink/pkg/client"
	logutils "github.com/clusterlink-net/clusterlink/pkg/util/log"
	"github.com/clusterlink-net/clusterlink/tests/e2e/utils"
)

const (
	gw1Name        = "mbg1"
	gw2Name        = "mbg2"
	srcSvc         = "iperf3-client"
	destSvc        = "iperf3-server"
	destPort       = uint16(5000)
	kindDirectPort = "30001"
)

var (
	allowAllPolicyFile = utils.ProjDir + "/tests/e2e/utils/testdata/policy/allowAll.json"
	gwctl1             *client.Client
	gwctl2             *client.Client
)

var cpType = flag.String("controlplane", "new", "Check which control-plane to use")

// TestIperf3 check e2e iperf3 test
func TestIperf3(t *testing.T) {
	_, err := logutils.SetLog("info", "")
	require.NoError(t, err)
	t.Run("Starting Cluster Setup", func(t *testing.T) {
		err := utils.StartClusterSetup(*cpType)
		if err != nil {
			t.Fatalf("Failed to setup cluster")
		}
		err = utils.LaunchApp(gw1Name, srcSvc, "mlabbe/iperf3", utils.ProjDir+"/tests/e2e/utils/testdata/manifests/iperf3/iperf3-client.yaml")
		if err != nil {
			t.Fatalf("Failed to LaunchApp iperf3 client mlabbe/iperf3")
		}

		err = utils.LaunchApp(gw2Name, destSvc, "mlabbe/iperf3", utils.ProjDir+"/tests/e2e/utils/testdata/manifests/iperf3/iperf3-server.yaml")
		if err != nil {
			t.Fatalf("Failed to LaunchApp iperf3 server mlabbe/iperf3")
		}

		gwctl1, err = utils.GetClient(gw1Name, *cpType)
		if err != nil {
			t.Fatalf("Failed to get Client")
		}
		gwctl2, err = utils.GetClient(gw2Name, *cpType)
		if err != nil {
			t.Fatalf("Failed to get Client")
		}
	})

	t.Run("Testing Peering", func(t *testing.T) {
		gw1IP, err := utils.GetKindIP(gw1Name)
		require.NoError(t, err)
		gw2IP, err := utils.GetKindIP(gw2Name)
		require.NoError(t, err)
		err = gwctl1.Peers.Create(&api.Peer{Name: gw2Name, Spec: api.PeerSpec{Gateways: []api.Endpoint{{Host: gw2IP, Port: utils.ControlPort}}}})
		require.NoError(t, err)
		err = gwctl2.Peers.Create(&api.Peer{Name: gw1Name, Spec: api.PeerSpec{Gateways: []api.Endpoint{{Host: gw1IP, Port: utils.ControlPort}}}})
		require.NoError(t, err)

		peers, err := gwctl1.Peers.List()
		require.NoError(t, err)
		require.NotEmpty(t, peers)
		peers, err = gwctl2.Peers.List()
		require.NoError(t, err)
		require.NotEmpty(t, peers)
	})
	t.Run("Testing Export service", func(t *testing.T) {
		err := gwctl2.Exports.Create(&api.Export{Name: destSvc, Spec: api.ExportSpec{Service: api.Endpoint{Host: destSvc, Port: destPort}}})
		require.NoError(t, err)
	})
	t.Run("Testing Import service", func(t *testing.T) {
		err := gwctl1.Imports.Create(&api.Import{Name: destSvc, Spec: api.ImportSpec{Service: api.Endpoint{Host: destSvc, Port: destPort}}})
		require.NoError(t, err)
		err = gwctl1.Bindings.Create(&api.Binding{Spec: api.BindingSpec{Import: destSvc, Peer: gw2Name}})
		require.NoError(t, err)
		imp, err := gwctl1.Imports.Get(destSvc)
		require.NoError(t, err)
		impSvc, _ := imp.(*api.Import)
		assert.Equal(t, impSvc.Name, destSvc)
	})
	t.Run("Testing policy", func(t *testing.T) {
		policy, err := utils.GetPolicyFromFile(allowAllPolicyFile)
		require.NoError(t, err)
		if *cpType == "new" {
			err = gwctl1.Policies.Create(policy)
			require.NoError(t, err)
			err = gwctl2.Policies.Create(policy)
			require.NoError(t, err)
		} else {
			err = gwctl1.SendAccessPolicy(policy, client.Add)
			require.NoError(t, err)
			err = gwctl2.SendAccessPolicy(policy, client.Add)
			require.NoError(t, err)
		}

	})
	t.Run("Testing Service Connectivity", func(t *testing.T) {
		mbg2Ip, _ := utils.GetKindIP(gw2Name)
		err := utils.UseKindCluster(gw1Name)
		require.NoError(t, err)
		iperf3Pod, _ := utils.GetPodNameIP(srcSvc)
		log.Println("Direct test")
		output, err := utils.GetOutput("kubectl exec -i " + iperf3Pod + " -- iperf3 -c " + mbg2Ip + " -p " + kindDirectPort)
		require.NoError(t, err)
		log.Printf("%s", output)
		log.Println("Test using the GWs")
		output, err = utils.GetOutput("kubectl exec -i " + iperf3Pod + " -- iperf3 -c " + destSvc + " -p " + strconv.Itoa(int(destPort)))
		require.NoError(t, err)
		log.Printf("%s", output)
	})
	err = utils.CleanUp()
	require.NoError(t, err)
}
