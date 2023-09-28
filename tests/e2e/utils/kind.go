package utils

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

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

// IsPodReady checks if a pod is ready using its label.
func IsPodReady(name string) error {
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
}

// isPodReadyByName checks if a pod is ready by the full name of the pod.
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

// UseKindCluster switches the context to the specified cluster
func UseKindCluster(name string) error {
	return runCmd("kubectl config use-context kind-" + name)
}

func createCluster(name string) (string, error) {
	err := DeleteCluster(name)
	if err != nil {
		return "", err
	}
	err = runCmd("kind create cluster --name=" + name)
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

// DeleteCluster deletes a kind cluster
func DeleteCluster(name string) error {
	return runCmd("kind delete cluster --name=" + name)
}

// StartClusterLink creates a cluster, and launches clusterlink
func StartClusterLink(name, cPortLocal, manifests string, cPort uint16) error {
	certs := "./mtls"
	clusterIP, err := createCluster(name)
	if err != nil {
		return err
	}

	err = runCmd("kubectl apply -f " + manifests + "mbg-role.yaml")
	if err != nil {
		return err
	}

	err = runCmd("kubectl create -f " + manifests + "mbg.yaml")
	if err != nil {
		return err
	}

	err = runCmd("kubectl create -f " + manifests + "dataplane.yaml")
	if err != nil {
		return err
	}

	err = IsPodReady("mbg")
	if err != nil {
		return err
	}

	gwPod, _ := GetPodNameIP("mbg")
	err = IsPodReady("dataplane")
	if err != nil {
		return err
	}

	dpPod, _ := GetPodNameIP("dataplane")
	cPortStr := strconv.Itoa(int(cPort))
	err = runCmd("kubectl create service nodeport dataplane --tcp=" + cPortLocal + ":" + cPortLocal + " --node-port=" + cPortStr)
	if err != nil {
		return err
	}

	startcmd := gwPod + " -- ./controlplane start --id " + name + " --ip " + clusterIP +
		" --cport " + cPortStr + " --cportLocal " + cPortLocal + " --certca " + certs + "/ca.crt --cert " +
		certs + "/" + name + ".crt --key " + certs + "/" + name + ".key"
	err = runCmdB("kubectl exec -i " + startcmd)
	if err != nil {
		return err
	}

	err = runCmdB("kubectl exec -i " + dpPod + " -- ./dataplane --id " + name + " --certca " + certs + "/ca.crt --cert " +
		certs + "/" + name + ".crt --key " + certs + "/" + name + ".key")
	if err != nil {
		return err
	}

	return nil
}

// LaunchApp launches an application using the specified image in the cluster
func LaunchApp(clusterName, svcName, svcImage, svcYaml string) error {
	err := UseKindCluster(clusterName)
	if err != nil {
		return err
	}

	err = runCmd("kind load docker-image " + svcImage + " --name=" + clusterName)
	if err != nil {
		log.Infof("Download locally docker-image %v", svcImage)

	}

	err = runCmd("kubectl create -f " + svcYaml)
	if err != nil {
		return err
	}

	err = IsPodReady(svcName)
	time.Sleep(2 * time.Second)
	return err
}

// CreateK8sService creates a K8s service for an application
func CreateK8sService(name, port, targetPort string) error {
	return runCmd("kubectl create service nodeport " + name + " --tcp=" + port + ":" + port + " --node-port=" + targetPort)
}

// GetKindIP returns the IP of the Kind cluster
func GetKindIP(name string) (string, error) {
	err := UseKindCluster(name)
	if err != nil {
		return "", err
	}

	output, err := exec.Command("kubectl", "get", "nodes", "-o", "jsonpath={.items[0].status.addresses[?(@.type=='InternalIP')].address}").Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}
