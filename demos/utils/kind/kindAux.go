package kindAux

import (
	"os/exec"
	"time"

	mbgAux "github.ibm.com/mbg-agent/demos/utils"
)

func UseKindCluster(name string) {
	mbgAux.RunCmd("kubectl config use-context kind-" + name)
}

func CreateKindMbg(name, dataplane string, logfile bool) { //use Python script -TODO change to go
	script := mbgAux.ProjDir + "/demos/iperf3/kind/start_cluster_mbg.py"
	cmd := script
	cmd += " -m " + name + " -d " + dataplane
	if logfile {
		cmd += " --noLogFile"
	}
	mbgAux.RunCmd(cmd)
}

func CreateServiceInKind(mbgName, svcName, svcImage, svcYaml string) {
	UseKindCluster(mbgName)
	mbgAux.RunCmd("kind load docker-image " + svcImage + " --name=" + mbgName)
	mbgAux.RunCmd("kubectl create -f " + svcYaml)
	mbgAux.PodIsReady(svcName)
	time.Sleep(2 * time.Second)
}

func GetKindIp(name string) (string, error) {
	UseKindCluster(name)
	output, err := exec.Command("kubectl", "get", "nodes", "-o", "jsonpath={.items[0].status.addresses[?(@.type=='InternalIP')].address}").Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}
