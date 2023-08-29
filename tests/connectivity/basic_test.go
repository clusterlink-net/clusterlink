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
	"github.ibm.com/mbg-agent/pkg/util"
	"github.ibm.com/mbg-agent/tests/utils"
)

const (
	gw1crt  = "mbg1.crt"
	gw1key  = "mbg1.key"
	gw1Name = "mbg1"
	gw2crt  = "mbg2.crt"
	gw2key  = "mbg2.key"
	gw2Name = "mbg2"

	caCrt         = "ca.crt"
	cPortUint     = uint16(30443)
	cPort         = "30443"
	cPortLocal    = "443"
	kindDestPort  = "30001"
	curlClient    = "curl-client"
	pingerService = "pinger-server"
	pingerPort    = uint16(3000)
)

var (
	mtlsFolder string = utils.ProjDir + "/tests/utils/mtls/"
	manifests  string = utils.ProjDir + "/tests/utils/manifests/"
	gwctl1     *client.Client
	gwctl2     *client.Client
)

func TestConnectivity(t *testing.T) {
	t.Run("Starting Cluster Setup", func(t *testing.T) {
		err := startClusterSetup()
		if err != nil {
			t.Fatalf("Failed to setup cluster")
		}
	})

	t.Run("Testing Peering", func(t *testing.T) {
		gw1IP, err := utils.GetKindIP(gw1Name)
		require.NoError(t, err)
		gw2IP, err := utils.GetKindIP(gw2Name)
		require.NoError(t, err)
		err = gwctl1.Peers.Create(&api.Peer{Name: gw2Name, Spec: api.PeerSpec{Gateways: []api.Endpoint{{Host: gw2IP, Port: cPortUint}}}})
		require.NoError(t, err)
		err = gwctl2.Peers.Create(&api.Peer{Name: gw1Name, Spec: api.PeerSpec{Gateways: []api.Endpoint{{Host: gw1IP, Port: cPortUint}}}})
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
		utils.UseKindCluster(gw1Name)
		curlClient, _ := utils.GetPodNameIP(curlClient)
		output, err := utils.GetOutput("kubectl exec -i " + curlClient + " -- curl -s http://pinger-server:3000/ping")
		require.NoError(t, err)
		log.Printf("Got %s", output)
		expected := strings.Split(output, " ")
		assert.Equal(t, "pong", strings.TrimSuffix(expected[1], "\n"))
	})
	cleanup()
}

func startClusterSetup() error {
	utils.StartClusterLink(gw1Name, cPortLocal, cPort, manifests)
	utils.StartClusterLink(gw2Name, cPortLocal, cPort, manifests)
	gw1IP, err := utils.GetKindIP(gw1Name)
	if err != nil {
		return err
	}
	gw2IP, err := utils.GetKindIP(gw2Name)
	if err != nil {
		return err
	}
	parsedCertData, err := util.ParseTLSFiles(mtlsFolder+caCrt, mtlsFolder+gw1crt, mtlsFolder+gw1key)
	if err != nil {
		log.Error(err)
		return err
	}
	gwctl1 = client.New(gw1IP, cPortUint, parsedCertData.ClientConfig(gw1Name))

	parsedCertData, err = util.ParseTLSFiles(mtlsFolder+caCrt, mtlsFolder+gw2crt, mtlsFolder+gw2key)
	if err != nil {
		log.Error(err)
		return err
	}
	gwctl2 = client.New(gw2IP, cPortUint, parsedCertData.ClientConfig(gw2Name))

	return startTestPods()
}

func cleanup() {
	utils.DeleteCluster(gw1Name)
	utils.DeleteCluster(gw2Name)
}

func startTestPods() error {
	err := utils.LaunchApp(gw1Name, curlClient, "curlimages/curl", manifests+curlClient+".yaml")
	if err != nil {
		return err
	}
	err = utils.LaunchApp(gw2Name, pingerService, "subfuzion/pinger", manifests+pingerService+".yaml")
	if err != nil {
		return err
	}
	err = utils.CreateK8sService(pingerService, strconv.Itoa(int(pingerPort)), kindDestPort)
	if err != nil {
		return err
	}
	return nil
}
