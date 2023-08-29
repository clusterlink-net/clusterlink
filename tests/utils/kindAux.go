package utils

import (
	"os/exec"
	"time"
)

func UseKindCluster(name string) {
	RunCmd("kubectl config use-context kind-" + name)
}

func CreateKindMbg(name, dataplane string, logfile bool) { // use Python script -TODO change to go
	script := ProjDir + "/demos/iperf3/kind/start_cluster_mbg.py"
	cmd := script
	cmd += " -m " + name + " -d " + dataplane
	if logfile {
		cmd += " --noLogFile"
	}
	RunCmd(cmd)
}

func CreateServiceInKind(mbgName, svcName, svcImage, svcYaml string) {
	UseKindCluster(mbgName)
	RunCmd("kind load docker-image " + svcImage + " --name=" + mbgName)
	RunCmd("kubectl create -f " + svcYaml)

	_, _ = PodIsReady(svcName) // intentionally ignoring errors on demo files
	time.Sleep(2 * time.Second)
}

func CreateK8sService(name, port, targetPort string) {
	RunCmd("kubectl create service nodeport " + name + " --tcp=" + port + ":" + port + " --node-port=" + targetPort)
}

func GetKindIP(name string) (string, error) {
	UseKindCluster(name)
	output, err := exec.Command("kubectl", "get", "nodes", "-o", "jsonpath={.items[0].status.addresses[?(@.type=='InternalIP')].address}").Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}
