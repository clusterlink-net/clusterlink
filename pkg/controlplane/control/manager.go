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
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	dpapp "github.com/clusterlink-net/clusterlink/cmd/cl-dataplane/app"
	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/util/tls"
)

type exportServiceNotExistError struct {
	name types.NamespacedName
}

func (e exportServiceNotExistError) Error() string {
	return fmt.Sprintf(
		"service '%s/%s' does not exist",
		e.name.Namespace, e.name.Name)
}

// Manager is responsible for handling control operations,
// which needs to be coordinated across all dataplane/controlplane instances.
// This includes target port generation for imported services, as well as
// k8s service creation per imported service.
type Manager struct {
	peerManager

	client  client.Client
	crdMode bool
	ports   *portManager

	logger *logrus.Entry
}

// AddImport adds a listening socket for an imported remote service.
func (m *Manager) AddImport(ctx context.Context, imp *v1alpha1.Import) error {
	m.logger.Infof("Adding import '%s/%s'.", imp.Namespace, imp.Name)

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
		if !k8serrors.IsNotFound(err) {
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

// addExport defines a new route target for ingress dataplane connections.
func (m *Manager) addExport(ctx context.Context, export *v1alpha1.Export) (err error) {
	m.logger.Infof("Adding export '%s/%s'.", export.Namespace, export.Name)

	defer func() {
		validCond := &metav1.Condition{
			Type:   v1alpha1.ExportValid,
			Status: metav1.ConditionTrue,
			Reason: "Verified",
		}

		if err != nil {
			validCond.Status = metav1.ConditionFalse
			validCond.Reason = "Error"
			validCond.Message = err.Error()
		}

		conditions := &export.Status.Conditions
		if conditionChanged(conditions, validCond) {
			meta.SetStatusCondition(conditions, *validCond)

			m.logger.Infof(
				"Updating export '%s/%s' status: %v.",
				export.Namespace, export.Name, *conditions)
			statusError := m.client.Status().Update(ctx, export)
			if statusError != nil {
				if err == nil {
					err = statusError
					return
				}

				m.logger.Warnf("Error updating export status: %v.", statusError)
				return
			}
		}

		if errors.Is(err, &exportServiceNotExistError{}) {
			err = reconcile.TerminalError(err)
		}
	}()

	if export.Spec.Host != "" {
		return nil
	}

	name := types.NamespacedName{
		Name:      export.Name,
		Namespace: export.Namespace,
	}

	if err := m.client.Get(ctx, name, &v1.Service{}); err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}

		return exportServiceNotExistError{name}
	}

	return nil
}

// addService adds a new service.
func (m *Manager) addService(ctx context.Context, service *v1.Service) error {
	return m.checkExportService(ctx, types.NamespacedName{
		Namespace: service.Namespace,
		Name:      service.Name,
	})
}

// deleteService deletes a service.
func (m *Manager) deleteService(ctx context.Context, name types.NamespacedName) error {
	return m.checkExportService(ctx, name)
}

func (m *Manager) checkExportService(ctx context.Context, name types.NamespacedName) error {
	var export v1alpha1.Export
	if err := m.client.Get(ctx, name, &export); err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}

		return nil
	}

	return m.addExport(ctx, &export)
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

func conditionChanged(conditions *[]metav1.Condition, cond *metav1.Condition) bool {
	oldCond := meta.FindStatusCondition(*conditions, cond.Type)
	if oldCond == nil {
		return true
	}

	if oldCond.Status != cond.Status {
		return true
	}

	if oldCond.Reason != cond.Reason {
		return true
	}

	return oldCond.Message != cond.Message
}

// NewManager returns a new control manager.
func NewManager(cl client.Client, peerTLS *tls.ParsedCertData, crdMode bool) *Manager {
	logger := logrus.WithField("component", "controlplane.control.manager")

	return &Manager{
		peerManager: newPeerManager(cl, peerTLS),
		client:      cl,
		crdMode:     crdMode,
		ports:       newPortManager(),
		logger:      logger,
	}
}
