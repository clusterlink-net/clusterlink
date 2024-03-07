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
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/clusterlink-net/clusterlink/pkg/api"
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
			Host: export.ExportSpec.Service.Host,
			Port: export.ExportSpec.Service.Port,
		},
	}
}

func exportToAPI(export *store.Export) *api.Export {
	if export == nil {
		return nil
	}

	return &api.Export{
		Name: export.Name,
		Spec: export.ExportSpec,
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
	m.authzManager.AddExport(k8sExport)
	return m.xdsManager.AddExport(k8sExport)
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
	m.authzManager.AddExport(k8sExport)
	return m.xdsManager.AddExport(k8sExport)
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

	m.authzManager.DeleteExport(namespacedName)

	return export, nil
}

// GetAllExports returns the list of all exports.
func (m *Manager) GetAllExports() []*store.Export {
	m.logger.Info("Listing all exports.")
	return m.exports.GetAll()
}

// Decode an export.
func (h *exportHandler) Decode(data []byte) (any, error) {
	var export api.Export
	if err := json.Unmarshal(data, &export); err != nil {
		return nil, fmt.Errorf("cannot decode export: %w", err)
	}

	if export.Name == "" {
		return nil, fmt.Errorf("empty export name")
	}

	if export.Spec.Service.Host == "" {
		return nil, fmt.Errorf("missing service name")
	}

	if export.Spec.Service.Port == 0 {
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
	apiExports := make([]*api.Export, len(exports))
	for i, export := range exports {
		apiExports[i] = exportToAPI(export)
	}
	return apiExports, nil
}
