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

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var ColorReset = "\033[0m"
var ColorGreen = "\033[32m"
var ColorYellow = "\033[33m"

/*******************************************************/
/*   mbg Function                                      */
/*******************************************************/

var ProjDir string = GetProjFolder()

func GetProjFolder() string {
	_, filename, _, _ := runtime.Caller(1)
	return path.Dir(path.Dir(path.Dir(filename)))
}

func SetLog() {
	log.SetFormatter(&log.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		PadLevelText:    true,
		DisableQuote:    true,
	},
	)
}

/*******************************************************/
/*   K8s support functions in go                       */
/*******************************************************/
func PrintHeader(msg string) {
	log.Println(ColorGreen + msg + ColorReset)
}

// Use to get k8s client api
func createClientset() (*kubernetes.Clientset, error) {
	// load the kubeconfig file
	kubeconfigPath := filepath.Join(homedir.HomeDir(), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	// config, err := rest.InClusterConfig()
	// clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Errorf("failed to create Kubernetes API client: %v\n", err)
		return nil, fmt.Errorf("failed to create Kubernetes API client: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Errorf("failed to create Kubernetes API client: %v\n", err)
		return nil, fmt.Errorf("failed to create Kubernetes API client: %v", err)
	}
	return clientset, nil
}

// Check if pod is ready by is label
func PodIsReady(labelSelector string) (bool, error) {
	// create a Kubernetes API client
	clientset, err := createClientset()
	if err != nil {
		return false, fmt.Errorf("failed to create Kubernetes API client: %v", err)
	}
	// retrieve a list of pods matching the label selector
	pods, err := clientset.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return false, fmt.Errorf("failed to retrieve pods: %v", err)
	}

	// check if any pods are ready
	for i := range pods.Items {
		if !IsPodReadyByName(&pods.Items[i]) {
			return false, nil
		}
	}
	return true, nil
}

// Check if pod is ready by the full name of the pod
func IsPodReadyByName(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func GetPodNameIp(name string) (string, string) {
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
		log.Infof("Error retrieving pod: %v\n", err)
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
		log.Infof("Pod %s in namespace %s is not ready\n", name, namespace)
		return "", ""
	}

	// retrieve the pod's IP address
	podIP := pod.Status.PodIP
	log.Infof("Pod %s in namespace %s is ready and has IP address %s\n", name, namespace, podIP)

	return pod.GetName(), podIP
}

func CreateK8sSvc(name, port string) {
	RunCmd("kubectl create service clusterip " + name + " --tcp=" + port + ":" + port)
	RunCmd("kubectl patch service " + name + " -p " + "{\"spec\":{\"selector\":{\"app\":\"mbg\"}}}")
}

// func createK8sService(name, namespace, selectorKey, selectorValue string, port, targetPort int32) error {
// 	config, err := rest.InClusterConfig()
// 	if err != nil {
// 		return err
// 	}
// 	clientset, err := kubernetes.NewForConfig(config)
// 	if err != nil {
// 		return err
// 	}

// 	service := &v1.Service{
// 		ObjectMeta: v1.ObjectMeta{
// 			Name:      name,
// 			Namespace: namespace,
// 		},
// 		Spec: v1.ServiceSpec{
// 			Selector: map[string]string{
// 				selectorKey: selectorValue,
// 			},
// 			Ports: []v1.ServicePort{
// 				{
// 					Name:       "http",
// 					Port:       port,
// 					TargetPort: intstr.FromInt(int(targetPort)),
// 				},
// 			},
// 			Type: v1.ServiceTypeClusterIP,
// 		},
// 	}

// 	_, err = clientset.CoreV1().Services(namespace).Create(context.Background(), service, v1.CreateOptions{})
// 	return err
// }

/*******************************************************/
/*   Execute commands in go                            */
/*******************************************************/
// RunCmdNoPipe executes command and print in the end the result
//
//nolint:gosec // Ignore G204: Subprocess launched with a potential tainted input or cmd arguments
func RunCmdNoPipe(c string) {
	log.Println(ColorYellow + c + ColorReset)
	argSplit := strings.Split(c, " ")
	cmd := exec.Command(argSplit[0], argSplit[1:]...)
	stdout, err := cmd.CombinedOutput()
	if err != nil {
		log.Println(err.Error())
		return
	}

	// Print the output
	fmt.Println(string(stdout))
}

// RunCmd executes command with interactive printing
//
//nolint:gosec // Ignore G204: Subprocess launched with a potential tainted input or cmd arguments
func RunCmd(c string) {
	log.Println(ColorYellow + c + ColorReset)
	argSplit := strings.Split(c, " ")
	cmd := exec.Command(argSplit[0], argSplit[1:]...)
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	// Start command execution
	if err := cmd.Start(); err != nil {
		log.Error("Error starting command:", err)
		return
	}

	// Set up goroutines to read output pipes
	go readOutput(stdout)
	go readOutput(stderr)

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		log.Error("Command returned error:", err)
		return
	}
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

func GetOutput(c string) (string, error) {
	log.Println(ColorYellow + c + ColorReset)
	argSplit := strings.Split(c, " ")
	cmd := exec.Command(argSplit[0], argSplit[1:]...)
	stdout, err := cmd.CombinedOutput()
	if err != nil {
		log.Println(err.Error())
		return "", err
	}

	// Print the output
	return string(stdout), nil
}
