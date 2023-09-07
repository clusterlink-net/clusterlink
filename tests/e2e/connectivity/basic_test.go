//go:build e2e
// +build e2e

package connectivity

import (
	"strconv"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.ibm.com/mbg-agent/pkg/api"
	"github.ibm.com/mbg-agent/pkg/client"
	"github.ibm.com/mbg-agent/tests/e2e/utils"
)

const (
	gw1Name       = "mbg1"
	gw2Name       = "mbg2"
	curlClient    = "curl-client"
	pingerService = "pinger-server"
	pingerPort    = uint16(3000)
	kindDestPort  = "30001"
)

var (
	allowAllPolicyFile = utils.ProjDir + "/tests/e2e/utils/policy/allowAll.json"
	manifests          = utils.ProjDir + "/tests/e2e/utils/manifests/"
	gwctl1             *client.Client
	gwctl2             *client.Client
)

func TestConnectivity(t *testing.T) {
	t.Run("Starting Cluster Setup", func(t *testing.T) {
		err := utils.StartClusterSetup()
		if err != nil {
			t.Fatalf("Failed to setup cluster")
		}
		err = utils.LaunchApp(gw1Name, curlClient, "curlimages/curl", manifests+curlClient+".yaml")
		if err != nil {
			t.Fatalf("Failed to LaunchApp  curlimages/curl")
		}

		err = utils.LaunchApp(gw2Name, pingerService, "subfuzion/pinger", manifests+pingerService+".yaml")
		if err != nil {
			t.Fatalf("Failed to LaunchApp  subfuzion/pinger")
		}

		err = utils.CreateK8sService(pingerService, strconv.Itoa(int(pingerPort)), kindDestPort)
		if err != nil {
			t.Fatalf("Failed to CreateK8sService")
		}

		gwctl1, err = utils.GetClient(gw1Name)
		if err != nil {
			t.Fatalf("Failed to get Client")
		}
		gwctl2, err = utils.GetClient(gw2Name)
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
		_, destSvcIP := utils.GetPodNameIP(pingerService)
		err := gwctl2.Exports.Create(&api.Export{Name: pingerService, Spec: api.ExportSpec{Service: api.Endpoint{Host: destSvcIP, Port: pingerPort}}})
		require.NoError(t, err)
	})
	t.Run("Testing Import service", func(t *testing.T) {
		err := gwctl1.Imports.Create(&api.Import{Name: pingerService, Spec: api.ImportSpec{Service: api.Endpoint{Host: pingerService, Port: pingerPort}}})
		require.NoError(t, err)
		err = gwctl1.Bindings.Create(&api.Binding{Spec: api.BindingSpec{Import: pingerService, Peer: gw2Name}})
		require.NoError(t, err)
		imp, err := gwctl1.Imports.Get(pingerService)
		require.NoError(t, err)
		impSvc, _ := imp.(*api.Import)
		assert.Equal(t, impSvc.Name, pingerService)
	})
	t.Run("Testing Service Connectivity", func(t *testing.T) {
		policy, err := utils.GetPolicyFromFile(allowAllPolicyFile)
		require.NoError(t, err)
		err = gwctl1.SendAccessPolicy(policy, client.Add)
		require.NoError(t, err)
		err = gwctl2.SendAccessPolicy(policy, client.Add)
		require.NoError(t, err)

		utils.UseKindCluster(gw1Name)
		curlClient, _ := utils.GetPodNameIP(curlClient)
		output, err := utils.GetOutput("kubectl exec -i " + curlClient + " -- curl -s http://pinger-server:3000/ping")
		require.NoError(t, err)
		log.Printf("Got %s", output)
		expected := strings.Split(output, " ")
		assert.Equal(t, "pong", strings.TrimSuffix(expected[1], "\n"))
	})
	utils.CleanUp()
}
