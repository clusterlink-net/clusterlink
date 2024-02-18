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

	"github.com/clusterlink-net/clusterlink/pkg/api"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/store"
)

type bindingHandler struct {
	cp *controlplane.Instance
}

func bindingsToAPI(bindings []*store.Binding) []*api.Binding {
	apiBindings := make([]*api.Binding, len(bindings))
	for i, binding := range bindings {
		apiBindings[i] = &api.Binding{Spec: binding.BindingSpec}
	}
	return apiBindings
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
	return h.cp.CreateBinding(object.(*store.Binding))
}

// Create a binding.
func (h *bindingHandler) Update(object any) error {
	return h.cp.UpdateBinding(object.(*store.Binding))
}

// Get a binding.
func (h *bindingHandler) Get(name string) (any, error) {
	binding := bindingsToAPI(h.cp.GetBindings(name))
	if binding == nil {
		return nil, nil
	}
	return binding, nil
}

// Delete a binding.
func (h *bindingHandler) Delete(object any) (any, error) {
	return h.cp.DeleteBinding(object.(*store.Binding))
}

// List all bindings.
func (h *bindingHandler) List() (any, error) {
	return bindingsToAPI(h.cp.GetAllBindings()), nil
}
