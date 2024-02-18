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

type importHandler struct {
	cp *controlplane.Instance
}

func importToAPI(imp *store.Import) *api.Import {
	if imp == nil {
		return nil
	}

	return &api.Import{
		Name: imp.Name,
		Spec: imp.ImportSpec,
		Status: api.ImportStatus{
			Listener: api.Endpoint{ // Endpoint.Host is not set
				Port: imp.Port,
			},
		},
	}
}

// Decode an import.
func (h *importHandler) Decode(data []byte) (any, error) {
	var imp api.Import
	if err := json.Unmarshal(data, &imp); err != nil {
		return nil, fmt.Errorf("cannot decode import: %w", err)
	}

	if imp.Name == "" {
		return nil, fmt.Errorf("empty import name")
	}

	if imp.Spec.Service.Host == "" {
		return nil, fmt.Errorf("missing service name")
	}

	if imp.Spec.Service.Port == 0 {
		return nil, fmt.Errorf("missing service port")
	}

	return store.NewImport(&imp), nil
}

// Create an import.
func (h *importHandler) Create(object any) error {
	return h.cp.CreateImport(object.(*store.Import))
}

// Update an import.
func (h *importHandler) Update(object any) error {
	return h.cp.UpdateImport(object.(*store.Import))
}

// Get an import.
func (h *importHandler) Get(name string) (any, error) {
	imp := importToAPI(h.cp.GetImport(name))
	if imp == nil {
		return nil, nil
	}
	return imp, nil
}

// Delete an import.
func (h *importHandler) Delete(name any) (any, error) {
	return h.cp.DeleteImport(name.(string))
}

// List all imports.
func (h *importHandler) List() (any, error) {
	imports := h.cp.GetAllImports()
	apiImports := make([]*api.Import, len(imports))
	for i, imp := range imports {
		apiImports[i] = importToAPI(imp)
	}
	return apiImports, nil
}
