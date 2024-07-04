// Copyright (c) The ClusterLink Authors.
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

package util

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/klient"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/support"
	"sigs.k8s.io/e2e-framework/support/kind"

	clusterlink "github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/services"
)

const (
	// ExportedLogsPath is the path where test logs will be exported.
	ExportedLogsPath = "/tmp/clusterlink-k8s-tests"
)

// PodFailedError represents a pod that ran and returned a failure.
type PodFailedError struct{}

func (e PodFailedError) Error() string {
	return "pod failed"
}

// Service represents a kubernetes service.
type Service struct {
	// Name is the service name.
	Name string
	// Namespace is the service namespace.
	Namespace string
	// Port is the service external listening port.
	Port uint16
	// A key-value map to organize and categorize (scope and select) services.
	Labels map[string]string
}

// PodAndService represents a kubernetes service and a backing pod.
type PodAndService struct {
	Service

	// Image is the container image.
	Image string
	// Args are the container command line arguments.
	Args []string
}

// Pod represents a kubernetes pod.
type Pod struct {
	// Name is the pod name.
	Name string
	// Namespace is the pod namespace.
	Namespace string
	// Image is the container image.
	Image string
	// Command is the command to execute on the container
	Command []string
	// Args are the container command line arguments.
	Args []string
}

// KindCluster represents a kind kubernetes cluster.
type KindCluster struct {
	AsyncRunner

	created          sync.WaitGroup
	name             string
	ip               string
	cluster          support.E2EClusterProviderWithImageLoader
	resources        *resources.Resources
	clientset        *kubernetes.Clientset
	nodeportServices map[string]*map[string]*v1.Service // map[namespace][name]
}

// Name returns the cluster name.
func (c *KindCluster) Name() string {
	return c.name
}

// IP returns the cluster IP.
func (c *KindCluster) IP() string {
	return c.ip
}

// Resources returns the cluster resources.
func (c *KindCluster) Resources() *resources.Resources {
	return c.resources
}

// Start the kind cluster.
// Should be called only once.
func (c *KindCluster) Start() {
	c.created.Add(1)
	c.Run(func() error {
		// retry cluster creation up to 10 times
		var err error
		for i := 0; i < 10; i++ {
			//nolint:errcheck // ignore errors when cluster did not exist
			_ = c.cluster.Destroy(context.Background())

			_, err = c.cluster.Create(context.Background())
			if err == nil {
				break
			}
		}

		c.created.Done()
		if err != nil {
			return fmt.Errorf("unable to create kind cluster: %w", err)
		}

		// wait for controlplane to be ready and initialize clients
		if err := c.initializeClients(); err != nil {
			return err
		}

		// get cluster external-facing IP
		if err := c.initClusterIP(); err != nil {
			return fmt.Errorf("unable to get cluster IP: %w", err)
		}

		return nil
	})
}

func (c *KindCluster) initializeClients() error {
	client, err := klient.New(c.cluster.KubernetesRestConfig())
	if err != nil {
		return fmt.Errorf("error initializing API server client: %w", err)
	}

	if err := c.cluster.WaitForControlPlane(context.Background(), client); err != nil {
		return fmt.Errorf("error waiting for controlplane to be ready: %w", err)
	}

	cfg := c.cluster.KubernetesRestConfig()

	c.resources, err = resources.New(cfg)
	if err != nil {
		return fmt.Errorf("unable to initialize REST client: %w", err)
	}

	c.clientset, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("unable to initialize k8s clientset: %w", err)
	}

	// Add instance CRD to scheme.
	if err := clusterlink.AddToScheme(c.resources.GetScheme()); err != nil {
		return fmt.Errorf("unable to add clusterlink CRD: %w", err)
	}

	return nil
}

func (c *KindCluster) initClusterIP() error {
	var nodes v1.NodeList
	if err := c.resources.List(context.Background(), &nodes); err != nil {
		return fmt.Errorf("unable to list cluster nodes: %w", err)
	}

	if len(nodes.Items) != 1 {
		return fmt.Errorf("expected a single node, but got: %v", nodes)
	}

	addresses := nodes.Items[0].Status.Addresses
	for _, addr := range addresses {
		if addr.Type == v1.NodeInternalIP {
			if c.ip != "" {
				return fmt.Errorf("expected a single node IP, but got: %v", addresses)
			}

			c.ip = addr.Address
			if c.ip == "" {
				return fmt.Errorf("got empty node IP address: %v", addr)
			}
		}
	}

	if c.ip == "" {
		return fmt.Errorf("could not get node IP: %v", addresses)
	}

	return nil
}

// LoadImage loads a docker image to the cluster.
// Assumes Start was already called.
func (c *KindCluster) LoadImage(name string) {
	c.Run(func() error {
		c.created.Wait()
		if err := c.Error(); err != nil {
			return err
		}

		if err := c.cluster.LoadImage(context.Background(), name); err != nil {
			return fmt.Errorf("error loading image %s: %w", name, err)
		}

		return nil
	})
}

// ExportLogs exports cluster logs to files.
func (c *KindCluster) ExportLogs() error {
	if err := c.cluster.ExportLogs(context.Background(), ExportedLogsPath); err != nil {
		return fmt.Errorf("cannot export cluster logs: %w", err)
	}

	return nil
}

// Destroy cluster.
func (c *KindCluster) Destroy() error {
	if err := c.cluster.Destroy(context.Background()); err != nil {
		return fmt.Errorf("cannot destroy cluster: %w", err)
	}

	return nil
}

// CreatePodAndService creates a pod exposed by a service.
func (c *KindCluster) CreatePodAndService(podAndService *PodAndService) error {
	err := c.resources.Create(
		context.Background(),
		&v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podAndService.Name,
				Namespace: podAndService.Namespace,
				Labels:    map[string]string{"app": podAndService.Name},
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{{
					Name:  podAndService.Name,
					Image: podAndService.Image,
					Args:  podAndService.Args,
				}},
			},
		})
	if err != nil {
		return err
	}

	return c.resources.Create(
		context.Background(),
		&v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podAndService.Name,
				Namespace: podAndService.Namespace,
			},
			Spec: v1.ServiceSpec{
				Ports: []v1.ServicePort{{
					Port:       int32(podAndService.Port),
					TargetPort: intstr.FromInt32(int32(podAndService.Port)),
				}},
				Selector: map[string]string{"app": podAndService.Name},
			},
		})
}

// RunPod runs a pod, wait for its completion, and return its standard output.
func (c *KindCluster) RunPod(podSpec *Pod) (string, error) {
	// create pod
	err := c.resources.Create(
		context.Background(),
		&v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podSpec.Name,
				Namespace: podSpec.Namespace,
				Labels:    map[string]string{"app": podSpec.Name},
			},
			Spec: v1.PodSpec{
				RestartPolicy: v1.RestartPolicyNever,
				Containers: []v1.Container{{
					Name:    podSpec.Name,
					Image:   podSpec.Image,
					Command: podSpec.Command,
					Args:    podSpec.Args,
				}},
			},
		})
	if err != nil {
		return "", fmt.Errorf("cannot create pod: %w", err)
	}

	// defer pod deletion
	defer func() {
		err = c.resources.Delete(context.Background(), &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podSpec.Name,
				Namespace: podSpec.Namespace,
			},
		})
	}()

	// wait for pod status
	var pod v1.Pod
	for t := time.Now(); time.Since(t) < time.Second*30; time.Sleep(time.Millisecond * 100) {
		err = c.resources.Get(context.Background(), podSpec.Name, podSpec.Namespace, &pod)
		if err != nil {
			continue
		}

		switch pod.Status.Phase {
		case v1.PodPending:
			continue
		case v1.PodRunning:
			continue
		}

		break
	}

	if err != nil {
		return "", fmt.Errorf("cannot get pod status: %w", err)
	}
	if pod.Status.Phase != v1.PodSucceeded {
		if pod.Status.Phase == v1.PodFailed {
			return "", &PodFailedError{}
		}
		return "", fmt.Errorf("pod did not succeed: %s", pod.Status.Phase)
	}

	// get pod logs
	logReader, err := c.clientset.CoreV1().Pods(podSpec.Namespace).
		GetLogs(podSpec.Name, &v1.PodLogOptions{}).
		Stream(context.Background())
	if err != nil {
		return "", fmt.Errorf("cannot get pod logs: %w", err)
	}

	body, err := io.ReadAll(logReader)
	if err != nil {
		return "", fmt.Errorf("cannot read pod logs: %w", err)
	}

	if err := logReader.Close(); err != nil {
		return "", fmt.Errorf("cannot close pod logs: %w", err)
	}

	return string(body), err
}

// CreateNamespace creates a namespace.
func (c *KindCluster) CreateNamespace(name string) error {
	return c.resources.Create(
		context.Background(),
		&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}})
}

// DeleteNamespace deletes a namespace.
func (c *KindCluster) DeleteNamespace(name string) error {
	delete(c.nodeportServices, name)
	return c.resources.Delete(
		context.Background(),
		&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}})
}

// CreateFromYAML creates k8s objects from a yaml string, in a given namespace.
func (c *KindCluster) CreateFromYAML(yaml, namespace string) error {
	return decoder.DecodeEach(context.Background(),
		strings.NewReader(yaml),
		decoder.CreateHandler(c.resources),
		decoder.MutateNamespace(namespace))
}

// CreateFromPath creates k8s objects from a yaml in a folder.
func (c *KindCluster) CreateFromPath(folder string) error {
	return decoder.DecodeEachFile(context.Background(), os.DirFS(folder), "*", decoder.CreateHandler(c.resources))
}

// ExposeNodeport returns a nodeport (uint16) for accessing a given k8s service.
// The returned nodeport service is cached across subsequent calls.
func (c *KindCluster) ExposeNodeport(service *Service) (uint16, error) {
	var k8sService v1.Service
	err := c.resources.Get(context.Background(), service.Name, service.Namespace, &k8sService)
	if err != nil {
		return 0, fmt.Errorf("error getting service: %w", err)
	}

	if k8sService.Spec.Type == v1.ServiceTypeExternalName {
		splitted := strings.SplitN(k8sService.Spec.ExternalName, ".", 3)
		if len(splitted) != 3 || splitted[2] != "svc.cluster.local" {
			return 0, fmt.Errorf(
				"error parsing external name service name '%s'",
				k8sService.Spec.ExternalName)
		}

		err = c.resources.Get(context.Background(), splitted[0], splitted[1], &k8sService)
		if err != nil {
			return 0, fmt.Errorf("error getting backing service: %w", err)
		}
	}

	if int32(service.Port) != k8sService.Spec.Ports[0].Port {
		return 0, &services.ConnectionRefusedError{}
	}

	if len(k8sService.Spec.Ports) != 1 {
		return 0, fmt.Errorf("expected a single port service, but got: %v", k8sService.Spec.Ports)
	}

	svcs := c.nodeportServices[service.Namespace]
	if svcs == nil {
		svcs = &map[string]*v1.Service{}
		c.nodeportServices[service.Namespace] = svcs
	}

	nodeportService := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nodeport-" + service.Name,
			Namespace: k8sService.Namespace,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{{
				Port:       k8sService.Spec.Ports[0].Port,
				TargetPort: k8sService.Spec.Ports[0].TargetPort,
			}},
			Selector: k8sService.Spec.Selector,
			Type:     v1.ServiceTypeNodePort,
		},
	}

	cachedService := (*svcs)[service.Name]
	cacheMiss := true
	switch {
	case cachedService == nil:
		err := c.resources.Create(context.Background(), nodeportService)
		if err != nil {
			return 0, fmt.Errorf("cannot create nodeport service: %w", err)
		}
	case !reflect.DeepEqual(cachedService.Spec.Selector, k8sService.Spec.Selector) ||
		cachedService.Spec.Ports[0].Port != k8sService.Spec.Ports[0].Port ||
		cachedService.Spec.Ports[0].TargetPort != k8sService.Spec.Ports[0].TargetPort:

		err := c.resources.Update(context.Background(), nodeportService)
		if err != nil {
			return 0, fmt.Errorf("cannot update nodeport service: %w", err)
		}
	default:
		cacheMiss = false
	}

	if cacheMiss {
		err := c.resources.Get(
			context.Background(), nodeportService.Name, nodeportService.Namespace, nodeportService)
		if err != nil {
			return 0, fmt.Errorf("error getting service: %w", err)
		}

		cachedService = nodeportService
		(*svcs)[service.Name] = cachedService
	}

	return uint16(cachedService.Spec.Ports[0].NodePort), nil
}

// ScaleDeployment updates the number of deployment replicas, and waits for this update to complete.
func (c *KindCluster) ScaleDeployment(name, namespace string, replicas int32) error {
	// export logs since pods may be terminated after scale
	if err := c.ExportLogs(); err != nil {
		return fmt.Errorf("unable to export logs: %w", err)
	}

	scale, err := c.clientset.AppsV1().Deployments(namespace).GetScale(
		context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("unable to get deployment scale: %w", err)
	}

	scale.Spec.Replicas = replicas

	_, err = c.clientset.AppsV1().Deployments(namespace).UpdateScale(
		context.Background(), name, scale, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("unable to update deployment scale: %w", err)
	}

	// wait for scale to update
	for t := time.Now(); time.Since(t) < time.Second*60; time.Sleep(time.Millisecond * 500) {
		scale, err = c.clientset.AppsV1().Deployments(namespace).GetScale(
			context.Background(), name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("unable to get deployment scale: %w", err)
		}

		if scale.Status.Replicas == replicas {
			return nil
		}
	}

	return fmt.Errorf("timeout while waiting for deployment scale to update")
}

// StatusObject represents a k8s object with status conditions.
type StatusObject interface {
	k8s.Object
	GetStatusConditions() []metav1.Condition
}

// WaitFor waits for a condition to be set on an object.
func (c *KindCluster) WaitFor(
	obj k8s.Object,
	statusConditions *[]metav1.Condition,
	conditionType string,
	expectedConditionStatus bool,
) error {
	return wait.For(func(ctx context.Context) (bool, error) {
		err := c.resources.Get(ctx, obj.GetName(), obj.GetNamespace(), obj)
		if err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}

			return false, err
		}

		cond := meta.FindStatusCondition(*statusConditions, conditionType)
		if cond == nil {
			return false, nil
		}

		switch cond.Status {
		case metav1.ConditionFalse:
			return !expectedConditionStatus, nil
		case metav1.ConditionTrue:
			return expectedConditionStatus, nil
		default:
			return false, fmt.Errorf("unexpected condition status: %v", cond.Status)
		}
	}, wait.WithTimeout(time.Second*60))
}

// WaitFor waits for a condition to be set on an object.
func (c *KindCluster) WaitForDeletion(obj k8s.Object) error {
	return wait.For(func(ctx context.Context) (bool, error) {
		err := c.resources.Get(ctx, obj.GetName(), obj.GetNamespace(), obj)
		if err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}

			return false, err
		}

		return false, nil
	})
}

// NewKindCluster returns a new yet to be running kind cluster.
func NewKindCluster(name string) *KindCluster {
	return &KindCluster{
		cluster:          kind.NewCluster(name).WithVersion("v0.22.0").(support.E2EClusterProviderWithImageLoader),
		name:             name,
		nodeportServices: make(map[string]*map[string]*v1.Service),
	}
}
