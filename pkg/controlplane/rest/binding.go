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

	"github.com/clusterlink-net/clusterlink/pkg/api"
	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/store"
)

type bindingHandler struct {
	manager *Manager
}

func bindingsToAPI(bindings []*store.Binding) []*api.Binding {
	apiBindings := make([]*api.Binding, len(bindings))
	for i, binding := range bindings {
		apiBindings[i] = &api.Binding{Spec: binding.BindingSpec}
	}
	return apiBindings
}

// CreateBinding creates a binding of an imported service to a remote exported service.
func (m *Manager) CreateBinding(binding *store.Binding) error {
	m.logger.Infof("Creating binding '%s'->'%s'.", binding.Import, binding.Peer)

	m.authzManager.AddImport(&v1alpha1.Import{
		ObjectMeta: metav1.ObjectMeta{Name: binding.Import},
		Spec: v1alpha1.ImportSpec{
			Sources: []v1alpha1.ImportSource{
				{
					Peer:       binding.Peer,
					ExportName: binding.Import,
				},
			},
		},
	})

	if m.initialized {
		if err := m.bindings.Create(binding); err != nil {
			return err
		}
	}

	return nil
}

// UpdateBinding updates a binding of an imported service to a remote exported service.
func (m *Manager) UpdateBinding(binding *store.Binding) error {
	m.logger.Infof("Updating binding '%s'->'%s'.", binding.Import, binding.Peer)

	m.authzManager.AddImport(&v1alpha1.Import{
		ObjectMeta: metav1.ObjectMeta{Name: binding.Import},
		Spec: v1alpha1.ImportSpec{
			Sources: []v1alpha1.ImportSource{
				{
					Peer:       binding.Peer,
					ExportName: binding.Import,
				},
			},
		},
	})

	err := m.bindings.Update(binding, func(old *store.Binding) *store.Binding {
		return binding
	})
	if err != nil {
		return err
	}

	return nil
}

// GetBindings returns all bindings for a given imported service.
func (m *Manager) GetBindings(imp string) []*store.Binding {
	m.logger.Infof("Getting bindings for import '%s'.", imp)
	return m.bindings.Get(imp)
}

// DeleteBinding removes a binding of an imported service to a remote exported service.
func (m *Manager) DeleteBinding(binding *store.Binding) (*store.Binding, error) {
	m.logger.Infof("Deleting binding '%s'->'%s'.", binding.Import, binding.Peer)

	// TODO: m.authzManager.Delete*

	return m.bindings.Delete(binding)
}

// GetAllBindings returns the list of all bindings.
func (m *Manager) GetAllBindings() []*store.Binding {
	m.logger.Info("Listing all bindings.")
	return m.bindings.GetAll()
}

// Decode a binding.
func (h *bindingHandler) Decode(data []byte) (any, error) {
	var binding api.Binding
	if err := json.Unmarshal(data, &binding); err != nil {
		return nil, fmt.Errorf("cannot decode binding: %w", err)
	}

	if binding.Spec.Import == "" {
		return nil, fmt.Errorf("empty import name")
	}

	if binding.Spec.Peer == "" {
		return nil, fmt.Errorf("empty peer name")
	}

	return store.NewBinding(&binding), nil
}

// Create a binding.
func (h *bindingHandler) Create(object any) error {
	return h.manager.CreateBinding(object.(*store.Binding))
}

// Create a binding.
func (h *bindingHandler) Update(object any) error {
	return h.manager.UpdateBinding(object.(*store.Binding))
}

// Get a binding.
func (h *bindingHandler) Get(name string) (any, error) {
	binding := bindingsToAPI(h.manager.GetBindings(name))
	if binding == nil {
		return nil, nil
	}
	return binding, nil
}

// Delete a binding.
func (h *bindingHandler) Delete(object any) (any, error) {
	return h.manager.DeleteBinding(object.(*store.Binding))
}

// List all bindings.
func (h *bindingHandler) List() (any, error) {
	return bindingsToAPI(h.manager.GetAllBindings()), nil
}
