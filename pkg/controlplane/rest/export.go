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

type exportHandler struct {
	cp *controlplane.Instance
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
	return h.cp.CreateExport(object.(*store.Export))
}

// Update an export.
func (h *exportHandler) Update(object any) error {
	return h.cp.UpdateExport(object.(*store.Export))
}

// Get an export.
func (h *exportHandler) Get(name string) (any, error) {
	export := exportToAPI(h.cp.GetExport(name))
	if export == nil {
		return nil, nil
	}
	return export, nil
}

// Delete an export.
func (h *exportHandler) Delete(name any) (any, error) {
	return h.cp.DeleteExport(name.(string))
}

// List all exports.
func (h *exportHandler) List() (any, error) {
	exports := h.cp.GetAllExports()
	apiExports := make([]*api.Export, len(exports))
	for i, export := range exports {
		apiExports[i] = exportToAPI(export)
	}
	return apiExports, nil
}
