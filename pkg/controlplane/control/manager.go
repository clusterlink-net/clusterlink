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

package control

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	discv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cpapp "github.com/clusterlink-net/clusterlink/cmd/cl-dataplane/app"
	dpapp "github.com/clusterlink-net/clusterlink/cmd/cl-dataplane/app"
	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/util/tls"
)

// Manager is responsible for handling control operations,
// which needs to be coordinated across all dataplane/controlplane instances.
// This includes target port generation for imported services, as well as
// k8s service creation per imported service.
type Manager struct {
	peerManager

	client  client.Client
	crdMode bool
	ports   *portManager

	podLock sync.RWMutex
	// clDataplanePodInfo stores the IPv4/v6 address of cl-dataplane pod per namespace
	clDataplanePodInfo map[string]string
	logger             *logrus.Entry
}

func (m *Manager) addImportEndpointSlice(ctx context.Context, imp *v1alpha1.Import) error {
	m.logger.Infof("Adding import endpointslice '%s/%s'.", imp.Namespace, imp.Name)

	protocol := v1.ProtocolTCP
	var port32 int32
	newEndpointslice := discv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      imp.Name + "-" + uuid.NewString()[:5],
			Namespace: imp.Namespace,
			Labels: map[string]string{
				"kubernetes.io/service-name":             imp.Spec.Merge,
				"endpointslice.kubernetes.io/managed-by": cpapp.Name,
			},
		},
		AddressType: discv1.AddressTypeIPv4,
		Endpoints: []discv1.Endpoint{
			{
				Addresses: []string{m.clDataplanePodInfo[imp.Namespace]},
			},
		},
	}
	var oldEndpointslice discv1.EndpointSlice
	var create bool
	err := m.client.Get(
		ctx,
		types.NamespacedName{
			Name:      imp.Name,
			Namespace: imp.Namespace,
		},
		&oldEndpointslice)

	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		create = true
	}
	fullName := imp.Namespace + "/" + imp.Name
	port, err := m.ports.Lease(fullName, imp.Spec.TargetPort)
	if err != nil {
		return fmt.Errorf("cannot generate listening port: %w", err)
	}
	imp.Spec.TargetPort = port
	port32 = int32(port)
	endpointPort := discv1.EndpointPort{
		Port:     &port32,
		Protocol: &protocol,
	}
	if create {
		newEndpointslice.Ports = make([]discv1.EndpointPort, 1)
		newEndpointslice.Ports[0] = endpointPort
		err = m.client.Create(ctx, &newEndpointslice)
	} else {
		oldEndpointslice.Ports = append(oldEndpointslice.Ports, endpointPort)
		err = m.client.Update(ctx, &oldEndpointslice)
	}

	if err != nil && create {
		m.ports.Release(fullName)
	}

	return err
}

func (m *Manager) addImportService(ctx context.Context, imp *v1alpha1.Import) error {
	m.logger.Infof("Adding import service '%s/%s'.", imp.Namespace, imp.Name)

	newService := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      imp.Name,
			Namespace: imp.Namespace,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Protocol:   v1.ProtocolTCP,
					Port:       int32(imp.Spec.Port),
					TargetPort: intstr.FromInt32(int32(imp.Spec.TargetPort)),
				},
			},
			Type:     v1.ServiceTypeClusterIP,
			Selector: map[string]string{"app": dpapp.Name},
		},
	}

	var oldService v1.Service
	var create bool
	err := m.client.Get(
		ctx,
		types.NamespacedName{
			Name:      imp.Name,
			Namespace: imp.Namespace,
		},
		&oldService)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		create = true
	}

	// if service exists, and import specifies a random (0) target port,
	// then use existing service target port instead of allocating a new port
	if !create && len(oldService.Spec.Ports) == 1 && imp.Spec.TargetPort == 0 {
		imp.Spec.TargetPort = uint16(oldService.Spec.Ports[0].TargetPort.IntVal)
		newService.Spec.Ports[0].TargetPort = intstr.FromInt32(int32(imp.Spec.TargetPort))
	}

	newPort := imp.Spec.TargetPort == 0

	fullName := imp.Namespace + "/" + imp.Name
	port, err := m.ports.Lease(fullName, imp.Spec.TargetPort)
	if err != nil {
		return fmt.Errorf("cannot generate listening port: %w", err)
	}

	if newPort {
		imp.Spec.TargetPort = port
		newService.Spec.Ports[0].TargetPort = intstr.FromInt32(int32(port))
		if m.crdMode {
			if err := m.client.Update(ctx, imp); err != nil {
				m.ports.Release(fullName)
				return err
			}
		}
	}

	if create {
		err = m.client.Create(ctx, newService)
	} else if serviceChanged(&oldService, newService) {
		err = m.client.Update(ctx, newService)
	}

	if err != nil && newPort {
		m.ports.Release(fullName)
	}

	return err
}

// deleteClDataplane deletes pod to ipToPod list.
func (m *Manager) deleteClDataplane(podID types.NamespacedName) {
	if strings.Contains(podID.Name, dpapp.Name) {
		m.logger.Infof("Detected cl-dataplane(%s) pod delete in namespace: %s", podID.Name, podID.Namespace)
	}
}

// addPod adds or updates pod to ipToPod and podList.
func (m *Manager) addClDataplane(pod *v1.Pod) {
	m.podLock.Lock()
	defer m.podLock.Unlock()

	if pod.Labels["app"] == dpapp.Name {
		m.logger.Infof("Detected cl-dataplane(%s) pod add/update in namespace: %s, with IP %s", pod.Name, pod.Namespace, pod.Status.PodIP)
		m.clDataplanePodInfo[pod.Namespace] = pod.Status.PodIP
	}
}

// AddImport adds a listening socket for an imported remote service.
func (m *Manager) AddImport(ctx context.Context, imp *v1alpha1.Import) error {
	if imp.Spec.Merge != "" {
		return m.addImportEndpointSlice(ctx, imp)
	}
	return m.addImportService(ctx, imp)
}

// DeleteImport removes the listening socket of a previously imported service.
func (m *Manager) DeleteImport(ctx context.Context, name types.NamespacedName) error {
	m.logger.Infof("Deleting import '%s/%s'.", name.Namespace, name.Name)

	m.ports.Release(name.Namespace + "/" + name.Name)
	return m.client.Delete(
		ctx,
		&v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name.Name,
				Namespace: name.Namespace,
			},
		})
}

func serviceChanged(svc1, svc2 *v1.Service) bool {
	if svc1.Spec.Type != svc2.Spec.Type {
		return true
	}

	if len(svc1.Spec.Ports) != len(svc2.Spec.Ports) {
		return true
	}

	for i := 0; i < len(svc1.Spec.Ports); i++ {
		if svc1.Spec.Ports[i].Protocol != svc2.Spec.Ports[i].Protocol {
			return true
		}

		if svc1.Spec.Ports[i].Port != svc2.Spec.Ports[i].Port {
			return true
		}

		if svc1.Spec.Ports[i].TargetPort != svc2.Spec.Ports[i].TargetPort {
			return true
		}
	}

	return false
}

// NewManager returns a new control manager.
func NewManager(cl client.Client, peerTLS *tls.ParsedCertData, crdMode bool) *Manager {
	logger := logrus.WithField("component", "controlplane.control.manager")

	return &Manager{
		peerManager:        newPeerManager(cl, peerTLS),
		client:             cl,
		crdMode:            crdMode,
		ports:              newPortManager(),
		clDataplanePodInfo: make(map[string]string),
		logger:             logger,
	}
}
