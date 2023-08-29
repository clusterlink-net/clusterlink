package utils

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// ProjDir is the current directory of the project
var ProjDir string = getProjFolder()

func getProjFolder() string {
	_, filename, _, _ := runtime.Caller(1)
	return path.Dir(path.Dir(path.Dir(filename)))
}

// Use to get k8s client api
func createClientset() (*kubernetes.Clientset, error) {
	kubeconfigPath := filepath.Join(homedir.HomeDir(), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)

	if err != nil {
		log.Errorf("failed to create Kubernetes API client: %v", err)
		return nil, fmt.Errorf("failed to create Kubernetes API client: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Errorf("failed to create Kubernetes API client: %v", err)
		return nil, fmt.Errorf("failed to create Kubernetes API client: %v", err)
	}
	return clientset, nil
}

// Check if pod is ready using its label
func isPodReady(name string) error {
	namespace := "default"
	clientset, err := createClientset()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes API client: %v", err)
	}

	listOptions := metav1.ListOptions{
		LabelSelector: "app=" + name,
	}
	// retrieve a list of pods matching the label selector
	for {
		pods, _ := clientset.CoreV1().Pods(namespace).List(context.Background(), listOptions)

		if err != nil {
			log.Infof("Error retrieving pod: %v", err)
			return err
		}
		// wait until  pods are ready
		for i := range pods.Items {
			if isPodReadyByName(&pods.Items[i]) {
				return nil
			}
		}
		time.Sleep(time.Second)
	}
	return nil
}

// Check if pod is ready by the full name of the pod
func isPodReadyByName(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

// GetPodNameIP returns the pod name and pod IP
func GetPodNameIP(name string) (string, string) {
	clientset, err := createClientset()
	if err != nil {
		return "", ""
	}
	// specify the namespace and name of the pod you want to check
	namespace := "default"

	// retrieve the pod object from the API server
	// Set the label selector for the pod
	labelSelector := "app=" + name

	listOptions := metav1.ListOptions{
		LabelSelector: "app=" + name,
	}
	pods, _ := clientset.CoreV1().Pods(namespace).List(context.Background(), listOptions)

	if err != nil {
		log.Infof("Error retrieving pod: %v", err)
		return "", ""
	}

	// Print the name of the first pod in the list
	if len(pods.Items) > 0 {
		log.Info("Pod name:", pods.Items[0].GetName())
	} else {
		log.Info("No pods found with label:", labelSelector)
		return "", ""
	}
	pod := pods.Items[0]
	// check if the pod is ready
	isReady := false
	for _, condition := range pod.Status.Conditions {
		if condition.Type == "Ready" && condition.Status == "True" {
			isReady = true
			break
		}
	}
	if !isReady {
		log.Infof("Pod %s in namespace %s is not ready", name, namespace)
		return "", ""
	}

	// retrieve the pod's IP address
	podIP := pod.Status.PodIP
	log.Infof("Pod %s in namespace %s is ready and has IP address %s", name, namespace, podIP)

	return pod.GetName(), podIP
}

func runCmdB(c string) error {
	log.Println(c)
	argSplit := strings.Split(c, " ")
	cmd := exec.Command(argSplit[0], argSplit[1:]...)
	if err := cmd.Start(); err != nil {
		log.Error("Error starting command:", err)
		return err
	}
	time.Sleep(time.Second)
	return nil
}

//nolint:gosec // Ignore G204: Subprocess launched with a potential tainted input or cmd arguments
func runCmd(c string) error {
	log.Println(c)
	argSplit := strings.Split(c, " ")
	cmd := exec.Command(argSplit[0], argSplit[1:]...)
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	// Start command execution
	if err := cmd.Start(); err != nil {
		log.Error("Error starting command:", err)
		return err
	}

	// Set up goroutines to read output pipes
	go readOutput(stdout)
	go readOutput(stderr)

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		log.Error("Command returned error:", err)
		return err
	}
	return nil
}

func readOutput(pipe io.Reader) {
	buf := make([]byte, 1024)
	for {
		n, err := pipe.Read(buf)
		if n > 0 {
			fmt.Print(string(buf[:n]))
		}

		if err != nil {
			if err != io.EOF && err != io.ErrClosedPipe && !strings.Contains(err.Error(), "file already closed") {
				log.Error("Error reading output:", err, err.Error())
			}
			break
		}
	}
}

// GetOutput returns the output of a specified command
func GetOutput(c string) (string, error) {
	log.Println(c)
	argSplit := strings.Split(c, " ")
	cmd := exec.Command(argSplit[0], argSplit[1:]...)
	stdout, err := cmd.CombinedOutput()
	if err != nil {
		log.Println(err.Error())
		return "", err
	}
	return string(stdout), nil
}

// UseKindCluster switches the context to the specified cluster
func UseKindCluster(name string) {
	runCmd("kubectl config use-context kind-" + name)
}

func createCluster(name string) (string, error) {
	err := runCmd("kind create cluster --name=" + name)
	if err != nil {
		return "", err
	}
	err = runCmd("kind load docker-image mbg --name=" + name)
	if err != nil {
		return "", err
	}
	ip, err := GetKindIP(name)
	return ip, err
}

func DeleteCluster(name string) {
	runCmd("kind delete cluster --name=" + name)
}

// StartClusterLink creates a cluster, and launches clusterlink
func StartClusterLink(name, cPortLocal, cPort, manifests string) error {
	certs := "./mtls"
	clusterIP, err := createCluster(name)
	if err != nil {
		return err
	}
	runCmd("kubectl apply -f " + manifests + "mbg-role.yaml")
	runCmd("kubectl create -f " + manifests + "mbg.yaml")
	runCmd("kubectl create -f " + manifests + "dataplane.yaml")
	err = isPodReady("mbg")
	if err != nil {
		return err
	}
	gwPod, _ := GetPodNameIP("mbg")
	err = isPodReady("dataplane")
	if err != nil {
		return err
	}
	dpPod, _ := GetPodNameIP("dataplane")
	runCmd("kubectl create service nodeport dataplane --tcp=" + cPortLocal + ":" + cPortLocal + " --node-port=" + cPort)
	startcmd := gwPod + " -- ./controlplane start --id " + name + " --ip " + clusterIP +
		" --cport " + cPort + " --cportLocal " + cPortLocal + " --certca " + certs + "/ca.crt --cert " +
		certs + "/" + name + ".crt --key " + certs + "/" + name + ".key"
	runCmdB("kubectl exec -i " + startcmd)
	runCmdB("kubectl exec -i " + dpPod + " -- ./dataplane --id " + name + " --certca " + certs + "/ca.crt --cert " +
		certs + "/" + name + ".crt --key " + certs + "/" + name + ".key")
	return nil
}

// LaunchApp launches an application using the specified image in the cluster
func LaunchApp(clusterName, svcName, svcImage, svcYaml string) error {
	UseKindCluster(clusterName)
	runCmd("kind load docker-image " + svcImage + " --name=" + clusterName)
	err := runCmd("kubectl create -f " + svcYaml)
	if err != nil {
		return err
	}

	err = isPodReady(svcName)
	time.Sleep(2 * time.Second)
	return err
}

// CreateK8sService creates a K8s service for an application
func CreateK8sService(name, port, targetPort string) error {
	return runCmd("kubectl create service nodeport " + name + " --tcp=" + port + ":" + port + " --node-port=" + targetPort)
}

// GetKindIP returns the IP of the Kind cluster
func GetKindIP(name string) (string, error) {
	UseKindCluster(name)
	output, err := exec.Command("kubectl", "get", "nodes", "-o", "jsonpath={.items[0].status.addresses[?(@.type=='InternalIP')].address}").Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}
