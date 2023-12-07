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

package http

import (
	"encoding/json"
	"fmt"

	"github.com/clusterlink-net/clusterlink/pkg/api"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/store"
	"github.com/clusterlink-net/clusterlink/pkg/util/rest"
)

func (s *Server) addAPIHandlers() {
	// TODO: verify that request originates from a local admin

	s.AddObjectHandlers(&rest.ServerObjectSpec{
		BasePath:      "/peers",
		Handler:       &peerHandler{cp: s.cp},
		DeleteByValue: false,
	})

	s.AddObjectHandlers(&rest.ServerObjectSpec{
		BasePath:      "/exports",
		Handler:       &exportHandler{cp: s.cp},
		DeleteByValue: false,
	})

	s.AddObjectHandlers(&rest.ServerObjectSpec{
		BasePath:      "/imports",
		Handler:       &importHandler{cp: s.cp},
		DeleteByValue: false,
	})

	s.AddObjectHandlers(&rest.ServerObjectSpec{
		BasePath:      "/bindings",
		Handler:       &bindingHandler{cp: s.cp},
		DeleteByValue: true,
	})

	s.AddObjectHandlers(&rest.ServerObjectSpec{
		BasePath:      "/policies",
		Handler:       &accessPolicyHandler{cp: s.cp},
		DeleteByValue: false,
	})

	s.AddObjectHandlers(&rest.ServerObjectSpec{
		BasePath:      "/lbpolicies",
		Handler:       &lbPolicyHandler{cp: s.cp},
		DeleteByValue: false,
	})
}

type peerHandler struct {
	cp *controlplane.Instance
}

// Decode a peer.
func (h *peerHandler) Decode(data []byte) (any, error) {
	var peer api.Peer
	if err := json.Unmarshal(data, &peer); err != nil {
		return nil, fmt.Errorf("cannot decode peer: %w", err)
	}

	if peer.Name == "" {
		return nil, fmt.Errorf("empty peer name")
	}

	for i, ep := range peer.Spec.Gateways {
		if ep.Host == "" {
			return nil, fmt.Errorf("gateway #%d missing host", i)
		}
		if ep.Port == 0 {
			return nil, fmt.Errorf("gateway #%d (host '%s') missing port", i, ep.Host)
		}
	}

	return store.NewPeer(&peer), nil
}

// Create a peer.
func (h *peerHandler) Create(object any) error {
	return h.cp.CreatePeer(object.(*store.Peer))
}

// Update a peer.
func (h *peerHandler) Update(object any) error {
	return h.cp.UpdatePeer(object.(*store.Peer))
}

func peerToAPI(peer *store.Peer) *api.Peer {
	if peer == nil {
		return nil
	}

	return &api.Peer{
		Name:   peer.Name,
		Spec:   peer.PeerSpec,
		Status: api.PeerStatus{}, // TODO
	}
}

// Get a peer.
func (h *peerHandler) Get(name string) (any, error) {
	peer := peerToAPI(h.cp.GetPeer(name))
	if peer == nil {
		return nil, nil
	}
	return peer, nil
}

// Delete a peer.
func (h *peerHandler) Delete(name any) (any, error) {
	return h.cp.DeletePeer(name.(string))
}

// List all peers.
func (h *peerHandler) List() (any, error) {
	peers := h.cp.GetAllPeers()
	apiPeers := make([]*api.Peer, len(peers))
	for i, peer := range peers {
		apiPeers[i] = peerToAPI(peer)
	}
	return apiPeers, nil
}

type exportHandler struct {
	cp *controlplane.Instance
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

func exportToAPI(export *store.Export) *api.Export {
	if export == nil {
		return nil
	}

	return &api.Export{
		Name: export.Name,
		Spec: export.ExportSpec,
	}
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

type importHandler struct {
	cp *controlplane.Instance
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

func importToAPI(imp *store.Import) *api.Import {
	if imp == nil {
		return nil
	}

	return &api.Import{
		Name: imp.Name,
		Spec: imp.ImportSpec,
		Status: api.ImportStatus{
			Listener: api.Endpoint{
				Host: "", // TODO
				Port: imp.Port,
			},
		},
	}
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

type bindingHandler struct {
	cp *controlplane.Instance
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

func bindingsToAPI(bindings []*store.Binding) []*api.Binding {
	apiBindings := make([]*api.Binding, len(bindings))
	for i, binding := range bindings {
		apiBindings[i] = &api.Binding{Spec: binding.BindingSpec}
	}
	return apiBindings
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

type accessPolicyHandler struct {
	cp *controlplane.Instance
}

// Decode an access policy.
func (h *accessPolicyHandler) Decode(data []byte) (any, error) {
	var policy api.Policy
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil, fmt.Errorf("cannot decode access policy: %w", err)
	}

	if len(policy.Spec.Blob) == 0 {
		return nil, fmt.Errorf("empty spec blob")
	}

	return store.NewAccessPolicy(&policy), nil
}

// Create an access policy.
func (h *accessPolicyHandler) Create(object any) error {
	return h.cp.CreateAccessPolicy(object.(*store.AccessPolicy))
}

// Update an access policy.
func (h *accessPolicyHandler) Update(object any) error {
	return h.cp.UpdateAccessPolicy(object.(*store.AccessPolicy))
}

func accessPolicyToAPI(policy *store.AccessPolicy) *api.Policy {
	return &policy.Policy
}

// Delete an access policy.
func (h *accessPolicyHandler) Delete(name any) (any, error) {
	return h.cp.DeleteAccessPolicy(name.(string))
}

// Get an access policy.
func (h *accessPolicyHandler) Get(name string) (any, error) {
	policy := h.cp.GetAccessPolicy(name)
	if policy == nil {
		return nil, nil
	}
	return accessPolicyToAPI(policy), nil
}

// List all access policies.
func (h *accessPolicyHandler) List() (any, error) {
	policies := h.cp.GetAllAccessPolicies()
	apiPolicies := make([]*api.Policy, len(policies))
	for i, policy := range policies {
		apiPolicies[i] = accessPolicyToAPI(policy)
	}
	return apiPolicies, nil
}

type lbPolicyHandler struct {
	cp *controlplane.Instance
}

// Decode a load-balancing policy.
func (h *lbPolicyHandler) Decode(data []byte) (any, error) {
	var policy api.Policy
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil, fmt.Errorf("cannot decode load-balancing policy: %w", err)
	}

	if len(policy.Spec.Blob) == 0 {
		return nil, fmt.Errorf("empty spec blob")
	}

	return store.NewLBPolicy(&policy), nil
}

// Create a load-balancing policy.
func (h *lbPolicyHandler) Create(object any) error {
	return h.cp.CreateLBPolicy(object.(*store.LBPolicy))
}

// Update an load-balancing policy.
func (h *lbPolicyHandler) Update(object any) error {
	return h.cp.UpdateLBPolicy(object.(*store.LBPolicy))
}

func lbPolicyToAPI(policy *store.LBPolicy) *api.Policy {
	return &policy.Policy
}

// Delete a load-balancing policy.
func (h *lbPolicyHandler) Delete(name any) (any, error) {
	return h.cp.DeleteLBPolicy(name.(string))
}

// Get an load-balancing policy.
func (h *lbPolicyHandler) Get(name string) (any, error) {
	policy := h.cp.GetLBPolicy(name)
	if policy == nil {
		return nil, nil
	}
	return lbPolicyToAPI(policy), nil
}

// List all load-balancing policies.
func (h *lbPolicyHandler) List() (any, error) {
	policies := h.cp.GetAllLBPolicies()
	apiPolicies := make([]*api.Policy, len(policies))
	for i, policy := range policies {
		apiPolicies[i] = lbPolicyToAPI(policy)
	}
	return apiPolicies, nil
}
