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

package control

import (
	"context"
	"reflect"
	"strconv"
	"strings"

	//nolint:gosec // G505: use of weak cryptographic primitive is fine for service name
	"crypto/md5"
	"errors"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	discv1 "k8s.io/api/discovery/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sstrings "k8s.io/utils/strings"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	dpapp "github.com/clusterlink-net/clusterlink/cmd/cl-dataplane/app"
	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
)

const (
	AppName = "clusterlink.net"

	// service labels.
	LabelManagedBy       = "app.kubernetes.io/managed-by"
	LabelImportName      = "clusterlink.net/import-name"
	LabelImportNamespace = "clusterlink.net/import-namespace"

	// endpoint slice labels.
	LabelDPEndpointSliceName = "clusterlink.net/dataplane-endpointslice-name"
)

type exportServiceNotExistError struct {
	name types.NamespacedName
}

func (e exportServiceNotExistError) Error() string {
	return fmt.Sprintf(
		"export service '%s/%s' does not exist",
		e.name.Namespace, e.name.Name)
}

func (e exportServiceNotExistError) Is(target error) bool {
	_, ok := target.(*exportServiceNotExistError)
	return ok
}

type importServiceNotExistError struct {
	name types.NamespacedName
}

func (e importServiceNotExistError) Error() string {
	return fmt.Sprintf(
		"import service '%s/%s' does not exist",
		e.name.Namespace, e.name.Name)
}

func (e importServiceNotExistError) Is(target error) bool {
	_, ok := target.(*importServiceNotExistError)
	return ok
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

func (e conflictingServiceError) Is(target error) bool {
	_, ok := target.(*conflictingServiceError)
	return ok
}

type importEndpointSliceName struct {
	importName                 string
	dataplaneEndpointSliceName string
}

func (n *importEndpointSliceName) Get() string {
	return fmt.Sprintf(
		"clusterlink-%d-%s-%s",
		len(n.importName), n.importName, n.dataplaneEndpointSliceName)
}

func (n *importEndpointSliceName) Parse(importEndpointSliceName string) bool {
	components := strings.SplitN(importEndpointSliceName, "-", 3)
	if len(components) != 3 || components[0] != "clusterlink" {
		return false
	}

	importNameLength, err := strconv.Atoi(components[1])
	if err != nil || importNameLength <= 0 || importNameLength >= len(components[2]) {
		return false
	}

	n.importName = components[2][:importNameLength]
	n.dataplaneEndpointSliceName = components[2][importNameLength+1:]

	return n.dataplaneEndpointSliceName != ""
}

// Manager is responsible for handling control operations,
// which needs to be coordinated across all dataplane/controlplane instances.
// This includes target port generation for imported services, as well as
// k8s service creation per imported service.
type Manager struct {
	peerManager

	client    client.Client
	namespace string
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
		serviceValidCond := &metav1.Condition{
			Type:   v1alpha1.ImportServiceValid,
			Status: metav1.ConditionTrue,
			Reason: "Valid",
		}

		if err != nil {
			serviceValidCond.Status = metav1.ConditionFalse
			serviceValidCond.Reason = "Error"
			serviceValidCond.Message = err.Error()
		}

		conditions := &imp.Status.Conditions
		if conditionChanged(conditions, serviceValidCond) || conditionChanged(conditions, targetPortValidCond) {
			meta.SetStatusCondition(conditions, *targetPortValidCond)
			meta.SetStatusCondition(conditions, *serviceValidCond)

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

		if errors.Is(err, &conflictingServiceError{}) ||
			errors.Is(err, &conflictingTargetPortError{}) ||
			errors.Is(err, &importServiceNotExistError{}) {
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

	if imp.Labels[v1alpha1.LabelImportMerge] == "true" {
		return m.addImportEndpointSlices(ctx, imp)
	}

	importName := types.NamespacedName{
		Namespace: imp.Namespace,
		Name:      imp.Name,
	}

	// delete import endpoint slices, in case the import was previously a "merge: true" import
	if err := m.deleteImportEndpointSlices(ctx, importName); err != nil {
		return err
	}

	serviceName := imp.Name
	if imp.Namespace != m.namespace {
		serviceName = SystemServiceName(importName)
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

	if imp.Namespace != m.namespace {
		userService := &v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      imp.Name,
				Namespace: imp.Namespace,
				Labels:    make(map[string]string),
			},
			Spec: v1.ServiceSpec{
				ExternalName: fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, m.namespace),
				Type:         v1.ServiceTypeExternalName,
			},
		}

		if err := m.addImportService(ctx, imp, userService); err != nil {
			return err
		}
	}

	if imp.Spec.Alias != "" {
		if err := addCoreDNSRewrite(ctx, m.client, m.logger, &importName, imp.Spec.Alias); err != nil {
			m.logger.Errorf("failed to configure CoreDNS: %v.", err)
			return err
		}
	}
	return nil

}

// DeleteImport removes the listening socket of a previously imported service.
func (m *Manager) DeleteImport(ctx context.Context, name types.NamespacedName) error {
	m.logger.Infof("Deleting import '%s/%s'.", name.Namespace, name.Name)

	// delete user service
	errs := make([]error, 4)
	errs[0] = m.deleteImportService(ctx, name, name)

	if name.Namespace != m.namespace {
		// delete system service
		systemService := types.NamespacedName{
			Namespace: m.namespace,
			Name:      SystemServiceName(name),
		}
		errs[1] = m.deleteImportService(ctx, systemService, name)
	}

	// delete import endpoint slices
	errs[2] = m.deleteImportEndpointSlices(ctx, name)

	m.ports.Release(name)

	errs[3] = removeCoreDNSRewrite(ctx, m.client, m.logger, &name)

	return errors.Join(errs...)
}

// AddExport defines a new route target for ingress dataplane connections.
func (m *Manager) AddExport(ctx context.Context, export *v1alpha1.Export) (err error) {
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

// addEndpointSlice adds a dataplane / import endpoint slices.
func (m *Manager) addEndpointSlice(ctx context.Context, endpointSlice *discv1.EndpointSlice) error {
	if endpointSlice.Labels[discv1.LabelServiceName] == dpapp.Name && endpointSlice.Namespace == m.namespace {
		m.logger.Infof("Adding a dataplane endpoint slice: %s", endpointSlice.Name)

		mergeImportList := v1alpha1.ImportList{}
		labelSelector := client.MatchingLabels{v1alpha1.LabelImportMerge: "true"}
		if err := m.client.List(ctx, &mergeImportList, labelSelector); err != nil {
			return err
		}

		mergeImports := &mergeImportList.Items
		for i := range *mergeImports {
			err := m.checkImportEndpointSlice(ctx, &(*mergeImports)[i], endpointSlice)
			if err != nil {
				return err
			}
		}

		return nil
	}

	return m.checkEndpointSlice(ctx, endpointSlice.Namespace, endpointSlice.Name)
}

// deleteEndpointSlice is used to track deleted dataplane endpoint slices.
func (m *Manager) deleteEndpointSlice(ctx context.Context, name types.NamespacedName) error {
	if err := m.checkEndpointSlice(ctx, name.Namespace, name.Name); err != nil {
		return err
	}

	if name.Namespace != m.namespace {
		// not a dataplane endpoint slice
		return nil
	}

	importsEndpointSliceList := discv1.EndpointSliceList{}
	err := m.client.List(
		ctx,
		&importsEndpointSliceList,
		client.MatchingLabels{
			discv1.LabelManagedBy:    AppName,
			LabelDPEndpointSliceName: name.Name,
		})
	if err != nil {
		return err
	}

	importsEndpointSlices := &importsEndpointSliceList.Items
	for i := range *importsEndpointSlices {
		importEndpointSlice := &(*importsEndpointSlices)[i]
		m.logger.Infof(
			"Deleting import endpoint slice: %s/%s",
			importEndpointSlice.Namespace, importEndpointSlice.Name)
		err := m.client.Delete(ctx, importEndpointSlice)
		if err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func (m *Manager) checkExportService(ctx context.Context, name types.NamespacedName) error {
	var export v1alpha1.Export
	if err := m.client.Get(ctx, name, &export); err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}

		return nil
	}

	return m.AddExport(ctx, &export)
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

		m.logger.Infof("Updating target port for import %v.", name)
		if err := m.client.Update(ctx, imp); err != nil {
			m.ports.Release(name)
			return err
		}
	}

	return nil
}

func (m *Manager) addImportService(ctx context.Context, imp *v1alpha1.Import, service *v1.Service) error {
	service.Labels[LabelManagedBy] = AppName
	service.Labels[LabelImportName] = imp.Name
	service.Labels[LabelImportNamespace] = imp.Namespace

	importName := types.NamespacedName{
		Namespace: imp.Namespace,
		Name:      imp.Name,
	}

	if imp.Namespace != service.Namespace {
		m.lock.Lock()
		m.serviceToImport[service.Name] = importName
		m.lock.Unlock()
	}

	serviceName := types.NamespacedName{
		Name:      service.Name,
		Namespace: service.Namespace,
	}

	var oldService v1.Service
	err := m.client.Get(ctx, serviceName, &oldService)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}

		m.logger.Infof("Creating service: %s/%s.", service.Namespace, service.Name)
		return m.client.Create(ctx, service)
	}

	if !checkServiceLabels(&oldService, importName) {
		return conflictingServiceError{
			name:      serviceName,
			managedBy: oldService.Labels[LabelManagedBy],
		}
	}

	if !serviceChanged(&oldService, service) {
		// service already exists as expected
		return nil
	}

	m.logger.Infof("Updating service: %s/%s.", service.Namespace, service.Name)
	return m.client.Update(ctx, service)
}

func (m *Manager) deleteImportService(ctx context.Context, service, imp types.NamespacedName) error {
	var oldService v1.Service
	err := m.client.Get(ctx, service, &oldService)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}

		return nil
	}

	if !checkServiceLabels(&oldService, imp) {
		return nil
	}

	m.logger.Infof("Deleting service: %v.", service)
	err = m.client.Delete(ctx, &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      service.Name,
			Namespace: service.Namespace,
		},
	})
	if err != nil {
		return err
	}

	if oldService.Labels[LabelImportNamespace] != imp.Namespace {
		m.lock.Lock()
		delete(m.serviceToImport, service.Name)
		m.lock.Unlock()
	}

	return nil
}

func (m *Manager) checkEndpointSlice(ctx context.Context, namespace, endpointSliceName string) error {
	var parsed importEndpointSliceName
	if !parsed.Parse(endpointSliceName) {
		return nil
	}

	m.logger.Infof(
		"Checking import endpoint slice %s/%s'.",
		namespace, endpointSliceName)

	var imp v1alpha1.Import
	var dataplaneEndpointSlice discv1.EndpointSlice
	shouldDelete := false
	if err := m.client.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      parsed.importName,
	}, &imp); err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}

		m.logger.Infof(
			"Deleting an import endpoint slice with no corresponding import: %s",
			endpointSliceName)
		shouldDelete = true
	} else if imp.Labels[v1alpha1.LabelImportMerge] != "true" {
		m.logger.Infof(
			"Deleting an import endpoint slice with no corresponding merge-type import: %s",
			endpointSliceName)

		shouldDelete = true
	} else if err := m.client.Get(ctx, types.NamespacedName{
		Namespace: m.namespace,
		Name:      parsed.dataplaneEndpointSliceName,
	}, &dataplaneEndpointSlice); err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}

		m.logger.Infof(
			"Deleting an import endpoint slice with no corresponding dataplane endpoint slice: %s",
			endpointSliceName)
		shouldDelete = true
	}

	if shouldDelete {
		err := m.client.Delete(ctx, &discv1.EndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Name:      endpointSliceName,
				Namespace: namespace,
			},
		})
		if err != nil && !k8serrors.IsNotFound(err) {
			return err
		}

		return nil
	}

	return m.checkImportEndpointSlice(ctx, &imp, &dataplaneEndpointSlice)
}

func (m *Manager) checkImportEndpointSlice(
	ctx context.Context,
	imp *v1alpha1.Import,
	dataplaneEndpointSlice *discv1.EndpointSlice,
) error {
	m.logger.Infof(
		"Checking endpoint slice %s for import %s/%s'.",
		dataplaneEndpointSlice.Name, imp.Namespace, imp.Name)

	importEndpointSliceName := (&importEndpointSliceName{
		importName:                 imp.Name,
		dataplaneEndpointSliceName: dataplaneEndpointSlice.Name,
	}).Get()
	protocol := v1.ProtocolTCP
	port32 := int32(imp.Spec.TargetPort)

	importEndpointSlice := discv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      importEndpointSliceName,
			Namespace: imp.Namespace,
			Labels: map[string]string{
				discv1.LabelServiceName:  imp.Name,
				discv1.LabelManagedBy:    AppName,
				LabelDPEndpointSliceName: dataplaneEndpointSlice.Name,
			},
		},
		AddressType: discv1.AddressTypeIPv4,
		Endpoints:   dataplaneEndpointSlice.Endpoints,
		Ports: []discv1.EndpointPort{
			{
				Port:     &port32,
				Protocol: &protocol,
			},
		},
	}

	var oldImportEndpointSlice discv1.EndpointSlice
	err := m.client.Get(
		ctx,
		types.NamespacedName{
			Name:      importEndpointSliceName,
			Namespace: imp.Namespace,
		},
		&oldImportEndpointSlice)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return err
		}

		m.logger.Infof("Creating import endpoint slice: %s.", importEndpointSliceName)
		return m.client.Create(ctx, &importEndpointSlice)
	}

	if !endpointSliceChanged(&importEndpointSlice, &oldImportEndpointSlice) {
		return nil
	}

	m.logger.Infof("Updating import endpoint slice: %s.", importEndpointSliceName)
	return m.client.Update(ctx, &importEndpointSlice)
}

func (m *Manager) addImportEndpointSlices(ctx context.Context, imp *v1alpha1.Import) error {
	// check that import service exists
	importName := types.NamespacedName{
		Namespace: imp.Namespace,
		Name:      imp.Name,
	}

	if err := m.client.Get(ctx, importName, &v1.Service{}); err != nil {
		if k8serrors.IsNotFound(err) {
			return &importServiceNotExistError{name: importName}
		}

		return err
	}

	// get dataplane endpoint slices
	dataplaneEndpointSliceList := discv1.EndpointSliceList{}
	err := m.client.List(
		ctx,
		&dataplaneEndpointSliceList,
		client.MatchingLabels{discv1.LabelServiceName: dpapp.Name},
		client.InNamespace(m.namespace))
	if err != nil {
		return err
	}

	// copy dataplane endpoint slices to import endpoint slices
	dataplaneEndpointSlices := &dataplaneEndpointSliceList.Items
	for i := range *dataplaneEndpointSlices {
		err := m.checkImportEndpointSlice(ctx, imp, &(*dataplaneEndpointSlices)[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) deleteImportEndpointSlices(ctx context.Context, imp types.NamespacedName) error {
	endpointSlices := discv1.EndpointSliceList{}
	labelSelector := client.MatchingLabels{
		discv1.LabelManagedBy:   AppName,
		discv1.LabelServiceName: imp.Name,
	}
	if err := m.client.List(ctx, &endpointSlices, labelSelector, client.InNamespace(imp.Namespace)); err != nil {
		return err
	}

	for i := range endpointSlices.Items {
		endpointSlice := &endpointSlices.Items[i]
		m.logger.Infof("Deleting import endpoint slice: %s", endpointSlice.Name)
		err := m.client.Delete(ctx, endpointSlice)
		if err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

func checkServiceLabels(service *v1.Service, importName types.NamespacedName) bool {
	if managedBy, ok := service.Labels[LabelManagedBy]; !ok || managedBy != AppName {
		return false
	}

	if name, ok := service.Labels[LabelImportName]; !ok || name != importName.Name {
		return false
	}

	if namespace, ok := service.Labels[LabelImportNamespace]; !ok || namespace != importName.Namespace {
		return false
	}

	return true
}

func SystemServiceName(name types.NamespacedName) string {
	//nolint:gosec // G401: use of weak cryptographic primitive is fine for service name
	hash := md5.New()
	hash.Write([]byte(name.Namespace + "/" + name.Name))
	return fmt.Sprintf(
		"import-%s-%s-%x",
		k8sstrings.ShortenString(name.Name, 10),
		k8sstrings.ShortenString(name.Namespace, 10),
		hash.Sum(nil))
}

func serviceChanged(svc1, svc2 *v1.Service) bool {
	if svc1.Spec.Type != svc2.Spec.Type {
		return true
	}

	if svc1.Spec.ExternalName != svc2.Spec.ExternalName {
		return true
	}

	if len(svc1.Spec.Selector) != len(svc2.Spec.Selector) {
		return true
	}

	for key, value1 := range svc1.Spec.Selector {
		if value2, ok := svc2.Spec.Selector[key]; !ok || value2 != value1 {
			return true
		}
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

func endpointSliceChanged(endpointSlice1, endpointSlice2 *discv1.EndpointSlice) bool {
	if endpointSlice1.AddressType != endpointSlice2.AddressType {
		return true
	}

	if !reflect.DeepEqual(endpointSlice1.Labels, endpointSlice2.Labels) {
		return true
	}

	if len(endpointSlice1.Endpoints) != len(endpointSlice2.Endpoints) {
		return true
	}

	for i := range endpointSlice1.Endpoints {
		addresses1 := endpointSlice1.Endpoints[i].Addresses
		addresses2 := endpointSlice2.Endpoints[i].Addresses
		if len(addresses1) != len(addresses2) {
			return true
		}

		for j := range addresses1 {
			if addresses1[j] != addresses2[j] {
				return true
			}
		}
	}

	return false
}

// NewManager returns a new control manager.
func NewManager(cl client.Client, namespace string) *Manager {
	logger := logrus.WithField("component", "controlplane.control.manager")

	return &Manager{
		peerManager:     newPeerManager(cl),
		client:          cl,
		namespace:       namespace,
		ports:           newPortManager(),
		serviceToImport: make(map[string]types.NamespacedName),
		logger:          logger,
	}
}
