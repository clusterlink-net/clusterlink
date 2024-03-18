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
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"

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

const (
	appName              = "clusterlink.net"
	labelManagedBy       = "app.kubernetes.io/managed-by"
	labelImportName      = "clusterlink.net/import-name"
	labelImportNamespace = "clusterlink.net/import-namespace"
)

type exportServiceNotExistError struct {
	name types.NamespacedName
}

func (e exportServiceNotExistError) Error() string {
	return fmt.Sprintf(
		"service '%s/%s' does not exist",
		e.name.Namespace, e.name.Name)
}

type conflictingServiceError struct {
	name      types.NamespacedName
	managedBy string
}

func (e conflictingServiceError) Error() string {
	return fmt.Sprintf(
		"service '%s/%s' already exists and managed by '%s'",
		e.name.Namespace, e.name.Name, e.managedBy)
}

// Manager is responsible for handling control operations,
// which needs to be coordinated across all dataplane/controlplane instances.
// This includes target port generation for imported services, as well as
// k8s service creation per imported service.
type Manager struct {
	peerManager

	client    client.Client
	namespace string
	crdMode   bool
	ports     *portManager

	lock            sync.Mutex
	serviceToImport map[string]types.NamespacedName

	logger *logrus.Entry
}

// AddImport adds a listening socket for an imported remote service.
func (m *Manager) AddImport(ctx context.Context, imp *v1alpha1.Import) (err error) {
	m.logger.Infof("Adding import '%s/%s'.", imp.Namespace, imp.Name)

	targetPortValidCond := &metav1.Condition{
		Type:   v1alpha1.ImportTargetPortValid,
		Status: metav1.ConditionFalse,
	}

	defer func() {
		if !m.crdMode {
			return
		}

		serviceCreatedCond := &metav1.Condition{
			Type:   v1alpha1.ImportServiceCreated,
			Status: metav1.ConditionTrue,
			Reason: "Created",
		}

		if err != nil {
			serviceCreatedCond.Status = metav1.ConditionFalse
			serviceCreatedCond.Reason = "Error"
			serviceCreatedCond.Message = err.Error()
		}

		conditions := &imp.Status.Conditions
		if conditionChanged(conditions, serviceCreatedCond) || conditionChanged(conditions, targetPortValidCond) {
			meta.SetStatusCondition(conditions, *targetPortValidCond)
			meta.SetStatusCondition(conditions, *serviceCreatedCond)

			m.logger.Infof("Updating import '%s/%s' status: %v.", imp.Namespace, imp.Name, *conditions)
			statusError := m.client.Status().Update(ctx, imp)
			if statusError != nil {
				if err == nil {
					err = statusError
					return
				}

				m.logger.Warnf("Error updating import status: %v.", statusError)
				return
			}
		}

		if errors.Is(err, &conflictingServiceError{}) || errors.Is(err, &conflictingTargetPortError{}) {
			err = reconcile.TerminalError(err)
		}
	}()

	err = m.allocateTargetPort(ctx, imp)
	if err != nil {
		targetPortValidCond.Reason = "Error"
		targetPortValidCond.Message = err.Error()
		return err
	}

	targetPortValidCond.Status = metav1.ConditionTrue
	targetPortValidCond.Reason = "Leased"

	serviceName := imp.Name
	if imp.Namespace != m.namespace {
		serviceName = systemServiceName(types.NamespacedName{
			Namespace: imp.Namespace,
			Name:      imp.Name,
		})
	}

	systemService := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: m.namespace,
			Labels:    make(map[string]string),
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Protocol:   v1.ProtocolTCP,
					Port:       int32(imp.Spec.Port),
					TargetPort: intstr.FromInt32(int32(imp.Spec.TargetPort)),
				},
			},
			Selector: map[string]string{"app": dpapp.Name},
			Type:     v1.ServiceTypeClusterIP,
		},
	}

	if err := m.addImportService(ctx, imp, systemService); err != nil {
		return err
	}

	if imp.Namespace == m.namespace {
		return nil
	}

	userService := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      imp.Name,
			Namespace: imp.Namespace,
		},
		Spec: v1.ServiceSpec{
			ExternalName: fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, m.namespace),
			Type:         v1.ServiceTypeExternalName,
		},
	}

	return m.addImportService(ctx, imp, userService)
}

// DeleteImport removes the listening socket of a previously imported service.
func (m *Manager) DeleteImport(ctx context.Context, name types.NamespacedName) error {
	m.logger.Infof("Deleting import '%s/%s'.", name.Namespace, name.Name)

	// delete user service
	errs := make([]error, 2)
	errs[0] = m.deleteImportService(ctx, name)

	if name.Namespace != m.namespace {
		// delete system service
		errs[1] = m.deleteImportService(ctx, types.NamespacedName{
			Namespace: m.namespace,
			Name:      systemServiceName(name),
		})
	}

	m.ports.Release(name)

	err := errors.Join(errs...)
	if err != nil && m.crdMode {
		// if all errors are conflictingServiceError, mark as TerminalError
		// so that reconciler will not retry
		for _, err2 := range errs {
			if err2 != nil && !errors.Is(err2, &conflictingServiceError{}) {
				return err
			}
		}

		err = reconcile.TerminalError(err)
	}

	return err
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
	err := m.checkImportService(ctx, types.NamespacedName{
		Namespace: service.Namespace,
		Name:      service.Name,
	})
	if err != nil {
		return err
	}

	return m.checkExportService(ctx, types.NamespacedName{
		Namespace: service.Namespace,
		Name:      service.Name,
	})
}

// deleteService deletes a service.
func (m *Manager) deleteService(ctx context.Context, name types.NamespacedName) error {
	if err := m.checkImportService(ctx, name); err != nil {
		return err
	}
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

func (m *Manager) checkImportService(ctx context.Context, name types.NamespacedName) error {
	var imp v1alpha1.Import
	if err := m.client.Get(ctx, name, &imp); err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
	} else {
		if err := m.AddImport(ctx, &imp); err != nil {
			return err
		}
	}

	if name.Namespace != m.namespace {
		return nil
	}

	m.lock.Lock()
	name, ok := m.serviceToImport[name.Name]
	m.lock.Unlock()

	if !ok {
		return nil
	}

	if err := m.client.Get(ctx, name, &imp); err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}
	} else {
		return m.AddImport(ctx, &imp)
	}

	return nil
}

func (m *Manager) allocateTargetPort(ctx context.Context, imp *v1alpha1.Import) error {
	name := types.NamespacedName{
		Namespace: imp.Namespace,
		Name:      imp.Name,
	}

	leasedPort, err := m.ports.Lease(name, imp.Spec.TargetPort)
	if err != nil {
		return fmt.Errorf("cannot generate listening port: %w", err)
	}

	if imp.Spec.TargetPort == 0 {
		imp.Spec.TargetPort = leasedPort

		if m.crdMode {
			m.logger.Infof("Updating target port for import %v.", name)
			if err := m.client.Update(ctx, imp); err != nil {
				m.ports.Release(name)
				return err
			}
		}
	}

	return nil
}

func (m *Manager) addImportService(ctx context.Context, imp *v1alpha1.Import, service *v1.Service) error {
	service.Labels[labelManagedBy] = appName
	service.Labels[labelImportName] = imp.Name
	service.Labels[labelImportNamespace] = imp.Namespace

	if imp.Namespace != service.Namespace {
		m.lock.Lock()
		m.serviceToImport[service.Name] = types.NamespacedName{
			Namespace: imp.Namespace,
			Name:      imp.Namespace,
		}
		m.lock.Unlock()
	}

	var oldService v1.Service
	err := m.client.Get(
		ctx,
		types.NamespacedName{
			Name:      service.Name,
			Namespace: service.Namespace,
		},
		&oldService)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}

		m.logger.Infof("Creating service: %s/%s.", service.Namespace, service.Name)
		return m.client.Create(ctx, service)
	}

	if err := checkServiceLabels(&oldService, types.NamespacedName{
		Namespace: imp.Namespace,
		Name:      imp.Name,
	}); err != nil {
		return err
	}

	if !serviceChanged(&oldService, service) {
		// service already exists as expected
		return nil
	}

	m.logger.Infof("Updating service: %s/%s.", service.Namespace, service.Name)
	return m.client.Update(ctx, service)
}

func (m *Manager) deleteImportService(ctx context.Context, name types.NamespacedName) error {
	var oldService v1.Service
	err := m.client.Get(ctx, name, &oldService)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}

		return nil
	}

	if err := checkServiceLabels(&oldService, name); err != nil {
		return err
	}

	m.logger.Infof("Deleting service: %v.", name)
	err = m.client.Delete(ctx, &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.Name,
			Namespace: name.Namespace,
		},
	})
	if err != nil {
		return err
	}

	if oldService.Labels[labelImportNamespace] != name.Namespace {
		m.lock.Lock()
		delete(m.serviceToImport, name.Name)
		m.lock.Unlock()
	}

	return nil
}

func checkServiceLabels(service *v1.Service, importName types.NamespacedName) error {
	serviceName := types.NamespacedName{
		Namespace: service.Namespace,
		Name:      service.Name,
	}

	var managedBy string
	var ok bool
	if managedBy, ok = service.Labels[labelManagedBy]; !ok || managedBy != appName {
		return conflictingServiceError{
			name:      serviceName,
			managedBy: managedBy,
		}
	}

	if name, ok := service.Labels[labelImportName]; !ok || name != importName.Name {
		return conflictingServiceError{
			name:      serviceName,
			managedBy: managedBy,
		}
	}

	if namespace, ok := service.Labels[labelImportNamespace]; !ok || namespace != importName.Namespace {
		return conflictingServiceError{
			name:      serviceName,
			managedBy: managedBy,
		}
	}

	return nil
}

func systemServiceName(name types.NamespacedName) string {
	return "import-" + hex.EncodeToString(sha256.New().Sum([]byte(name.Namespace+"/"+name.Name)))
}

func serviceChanged(svc1, svc2 *v1.Service) bool {
	if svc1.Spec.Type != svc2.Spec.Type {
		return true
	}

	if svc1.Spec.ExternalName != svc2.Spec.ExternalName {
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
func NewManager(cl client.Client, peerTLS *tls.ParsedCertData, namespace string, crdMode bool) *Manager {
	logger := logrus.WithField("component", "controlplane.control.manager")

	return &Manager{
		peerManager:     newPeerManager(cl, peerTLS),
		client:          cl,
		namespace:       namespace,
		crdMode:         crdMode,
		ports:           newPortManager(),
		serviceToImport: make(map[string]types.NamespacedName),
		logger:          logger,
	}
}
