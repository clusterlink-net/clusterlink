// Copyright 2023 The ClusterLink Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

// Data contain the kube informer datat should be part of the control plane.
var Data kubeDataInterface = &kubeData{}

const (
	kubeConfigEnvVariable = "KUBECONFIG"
	syncTime              = 1 * time.Minute
	indexIP               = "byIP"
	typePod               = "Pod"
	typeService           = "Service"
	// AppLabel default pod label kubeinformer use.
	AppLabel = "app"
)

type kubeDataInterface interface {
	GetInfoIP(string) (*Info, error)
	GetInfoApp(string) (*Info, error)
	GetLabel(string, string) (string, error)
	GetIpFromLabel(string) ([]string, error)
	InitFromConfig(string) error
	CreateService(string, int, int, string, string) error
	CreateEndpoint(string, string, string, int) error
	DeleteService(string) error
	DeleteEndpoint(string) error
	CheckServiceExist(string) bool
	CheckEndpointExist(string) bool
}

// kubeData contain all k8s information of the cluster/namespace.
type kubeData struct {
	kubeDataInterface
	// pods and services cache the different object types as *Info pointers
	pods     cache.SharedIndexInformer
	services cache.SharedIndexInformer
	// replicaSets caches the ReplicaSets as partially-filled *ObjectMeta pointers
	replicaSets cache.SharedIndexInformer
	kubeClient  *kubernetes.Clientset
	serviceMap  map[string]string
	endpointMap map[string]string
	stopChan    chan struct{}
}

type owner struct {
	Type string
	Name string
}

// Info contains precollected metadata for Pods and Services.
// Not all the fields are populated for all the above types. To save
// memory, we just keep in memory the necessary data for each Type.
// For more information about which fields are set for each type, please
// refer to the instantiation function of the respective informers.
type Info struct {
	// Informers need that internal object is an ObjectMeta instance
	metav1.ObjectMeta
	Type   string
	Owner  owner
	HostIP string
	ips    []string
}

var commonIndexers = map[string]cache.IndexFunc{
	indexIP: func(obj interface{}) ([]string, error) {
		return obj.(*Info).ips, nil
	},
}

// GetInfoIP Return the pod information according to the pod Ip.
func (k *kubeData) GetInfoIP(ip string) (*Info, error) {
	if info, ok := k.fetchInformers(ip); ok {
		// Owner data might be discovered after the owned, so we fetch it
		// at the last moment
		if info.Owner.Name == "" {
			info.Owner = k.getOwner(info)
		}
		return info, nil
	}

	return nil, fmt.Errorf("informers can't find IP %s", ip)
}

// Return pod information according to the application name.
func (k *kubeData) GetInfoApp(app string) (*Info, error) {
	podLister := k.pods.GetIndexer()
	// Define the label selector
	labelSelector := labels.SelectorFromSet(labels.Set{"app": app})
	// List all pods
	allPods := podLister.List()

	// Find the first pod that matches the label selector.
	for _, pod := range allPods {
		if labelSelector.Matches(labels.Set(pod.(*Info).Labels)) {
			return pod.(*Info), nil
		}
	}
	return nil, fmt.Errorf("informers can't find App %s", app)
}

// Get Ip and key(prefix of label) and return pod label.
func (k *kubeData) GetLabel(ip string, key string) (string, error) {
	if info, ok := k.fetchInformers(ip); ok {
		// Owner data might be discovered after the owned, so we fetch it
		// at the last moment
		if info.Owner.Name == "" {
			info.Owner = k.getOwner(info)
		}
		return info.Labels[key], nil
	}

	return "", fmt.Errorf("informers can't find IP %s", ip)
}

// GetIPFromLabel Get label(prefix of label) and return pod ip.
func (k *kubeData) GetIPFromLabel(label string) ([]string, error) {
	namespace := "default"
	label = AppLabel + "=" + label

	podList, err := k.kubeClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: label,
	})
	if err != nil {
		log.Error(err)
		return nil, err
	}
	if len(podList.Items) == 0 {
		log.Errorf("No pods found for label selector %v", label)
		return nil, fmt.Errorf("no pods found for label selector %q", label)
	}

	podIPs := make([]string, 0, len(podList.Items))
	// We assume that the first matching pod is the correct one
	for _, p := range podList.Items {
		podIPs = append(podIPs, p.Status.PodIP)
	}

	log.Infof("The label %v match to pod ips %v\n", label, podIPs)
	return podIPs, nil
}

func (k *kubeData) fetchInformers(ip string) (*Info, bool) {
	if info, ok := infoForIP(k.pods.GetIndexer(), ip); ok {
		return info, true
	}
	if info, ok := infoForIP(k.services.GetIndexer(), ip); ok {
		return info, true
	}
	return nil, false
}

func infoForIP(idx cache.Indexer, ip string) (*Info, bool) {
	objs, err := idx.ByIndex(indexIP, ip)
	if err != nil {
		log.WithError(err).WithField("ip", ip).Debug("error accessing index. Ignoring")
		return nil, false
	}
	if len(objs) == 0 {
		return nil, false
	}
	return objs[0].(*Info), true
}

func (k *kubeData) getOwner(info *Info) owner {
	if len(info.OwnerReferences) != 0 {
		ownerReference := info.OwnerReferences[0]
		if ownerReference.Kind != "ReplicaSet" {
			return owner{
				Name: ownerReference.Name,
				Type: ownerReference.Kind,
			}
		}

		item, ok, err := k.replicaSets.GetIndexer().GetByKey(info.Namespace + "/" + ownerReference.Name)
		if err != nil {
			log.WithError(err).WithField("key", info.Namespace+"/"+ownerReference.Name).
				Debug("can't get ReplicaSet info from informer. Ignoring")
		} else if ok {
			if rsInfo, ok := item.(*metav1.ObjectMeta); ok {
				if len(rsInfo.OwnerReferences) > 0 {
					return owner{
						Name: rsInfo.OwnerReferences[0].Name,
						Type: rsInfo.OwnerReferences[0].Kind,
					}
				}
			}
		}
	}

	// If no owner references found, return itself as owner
	return owner{
		Name: info.Name,
		Type: info.Type,
	}
}

func (k *kubeData) initPodInformer(informerFactory informers.SharedInformerFactory) error {
	pods := informerFactory.Core().V1().Pods().Informer()
	// Transform any *v1.Pod instance into a *Info instance to save space
	// in the informer's cache
	if err := pods.SetTransform(func(i interface{}) (interface{}, error) {
		pod, ok := i.(*v1.Pod)
		if !ok {
			return nil, fmt.Errorf("was expecting a Pod. Got: %T", i)
		}
		ips := make([]string, 0, len(pod.Status.PodIPs))
		for _, ip := range pod.Status.PodIPs {
			// ignoring host-networked Pod IPs
			if ip.IP != pod.Status.HostIP {
				ips = append(ips, ip.IP)
			}
		}
		return &Info{
			ObjectMeta: metav1.ObjectMeta{
				Name:            pod.Name,
				Namespace:       pod.Namespace,
				Labels:          pod.Labels,
				OwnerReferences: pod.OwnerReferences,
			},
			Type:   typePod,
			HostIP: pod.Status.HostIP,
			ips:    ips,
		}, nil
	}); err != nil {
		return fmt.Errorf("can't set pods transform: %w", err)
	}
	if err := pods.AddIndexers(commonIndexers); err != nil {
		return fmt.Errorf("can't add %s indexer to Pods informer: %w", indexIP, err)
	}

	k.pods = pods
	return nil
}

func (k *kubeData) initServiceInformer(informerFactory informers.SharedInformerFactory) error {
	services := informerFactory.Core().V1().Services().Informer()
	// Transform any *v1.Service instance into a *Info instance to save space
	// in the informer's cache
	if err := services.SetTransform(func(i interface{}) (interface{}, error) {
		svc, ok := i.(*v1.Service)
		if !ok {
			return nil, fmt.Errorf("was expecting a Service. Got: %T", i)
		}
		if svc.Spec.ClusterIP == v1.ClusterIPNone {
			return nil, errors.New("not indexing service without ClusterIP")
		}
		return &Info{
			ObjectMeta: metav1.ObjectMeta{
				Name:      svc.Name,
				Namespace: svc.Namespace,
				Labels:    svc.Labels,
			},
			Type: typeService,
			ips:  svc.Spec.ClusterIPs,
		}, nil
	}); err != nil {
		return fmt.Errorf("can't set services transform: %w", err)
	}
	if err := services.AddIndexers(commonIndexers); err != nil {
		return fmt.Errorf("can't add %s indexer to Pods informer: %w", indexIP, err)
	}

	k.services = services
	return nil
}

func (k *kubeData) initReplicaSetInformer(informerFactory informers.SharedInformerFactory) error {
	k.replicaSets = informerFactory.Apps().V1().ReplicaSets().Informer()
	// To save space, instead of storing a complete *appvs1.Replicaset instance, the
	// informer's cache will store a *metav1.ObjectMeta with the minimal required fields
	if err := k.replicaSets.SetTransform(func(i interface{}) (interface{}, error) {
		rs, ok := i.(*appsv1.ReplicaSet)
		if !ok {
			return nil, fmt.Errorf("was expecting a ReplicaSet. Got: %T", i)
		}
		return &metav1.ObjectMeta{
			Name:            rs.Name,
			Namespace:       rs.Namespace,
			OwnerReferences: rs.OwnerReferences,
		}, nil
	}); err != nil {
		return fmt.Errorf("can't set ReplicaSets transform: %w", err)
	}
	return nil
}

func (k *kubeData) InitFromConfig(kubeConfigPath string) error {
	// Initialization variables
	k.stopChan = make(chan struct{})

	config, err := loadConfig(kubeConfigPath)
	if err != nil {
		return err
	}

	k.kubeClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	err = k.initInformers(k.kubeClient)
	if err != nil {
		return err
	}

	return nil
}

func loadConfig(kubeConfigPath string) (*rest.Config, error) {
	// if no config path is provided, load it from the env variable
	if kubeConfigPath == "" {
		kubeConfigPath = os.Getenv(kubeConfigEnvVariable)
	}
	// otherwise, load it from the $HOME/.kube/config file
	if kubeConfigPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("can't get user home dir: %w", err)
		}
		kubeConfigPath = path.Join(homeDir, ".kube", "config")
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err == nil {
		return config, nil
	}
	// fallback: use in-cluster config
	config, err = rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("can't access kubenetes. Tried using config from: "+
			"config parameter, %s env, homedir and InClusterConfig. Got: %w",
			kubeConfigEnvVariable, err)
	}
	return config, nil
}

func (k *kubeData) initInformers(client kubernetes.Interface) error {
	informerFactory := informers.NewSharedInformerFactory(client, syncTime)

	err := k.initPodInformer(informerFactory)
	if err != nil {
		return err
	}
	err = k.initServiceInformer(informerFactory)
	if err != nil {
		return err
	}
	err = k.initReplicaSetInformer(informerFactory)
	if err != nil {
		return err
	}

	log.Infof("Starting kubernetes informers, waiting for synchronization")
	informerFactory.Start(k.stopChan)
	informerFactory.WaitForCacheSync(k.stopChan)
	k.serviceMap = make(map[string]string)
	k.endpointMap = make(map[string]string)
	log.Infof("Kubernetes informers started")

	return nil
}

// CreateService Add support to create a service/NodePort for a target port.
func (k *kubeData) CreateService(serviceName string, port int, targetPort int, namespace, svcAppName string) error {
	var selectorMap map[string]string
	if svcAppName != "" {
		selectorMap = map[string]string{"app": svcAppName}
	}
	serviceSpec := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: serviceName},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Protocol:   v1.ProtocolTCP,
					Port:       int32(port),
					TargetPort: intstr.FromInt(targetPort),
				},
			},
			Type:     v1.ServiceTypeClusterIP,
			Selector: selectorMap,
		},
	}

	_, err := k.kubeClient.CoreV1().Services(namespace).Create(context.TODO(), serviceSpec, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	k.serviceMap[serviceName] = namespace

	return nil
}

// CreateEndpoint create k8s endpoint.
func (k *kubeData) CreateEndpoint(epName, namespace, targetIP string, targetPort int) error {
	endpoint := &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      epName,
			Namespace: namespace,
		},
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{
						IP: targetIP, // Replace with the desired IP address of the endpoint.
					},
				},
				Ports: []v1.EndpointPort{
					{
						Port: int32(targetPort),
					},
				},
			},
		},
	}

	_, err := k.kubeClient.CoreV1().Endpoints(namespace).Create(context.TODO(), endpoint, metav1.CreateOptions{})
	if err != nil {
		log.Errorf("Error creating endpoint: %s", err)
		return err
	}
	k.serviceMap[epName] = namespace

	return nil
}

// DeleteService delete k8s service.
func (k *kubeData) DeleteService(serviceName string) error {
	if namespace, ok := k.serviceMap[serviceName]; ok {
		return k.kubeClient.CoreV1().Services(namespace).Delete(context.TODO(), serviceName, metav1.DeleteOptions{})
	}

	return fmt.Errorf("serviceName: %s is not exists", serviceName)
}

// DeleteEndpoint delete k8s endpoint.
func (k *kubeData) DeleteEndpoint(epName string) error {
	if namespace, ok := k.endpointMap[epName]; ok {
		return k.kubeClient.CoreV1().Endpoints(namespace).Delete(context.TODO(), epName, metav1.DeleteOptions{})
	}

	return fmt.Errorf("epName: %s is not exists", epName)
}
