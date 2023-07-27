// ###############################################################
// Name: Simple iperf3  test
// Desc: create 2 kind clusters :
// 1) MBG and iperf3 client
// 2) MBG and iperf3 server
// ##############################################################
package main

import (
	"os"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	"github.ibm.com/mbg-agent/cmd/gwctl/config"
	"github.ibm.com/mbg-agent/pkg/admin"
	api "github.ibm.com/mbg-agent/pkg/api/admin"
	mbgAux "github.ibm.com/mbg-agent/tests/utils"
	kindAux "github.ibm.com/mbg-agent/tests/utils/kind"
)

const (
	mbgCaCrt = "./mtls/ca.crt"
	//MBG1 parameters
	mbg1DataPort   = "30001"
	mbg1cPort      = 30443
	mbg1cPortLocal = "443"
	mbg1crt        = "./mtls/mbg1.crt"
	mbg1key        = "./mtls/mbg1.key"
	mbg1Name       = "mbg1"
	gwctl1Name     = "gwctl1"
	mbg1cni        = "default"
	srcSvc         = "iperf3-client"

	//MBG2 parameters
	mbg2DataPort   = "30001"
	mbg2cPort      = 30443
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

var (
	mtlsFolder string = mbgAux.ProjDir + "/tests/utils/"
	folCl      string = mbgAux.ProjDir + "/tests/iperf3/manifests/iperf3-client"
	folSv      string = mbgAux.ProjDir + "/tests/iperf3/manifests/iperf3-server"
)

func main() {
	// call a Python function
	dataplane := "mtls"
	nologfile := false
	mbgAux.SetLog()
	log.Println("Working directory", mbgAux.ProjDir)
	//exec.chdir(proj_dir)
	//clean
	log.Print("Clean old kinds")
	mbgAux.RunCmd("make clean-kind")

	// build docker environment
	mbgAux.PrintHeader("Build docker image")
	mbgAux.RunCmd("make docker-build")
	kindAux.CreateKindMbg(mbg1Name, dataplane, nologfile)
	kindAux.CreateKindMbg(mbg2Name, dataplane, nologfile)

	// //get parameters
	mbg1Ip, _ := kindAux.GetKindIp(mbg1Name)
	mbg2Ip, _ := kindAux.GetKindIp(mbg2Name)

	//set gwctl
	gwctl1, err := admin.NewClient(config.ClientConfig{
		ID:        gwctl1Name,
		GwIP:      mbg1Ip + ":" + strconv.Itoa(mbg1cPort),
		CaFile:    mtlsFolder + mbgCaCrt,
		CertFile:  mtlsFolder + mbg1crt,
		KeyFile:   mtlsFolder + mbg1key,
		Dataplane: dataplane,
	})
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
	gwctl2, err := admin.NewClient(config.ClientConfig{
		ID:        gwctl2Name,
		GwIP:      mbg2Ip + ":" + strconv.Itoa(mbg2cPort),
		CaFile:    mtlsFolder + mbgCaCrt,
		CertFile:  mtlsFolder + mbg2crt,
		KeyFile:   mtlsFolder + mbg2key,
		Dataplane: dataplane,
	})

	if err != nil {
		log.Error(err)
		os.Exit(1)
	}
	//Add Peer
	mbgAux.PrintHeader("Add peers and send hello")
	gwctl1.CreatePeer(api.Peer{Name: mbg2Name, Spec: api.PeerSpec{Gateways: []api.Endpoint{{Host: mbg2Ip, Port: mbg2cPort}}}})

	//Set iperf3 client
	mbgAux.PrintHeader("Add iperf3 client")
	kindAux.CreateServiceInKind(mbg1Name, srcSvc, "mlabbe/iperf3", folCl+"/"+srcSvc+".yaml")
	srcSvcPod, _ := mbgAux.GetPodNameIp(srcSvc)
	//gwctl1.AddService(srcSvc, "", "", "iperf3 client") //Allow to use all by default

	//Set iperf3 server
	mbgAux.PrintHeader("Add iperf3 server")
	kindAux.CreateServiceInKind(mbg2Name, destSvc, "mlabbe/iperf3", folSv+"/iperf3.yaml")
	destSvcPod, destSvcIP := mbgAux.GetPodNameIp(destSvc)

	gwctl2.CreateExportService(api.Export{Name: destSvc, Spec: api.ExportSpec{Service: api.Endpoint{Host: destSvcIP, Port: destPort}}})
	log.Println(srcSvcPod, destSvcPod)

	//Expose service
	mbgAux.PrintHeader("Start expose")
	kindAux.UseKindCluster(mbg2Name)
	gwctl2.CreateImportService(api.Import{Name: destSvc, Spec: api.ImportSpec{Service: api.Endpoint{Host: destSvc, Port: destPort}}})

	//bindK8sSvc()
	mbgAux.PrintHeader("Bind a service")
	kindAux.UseKindCluster(mbg1Name)
	gwctl1.CreateBinding(api.Binding{Spec: api.BindingSpec{Import: destSvc, Peer: mbg2Name}})
	time.Sleep(5 * time.Second)

	//iperf3test
	mbgAux.RunCmdNoPipe("kubectl exec -i " + srcSvcPod + " -- iperf3 -c " + destSvc + " -p " + strconv.Itoa(int(destPort)))

}

// ############################### MAIN ##########################
// if __name__ == "__main__":
//     parser = argparse.ArgumentParser(description='Description of your program')
//     parser.add_argument('-d','--dataplane', help='choose which dataplane to use mtls/tcp', required=False, default="mtls")
//     parser.add_argument('-c','--cni', help='Which cni to use default(kindnet)/flannel/calico/diff (different cni for each cluster)', required=False, default="default")
