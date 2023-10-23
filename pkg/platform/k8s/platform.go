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

package k8s

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	defaultNamespace = "default"
)

// Platform represents a k8s platform.
type Platform struct {
	podReconciler      *PodReconciler
	endpointReconciler *Reconciler
	serviceReconciler  *Reconciler
	client             client.Client
	namespace          string
	logger             *logrus.Entry
}

// CreateService creates a service.
func (p *Platform) CreateService(name, targetApp string, port, targetPort uint16) {
	serviceSpec := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: p.namespace},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					Port:       int32(port),
					TargetPort: intstr.FromInt(int(targetPort)),
				},
			},
			Type:     corev1.ServiceTypeClusterIP,
			Selector: map[string]string{"app": targetApp},
		},
	}
	p.logger.Infof("Creating K8s service at %s:%d.", name, port)
	go p.serviceReconciler.CreateResource(serviceSpec)
}

// DeleteService deletes a service.
func (p *Platform) DeleteService(name string) {
	serviceSpec := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: p.namespace}}

	p.logger.Infof("Deleting K8s service %s.", name)
	go p.serviceReconciler.DeleteResource(serviceSpec)
}

// UpdateService updates a service.
func (p *Platform) UpdateService(name, targetApp string, port, targetPort uint16) {
	serviceSpec := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: p.namespace},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					Port:       int32(port),
					TargetPort: intstr.FromInt(int(targetPort)),
				},
			},
			Type:     corev1.ServiceTypeClusterIP,
			Selector: map[string]string{"app": targetApp},
		},
	}

	p.logger.Infof("Updating K8s service at %s:%d.", name, port)
	go p.serviceReconciler.UpdateResource(serviceSpec)

}

// CreateEndpoint creates a K8s endpoint.
func (p *Platform) CreateEndpoint(name, targetIP string, targetPort uint16) {
	endpointSpec := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: p.namespace,
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP: targetIP, // Replace with the desired IP address of the endpoint.
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Port: int32(targetPort),
					},
				},
			},
		},
	}

	p.logger.Infof("Creating K8s endPoint at %s:%d that connected to external IP: %s:%d.", name, targetPort, targetIP, targetPort)
	go p.endpointReconciler.CreateResource(endpointSpec)

}

// UpdateEndpoint creates a K8s endpoint.
func (p *Platform) UpdateEndpoint(name, targetIP string, targetPort uint16) {
	endpointSpec := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: p.namespace,
		},
		Subsets: []corev1.EndpointSubset{
			{
				Addresses: []corev1.EndpointAddress{
					{
						IP: targetIP, // Replace with the desired IP address of the endpoint.
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Port: int32(targetPort),
					},
				},
			},
		},
	}

	p.logger.Infof("Updating K8s endPoint at %s:%d to external host: %s:%d.", name, targetPort, targetIP, targetPort)
	go p.endpointReconciler.UpdateResource(endpointSpec)

}

// DeleteEndpoint deletes a k8s endpoint.
func (p *Platform) DeleteEndpoint(name string) {
	endpointSpec := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: p.namespace}}

	p.logger.Infof("Deleting K8s endPoint %s.", name)
	go p.endpointReconciler.DeleteResource(endpointSpec)

}

// GetLabelsFromIP return all the labels for specific ip.
func (p *Platform) GetLabelsFromIP(ip string) map[string]string {
	return p.podReconciler.getLabelsFromIP(ip)
}

// NewPlatform returns a new Kubernetes platform.
func NewPlatform() (*Platform, error) {
	logger := logrus.WithField("component", "platform.k8s")
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	manager, err := ctrl.NewManager(cfg, ctrl.Options{})
	if err != nil {
		return nil, err
	}
	podReconciler, err := NewPodReconciler(&manager)
	if err != nil {
		return nil, err
	}

	err = ctrl.NewControllerManagedBy(manager).
		For(&corev1.Pod{}).
		Complete(podReconciler)
	if err != nil {
		return nil, err
	}

	// Start manger and all the controllers.
	go func() {
		if err := manager.Start(context.Background()); err != nil {
			logger.Error(err, "problem running manager")
		}
	}()

	// Get namespace
	namespace := os.Getenv("CL-NAMESPACE")
	if namespace == "" {
		namespace = defaultNamespace
		logger.Logger.Infoln("the CL-NAMESPACE environment variable is not set- use default namespace")
	}

	return &Platform{
		client:             manager.GetClient(),
		podReconciler:      podReconciler,
		serviceReconciler:  NewReconciler(manager.GetClient()),
		endpointReconciler: NewReconciler(manager.GetClient()),
		namespace:          namespace,
		logger:             logger,
	}, nil
}
