//go:build e2e
// +build e2e

package connectivity

import (
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.ibm.com/mbg-agent/pkg/api"
	"github.ibm.com/mbg-agent/pkg/client"
	"github.ibm.com/mbg-agent/pkg/util"
	"github.ibm.com/mbg-agent/tests/utils"
)

const (
	clusterNamePrefix = "connectivity"
	testTimeout       = 10 * time.Minute
	namespace         = "default"
)

var (
	mtlsFolder string = utils.ProjDir + "/demos/utils/"
	manifests  string = utils.ProjDir + "/tests/manifests"
	gwctl1     *client.Client
	gwctl2     *client.Client
)

func TestConnectivity(t *testing.T) {
	t.Run("Starting Cluster Setup", func(t *testing.T) {
		err := StartClusterSetup()
		if err != nil {
			t.Fatalf("Failed to setup cluster")
		}
	})
	t.Run("Testing Peering", func(t *testing.T) {
		mbg1IP, err := utils.GetKindIP(mbg1Name)
		require.NoError(t, err)
		mbg2IP, err := utils.GetKindIP(mbg2Name)
		require.NoError(t, err)
		err = gwctl1.Peers.Create(&api.Peer{Name: mbg2Name, Spec: api.PeerSpec{Gateways: []api.Endpoint{{Host: mbg2IP, Port: mbg2cPort}}}})
		require.NoError(t, err)
		err = gwctl2.Peers.Create(&api.Peer{Name: mbg1Name, Spec: api.PeerSpec{Gateways: []api.Endpoint{{Host: mbg1IP, Port: mbg1cPort}}}})
		require.NoError(t, err)

		peers, err := gwctl1.Peers.List()
		require.NoError(t, err)
		require.NotEmpty(t, peers)
		peers, err = gwctl2.Peers.List()
		require.NoError(t, err)
		require.NotEmpty(t, peers)
	})
	t.Run("Testing Export service", func(t *testing.T) {
		_, destSvcIP := utils.GetPodNameIp(pingerService)
		err := gwctl2.Exports.Create(&api.Export{Name: pingerService, Spec: api.ExportSpec{Service: api.Endpoint{Host: destSvcIP, Port: pingerPort}}})
		require.NoError(t, err)
	})
	t.Run("Testing Import service", func(t *testing.T) {
		err := gwctl1.Imports.Create(&api.Import{Name: pingerService, Spec: api.ImportSpec{Service: api.Endpoint{Host: pingerService, Port: pingerPort}}})
		require.NoError(t, err)

		err = gwctl1.Bindings.Create(&api.Binding{Spec: api.BindingSpec{Import: pingerService, Peer: mbg2Name}})
		require.NoError(t, err)
		imp, err := gwctl1.Imports.Get(pingerService)
		require.NoError(t, err)
		impSvc, _ := imp.(*api.Import)
		assert.Equal(t, impSvc.Name, pingerService)
	})

	t.Run("Testing Service Connectivity", func(t *testing.T) {
		utils.UseKindCluster(mbg1Name)
		curlClient, _ := utils.GetPodNameIp(curlClient)
		output, err := utils.GetOutput("kubectl exec -i " + curlClient + " -- curl -s http://pinger-server:3000/ping")
		require.NoError(t, err)
		log.Printf("Got %s", output)
		expected := strings.Split(output, " ")
		assert.Equal(t, "pong", strings.TrimSuffix(expected[1], "\n"))
	})
}

const (
	mbgCaCrt = "./mtls/ca.crt"
	// MBG1 parameters
	mbg1cPort      = uint16(30443)
	mbg1cPortLocal = "443"
	mbg1crt        = "./mtls/mbg1.crt"
	mbg1key        = "./mtls/mbg1.key"
	mbg1Name       = "mbg1"
	gwctl1Name     = "gwctl1"
	mbg1cni        = "default"
	srcSvc         = "iperf3-client"
	curlClient     = "curl-client"

	// MBG2 parameters
	mbg2cPort      = uint16(30443)
	mbg2cPortLocal = "443"
	mbg2crt        = "./mtls/mbg2.crt"
	mbg2key        = "./mtls/mbg2.key"
	mbg2Name       = "mbg2"
	gwctl2Name     = "gwctl2"
	mbg2cni        = "default"
	destSvc        = "iperf3-server"
	destPort       = uint16(5000)
	kindDestPort   = "30001"
	kindDestPort2  = "30002"
	pingerService  = "pinger-server"
	pingerPort     = uint16(3000)
)

func StartClusterSetup() error {
	// call a Python function
	dataplane := "mtls"
	nologfile := false
	utils.SetLog()

	utils.RunCmd("make clean-kind")

	utils.CreateKindMbg(mbg1Name, dataplane, nologfile)
	utils.CreateKindMbg(mbg2Name, dataplane, nologfile)

	mbg1IP, err := utils.GetKindIP(mbg1Name)
	if err != nil {
		return err
	}
	mbg2IP, err := utils.GetKindIP(mbg2Name)
	if err != nil {
		return err
	}
	parsedCertData, err := util.ParseTLSFiles(mtlsFolder+mbgCaCrt, mtlsFolder+mbg1crt, mtlsFolder+mbg1key)
	if err != nil {
		log.Error(err)
		return err
	}
	gwctl1 = client.New(mbg1IP, mbg1cPort, parsedCertData.ClientConfig(mbg1Name))

	parsedCertData, err = util.ParseTLSFiles(mtlsFolder+mbgCaCrt, mtlsFolder+mbg2crt, mtlsFolder+mbg2key)
	if err != nil {
		log.Error(err)
		return err
	}
	gwctl2 = client.New(mbg2IP, mbg2cPort, parsedCertData.ClientConfig(mbg2Name))

	startTestServices()
	return nil
}

func startTestServices() {
	utils.PrintHeader("Add curl client")
	utils.CreateServiceInKind(mbg1Name, curlClient, "curlimages/curl", manifests+"/"+curlClient+".yaml")

	utils.PrintHeader("Add pinger service")
	utils.CreateServiceInKind(mbg2Name, pingerService, "subfuzion/pinger", manifests+"/"+pingerService+".yaml")
	utils.CreateK8sService(pingerService, "3000", kindDestPort2)
}
