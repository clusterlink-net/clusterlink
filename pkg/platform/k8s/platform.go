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

	logrusr "github.com/bombsimon/logrusr/v4"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/clusterlink-net/clusterlink/pkg/utils/netutils"
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

func (p *Platform) setExternalNameService(host, externalName string) *corev1.Service {
	eName := externalName
	if netutils.IsIP(eName) {
		eName += ".nip.io" // Convert IP to DNS address.
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: host, Namespace: p.namespace},
		Spec: corev1.ServiceSpec{
			Type:         corev1.ServiceTypeExternalName,
			ExternalName: eName,
		},
	}
}

func (p *Platform) setClusterIPService(host, targetApp string, port, targetPort uint16) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: host, Namespace: p.namespace},
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
}

// CreateService creates a service.
func (p *Platform) CreateService(name, host, targetApp string, port, targetPort uint16) {
	serviceSpec := p.setClusterIPService(host, targetApp, port, targetPort)
	p.logger.Infof("Creating K8s service at %s:%d.", host, port)
	go p.serviceReconciler.CreateResource(name, serviceSpec)
}

// UpdateService updates a service.
func (p *Platform) UpdateService(name, host, targetApp string, port, targetPort uint16) {
	serviceSpec := p.setClusterIPService(host, targetApp, port, targetPort)
	p.logger.Infof("Updating K8s service at %s:%d.", host, port)
	go p.serviceReconciler.UpdateResource(name, serviceSpec)
}

// CreateExternalService creates an external service.
func (p *Platform) CreateExternalService(name, host, externalName string) {
	serviceSpec := p.setExternalNameService(host, externalName)
	p.logger.Infof("Creating Kubernetes service %s of type ExternalName linked to %s.", host, externalName)
	go p.serviceReconciler.CreateResource(name, serviceSpec)
}

// UpdateExternalService updates an external service.
func (p *Platform) UpdateExternalService(name, host, externalName string) {
	serviceSpec := p.setExternalNameService(host, externalName)
	p.logger.Infof("Updating Kubernetes service %s of type ExternalName linked to %s.", host, externalName)
	go p.serviceReconciler.UpdateResource(name, serviceSpec)
}

// DeleteService deletes a service.
func (p *Platform) DeleteService(name, host string) {
	serviceSpec := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: host, Namespace: p.namespace}}

	p.logger.Infof("Deleting K8s service %s.", host)
	go p.serviceReconciler.DeleteResource(name, serviceSpec)
}

// GetLabelsFromIP return all the labels for specific ip.
func (p *Platform) GetLabelsFromIP(ip string) map[string]string {
	return p.podReconciler.getLabelsFromIP(ip)
}

// NewPlatform returns a new Kubernetes platform.
func NewPlatform() (*Platform, error) {
	logger := logrus.WithField("component", "platform.k8s")
	ctrl.SetLogger(logrusr.New(logrus.WithField("component", "k8s.controller-runtime")))

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
		client:            manager.GetClient(),
		podReconciler:     podReconciler,
		serviceReconciler: NewReconciler(manager.GetClient()),
		namespace:         namespace,
		logger:            logger,
	}, nil
}
