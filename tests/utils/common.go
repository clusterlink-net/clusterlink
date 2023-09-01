package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.ibm.com/mbg-agent/pkg/api"
	"github.ibm.com/mbg-agent/pkg/client"
	"github.ibm.com/mbg-agent/pkg/util"
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
	mtlsFolder = ProjDir + "/tests/utils/mtls/"
	manifests  = ProjDir + "/tests/utils/manifests/"
	gwctl1     *client.Client
	gwctl2     *client.Client
)

func StartClusterSetup() error {
	StartClusterLink(gw1Name, cPortLocal, cPort, manifests)
	StartClusterLink(gw2Name, cPortLocal, cPort, manifests)
	return startTestPods()
}

func GetClient(name string) (*client.Client, error) {
	gwIP, err := GetKindIP(name)
	if err != nil {
		return nil, err
	}
	parsedCertData, err := util.ParseTLSFiles(mtlsFolder+caCrt, mtlsFolder+gw1crt, mtlsFolder+gw1key)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	gwctl := client.New(gwIP, cPortUint, parsedCertData.ClientConfig(name))
	return gwctl, nil
}
func GetClients() (*client.Client, *client.Client, error) {
	gw1IP, err := GetKindIP(gw1Name)
	if err != nil {
		return nil, nil, err
	}
	gw2IP, err := GetKindIP(gw2Name)
	if err != nil {
		return nil, nil, err
	}
	parsedCertData, err := util.ParseTLSFiles(mtlsFolder+caCrt, mtlsFolder+gw1crt, mtlsFolder+gw1key)
	if err != nil {
		log.Error(err)
		return nil, nil, err
	}
	gwctl1 = client.New(gw1IP, cPortUint, parsedCertData.ClientConfig(gw1Name))

	parsedCertData, err = util.ParseTLSFiles(mtlsFolder+caCrt, mtlsFolder+gw2crt, mtlsFolder+gw2key)
	if err != nil {
		log.Error(err)
		return nil, nil, err
	}
	gwctl2 = client.New(gw2IP, cPortUint, parsedCertData.ClientConfig(gw2Name))
	return gwctl1, gwctl2, nil
}
func cleanup() {
	DeleteCluster(gw1Name)
	DeleteCluster(gw2Name)
}

func startTestPods() error {
	err := LaunchApp(gw1Name, curlClient, "curlimages/curl", manifests+curlClient+".yaml")
	if err != nil {
		return err
	}
	err = LaunchApp(gw2Name, pingerService, "subfuzion/pinger", manifests+pingerService+".yaml")
	if err != nil {
		return err
	}
	err = CreateK8sService(pingerService, strconv.Itoa(int(pingerPort)), kindDestPort)
	if err != nil {
		return err
	}
	return nil
}

func GetPolicyFromFile(filename string) (api.Policy, error) {
	fileBuf, err := os.ReadFile(filename)
	if err != nil {
		return api.Policy{}, fmt.Errorf("error reading policy file: %w", err)
	}
	var policy api.Policy
	err = json.Unmarshal(fileBuf, &policy)
	if err != nil {
		return api.Policy{}, fmt.Errorf("error parsing Json in policy file: %w", err)
	}
	policy.Spec.Blob = fileBuf
	return policy, nil
}
