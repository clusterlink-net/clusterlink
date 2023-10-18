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
	"os"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	defaultNamespace = "default"
)

// Platform represents a k8s platform.
type Platform struct {
	endpointReconciler *Reconciler
	serviceReconciler  *Reconciler
	client             client.Client
	namespace          string
	logger             *logrus.Entry
}

// CreateService creates a service.
func (d *Platform) CreateService(name, targetApp string, port, targetPort uint16) {
	serviceSpec := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: d.namespace},
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
	d.logger.Infof("Creating K8s service at %s:%d.", name, port)
	go d.serviceReconciler.CreateResource(serviceSpec)
}

// DeleteService deletes a service.
func (d *Platform) DeleteService(name string) {
	serviceSpec := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: d.namespace}}

	d.logger.Infof("Deleting K8s service %s.", name)
	go d.serviceReconciler.DeleteResource(serviceSpec)
}

// UpdateService updates a service.
func (d *Platform) UpdateService(name, targetApp string, port, targetPort uint16) {
	serviceSpec := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: d.namespace},
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

	d.logger.Infof("Updating K8s service at %s:%d.", name, port)
	go d.serviceReconciler.UpdateResource(serviceSpec)

}

// CreateEndpoint creates a K8s endpoint.
func (d *Platform) CreateEndpoint(name, targetIP string, targetPort uint16) {
	endpointSpec := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: d.namespace,
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

	d.logger.Infof("Creating K8s endPoint at %s:%d that connected to external IP: %s:%d.", name, targetPort, targetIP, targetPort)
	go d.endpointReconciler.CreateResource(endpointSpec)

}

// UpdateEndpoint creates a K8s endpoint.
func (d *Platform) UpdateEndpoint(name, targetIP string, targetPort uint16) {
	endpointSpec := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: d.namespace,
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

	d.logger.Infof("Updating K8s endPoint at %s:%d to external host: %s:%d.", name, targetPort, targetIP, targetPort)
	go d.endpointReconciler.UpdateResource(endpointSpec)

}

// DeleteEndpoint deletes a k8s endpoint.
func (d *Platform) DeleteEndpoint(name string) {
	endpointSpec := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: d.namespace}}

	d.logger.Infof("Deleting K8s endPoint %s.", name)
	go d.endpointReconciler.DeleteResource(endpointSpec)

}

// NewPlatform returns a new Kubernetes platform.
func NewPlatform() (*Platform, error) {
	logger := logrus.WithField("component", "platform.k8s")
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	cl, err := client.New(cfg, client.Options{})
	if err != nil {
		return nil, err
	}

	// Get namespace
	namespace := os.Getenv("CL-NAMESPACE")
	if namespace == "" {
		namespace = defaultNamespace
		logger.Logger.Infoln("the CL-NAMESPACE environment variable is not set- use default namespace")
	}

	return &Platform{
		client:             cl,
		serviceReconciler:  NewReconciler(cl),
		endpointReconciler: NewReconciler(cl),
		namespace:          namespace,
		logger:             logger,
	}, nil
}
