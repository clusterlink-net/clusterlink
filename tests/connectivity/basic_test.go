//go:build e2e
// +build e2e

package connectivity

import (
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
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
	folCl      string = utils.ProjDir + "/demos/iperf3/manifests/iperf3-client"
	folSv      string = utils.ProjDir + "/demos/iperf3/manifests/iperf3-server"
	gwctl1     *client.Client
	gwctl2     *client.Client
)

func TestConnectivity(t *testing.T) {
	t.Logf("Start testing connectivity")
	t.Run("Testing Connectivity", func(t *testing.T) {
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
	t.Run("Testing Exports", func(t *testing.T) {
		_, destSvcIP := utils.GetPodNameIp(destSvc)
		err := gwctl2.Exports.Create(&api.Export{Name: destSvc, Spec: api.ExportSpec{Service: api.Endpoint{Host: destSvcIP, Port: destPort}}})
		require.NoError(t, err)
	})
	t.Run("Testing Imports", func(t *testing.T) {
		utils.PrintHeader("Start importing")
		utils.UseKindCluster(mbg2Name)
		err := gwctl2.Imports.Create(&api.Import{Name: destSvc, Spec: api.ImportSpec{Service: api.Endpoint{Host: destSvc, Port: destPort}}})
		require.NoError(t, err)

		utils.PrintHeader("Bind a service")
		utils.UseKindCluster(mbg1Name)
		err = gwctl1.Bindings.Create(&api.Binding{Spec: api.BindingSpec{Import: destSvc, Peer: mbg2Name}})
		require.NoError(t, err)
	})
}

const (
	mbgCaCrt = "./mtls/ca.crt"
	// MBG1 parameters
	mbg1DataPort   = "30001"
	mbg1cPort      = uint16(30443)
	mbg1cPortLocal = "443"
	mbg1crt        = "./mtls/mbg1.crt"
	mbg1key        = "./mtls/mbg1.key"
	mbg1Name       = "mbg1"
	gwctl1Name     = "gwctl1"
	mbg1cni        = "default"
	srcSvc         = "iperf3-client"

	// MBG2 parameters
	mbg2DataPort   = "30001"
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

	startServices()
	return nil
}

func startServices() {
	utils.PrintHeader("Add iperf3 client")
	utils.CreateServiceInKind(mbg1Name, srcSvc, "mlabbe/iperf3", folCl+"/"+srcSvc+".yaml")

	utils.PrintHeader("Add iperf3 server")
	utils.CreateServiceInKind(mbg2Name, destSvc, "mlabbe/iperf3", folSv+"/iperf3.yaml")
}
