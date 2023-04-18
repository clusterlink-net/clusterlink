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
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

var Data kubeDataInterface = &KubeData{}

const (
	kubeConfigEnvVariable = "KUBECONFIG"
	syncTime              = 1 * time.Minute
	IndexIP               = "byIP"
	typePod               = "Pod"
	typeService           = "Service"
)
const APP_LABEL = "app"

type kubeDataInterface interface {
	GetInfo(string) (*Info, error)
	GetLabel(string, string) (string, error)
	GetIpFromLabel(string) ([]string, error)
	InitFromConfig(string) error
}

type KubeData struct {
	kubeDataInterface
	// pods and services cache the different object types as *Info pointers
	pods     cache.SharedIndexInformer
	services cache.SharedIndexInformer
	// replicaSets caches the ReplicaSets as partially-filled *ObjectMeta pointers
	replicaSets cache.SharedIndexInformer
	stopChan    chan struct{}
}

type Owner struct {
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
	Owner  Owner
	HostIP string
	ips    []string
}

var commonIndexers = map[string]cache.IndexFunc{
	IndexIP: func(obj interface{}) ([]string, error) {
		return obj.(*Info).ips, nil
	},
}

func (k *KubeData) GetInfo(ip string) (*Info, error) {
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

// Get Ip and key(prefix of label) and return pod label
func (k *KubeData) GetLabel(ip string, key string) (string, error) {
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

// Get label(prefix of label) and return pod ip
func (k *KubeData) GetIpFromLabel(label string) ([]string, error) {
	namespace := "default"
	label = APP_LABEL + "=" + label
	// Create a Kubernetes clientset using the in-cluster configuration
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	podList, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: label,
	})
	if err != nil {
		log.Error(err)
		return nil, err
	}
	if len(podList.Items) == 0 {
		log.Error("No pods found for label selector %v", label)
		return nil, fmt.Errorf("No pods found for label selector %q", label)
	}

	var podIPs []string

	// We assume that the first matching pod is the correct one
	for _, p := range podList.Items {
		podIPs = append(podIPs, p.Status.PodIP)
	}

	log.Infof("The label %v match to pod ips %v\n", label, podIPs)
	return podIPs, nil
}

func (k *KubeData) fetchInformers(ip string) (*Info, bool) {
	if info, ok := infoForIP(k.pods.GetIndexer(), ip); ok {
		return info, true
	}
	if info, ok := infoForIP(k.services.GetIndexer(), ip); ok {
		return info, true
	}
	return nil, false
}

func infoForIP(idx cache.Indexer, ip string) (*Info, bool) {
	objs, err := idx.ByIndex(IndexIP, ip)
	if err != nil {
		log.WithError(err).WithField("ip", ip).Debug("error accessing index. Ignoring")
		return nil, false
	}
	if len(objs) == 0 {
		return nil, false
	}
	return objs[0].(*Info), true
}

func (k *KubeData) getOwner(info *Info) Owner {
	if len(info.OwnerReferences) != 0 {
		ownerReference := info.OwnerReferences[0]
		if ownerReference.Kind != "ReplicaSet" {
			return Owner{
				Name: ownerReference.Name,
				Type: ownerReference.Kind,
			}
		}

		item, ok, err := k.replicaSets.GetIndexer().GetByKey(info.Namespace + "/" + ownerReference.Name)
		if err != nil {
			log.WithError(err).WithField("key", info.Namespace+"/"+ownerReference.Name).
				Debug("can't get ReplicaSet info from informer. Ignoring")
		} else if ok {
			rsInfo := item.(*metav1.ObjectMeta)
			if len(rsInfo.OwnerReferences) > 0 {
				return Owner{
					Name: rsInfo.OwnerReferences[0].Name,
					Type: rsInfo.OwnerReferences[0].Kind,
				}
			}
		}
	}
	// If no owner references found, return itself as owner
	return Owner{
		Name: info.Name,
		Type: info.Type,
	}
}

func (k *KubeData) initPodInformer(informerFactory informers.SharedInformerFactory) error {
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
		return fmt.Errorf("can't add %s indexer to Pods informer: %w", IndexIP, err)
	}

	k.pods = pods
	return nil
}

func (k *KubeData) initServiceInformer(informerFactory informers.SharedInformerFactory) error {
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
		return fmt.Errorf("can't add %s indexer to Pods informer: %w", IndexIP, err)
	}

	k.services = services
	return nil
}

func (k *KubeData) initReplicaSetInformer(informerFactory informers.SharedInformerFactory) error {
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

func (k *KubeData) InitFromConfig(kubeConfigPath string) error {
	// Initialization variables
	k.stopChan = make(chan struct{})

	config, err := LoadConfig(kubeConfigPath)
	if err != nil {
		return err
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	err = k.initInformers(kubeClient)
	if err != nil {
		return err
	}

	return nil
}

func LoadConfig(kubeConfigPath string) (*rest.Config, error) {
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

func (k *KubeData) initInformers(client kubernetes.Interface) error {
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

	log.Infof("Starting kubernetes informers, waiting for syncronization")
	informerFactory.Start(k.stopChan)
	informerFactory.WaitForCacheSync(k.stopChan)
	log.Infof("Kubernetes informers started")

	return nil
}
