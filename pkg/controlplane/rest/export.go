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

// Copyright (c) 2022 The ClusterLink Authors.
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

// Copyright (C) The ClusterLink Authors.
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

package rest

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/store"
)

type exportHandler struct {
	manager *Manager
}

func toK8SExport(export *store.Export, namespace string) *v1alpha1.Export {
	return &v1alpha1.Export{
		ObjectMeta: metav1.ObjectMeta{
			Name:      export.Name,
			Namespace: namespace,
		},
		Spec: v1alpha1.ExportSpec{
			Host: export.ExportSpec.Host,
			Port: export.ExportSpec.Port,
		},
		Status: export.Status,
	}
}

func exportToAPI(export *store.Export) *v1alpha1.Export {
	if export == nil {
		return nil
	}

	return &v1alpha1.Export{
		ObjectMeta: metav1.ObjectMeta{
			Name: export.Name,
		},
		Spec:   export.ExportSpec,
		Status: export.Status,
	}
}

// CreateExport defines a new route target for ingress dataplane connections.
func (m *Manager) CreateExport(export *store.Export) error {
	m.logger.Infof("Creating export '%s'.", export.Name)

	if m.initialized {
		if err := m.exports.Create(export); err != nil {
			return err
		}
	}

	k8sExport := toK8SExport(export, m.namespace)
	if err := m.xdsManager.AddExport(k8sExport); err != nil {
		return err
	}
	return m.controlManager.AddExport(context.Background(), k8sExport)
}

// UpdateExport updates a new route target for ingress dataplane connections.
func (m *Manager) UpdateExport(export *store.Export) error {
	m.logger.Infof("Updating export '%s'.", export.Name)

	err := m.exports.Update(export.Name, func(old *store.Export) *store.Export {
		return export
	})
	if err != nil {
		return err
	}

	k8sExport := toK8SExport(export, m.namespace)
	if err := m.xdsManager.AddExport(k8sExport); err != nil {
		return err
	}
	return m.controlManager.AddExport(context.Background(), k8sExport)
}

// UpdateExportStatus updates the status of an existing export.
func (m *Manager) UpdateExportStatus(name string, status *v1alpha1.ExportStatus) {
	m.logger.Infof("Updating status of export '%s'.", name)

	err := m.exports.Update(name, func(old *store.Export) *store.Export {
		old.Status = *status
		return old
	})
	if err != nil {
		m.logger.Errorf("Error updating status of export '%s': %v", name, err)
	}
}

// GetExport returns an existing export.
func (m *Manager) GetExport(name string) *store.Export {
	m.logger.Infof("Getting export '%s'.", name)
	return m.exports.Get(name)
}

// DeleteExport removes the possibility for ingress dataplane connections to access a given service.
func (m *Manager) DeleteExport(name string) (*store.Export, error) {
	m.logger.Infof("Deleting export '%s'.", name)

	export, err := m.exports.Delete(name)
	if err != nil {
		return nil, err
	}
	if export == nil {
		return nil, nil
	}

	namespacedName := types.NamespacedName{
		Name:      name,
		Namespace: m.namespace,
	}
	err = m.xdsManager.DeleteExport(namespacedName)
	if err != nil {
		// practically impossible
		return export, err
	}

	return export, nil
}

// GetAllExports returns the list of all exports.
func (m *Manager) GetAllExports() []*store.Export {
	m.logger.Info("Listing all exports.")
	return m.exports.GetAll()
}

func (m *Manager) GetK8sExport(name string, export *v1alpha1.Export) error {
	storeExport := m.exports.Get(name)
	if storeExport == nil {
		return errors.NewNotFound(schema.GroupResource{}, name)
	}

	*export = *toK8SExport(storeExport, m.namespace)
	return nil
}

// Decode an export.
func (h *exportHandler) Decode(data []byte) (any, error) {
	var export v1alpha1.Export
	if err := json.Unmarshal(data, &export); err != nil {
		return nil, fmt.Errorf("cannot decode export: %w", err)
	}

	if export.Name == "" {
		return nil, fmt.Errorf("empty export name")
	}

	if export.Spec.Port == 0 {
		return nil, fmt.Errorf("missing service port")
	}

	return store.NewExport(&export), nil
}

// Create an export.
func (h *exportHandler) Create(object any) error {
	return h.manager.CreateExport(object.(*store.Export))
}

// Update an export.
func (h *exportHandler) Update(object any) error {
	return h.manager.UpdateExport(object.(*store.Export))
}

// Get an export.
func (h *exportHandler) Get(name string) (any, error) {
	export := exportToAPI(h.manager.GetExport(name))
	if export == nil {
		return nil, nil
	}
	return export, nil
}

// Delete an export.
func (h *exportHandler) Delete(name any) (any, error) {
	return h.manager.DeleteExport(name.(string))
}

// List all exports.
func (h *exportHandler) List() (any, error) {
	exports := h.manager.GetAllExports()
	apiExports := make([]*v1alpha1.Export, len(exports))
	for i, export := range exports {
		apiExports[i] = exportToAPI(export)
	}
	return apiExports, nil
}
