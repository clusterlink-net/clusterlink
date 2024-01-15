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
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/store"
)

type lbPolicyHandler struct {
	manager *Manager
}

func lbPolicyToAPI(policy *store.LBPolicy) *api.Policy {
	return &policy.Policy
}

// CreateLBPolicy creates a load-balancing policy to set a load-balancing scheme for specific connections.
func (m *Manager) CreateLBPolicy(policy *store.LBPolicy) error {
	m.logger.Infof("Creating load-balancing policy '%s'.", policy.Spec.Blob)

	if m.initialized {
		if err := m.lbPolicies.Create(policy); err != nil {
			return err
		}
	}

	return m.authzManager.AddLBPolicy(&api.Policy{Spec: policy.Spec})
}

// UpdateLBPolicy updates a load-balancing policy.
func (m *Manager) UpdateLBPolicy(policy *store.LBPolicy) error {
	m.logger.Infof("Updating load-balancing policy '%s'.", policy.Spec.Blob)

	err := m.lbPolicies.Update(policy.Name, func(old *store.LBPolicy) *store.LBPolicy {
		return policy
	})
	if err != nil {
		return err
	}

	return m.authzManager.AddLBPolicy(&api.Policy{Spec: policy.Spec})
}

// DeleteLBPolicy removes a load-balancing policy.
func (m *Manager) DeleteLBPolicy(name string) (*store.LBPolicy, error) {
	m.logger.Infof("Deleting load-balancing policy '%s'.", name)

	policy, err := m.lbPolicies.Delete(name)
	if err != nil {
		return nil, err
	}
	if policy == nil {
		return nil, nil
	}

	if err := m.authzManager.DeleteLBPolicy(&policy.Policy); err != nil {
		return nil, err
	}

	return policy, nil
}

// GetLBPolicy returns a load-balancing policy with the given name.
func (m *Manager) GetLBPolicy(name string) *store.LBPolicy {
	m.logger.Infof("Getting load-balancing policy '%s'.", name)
	return m.lbPolicies.Get(name)
}

// GetAllLBPolicies returns the list of all load-balancing Policies.p.
func (m *Manager) GetAllLBPolicies() []*store.LBPolicy {
	m.logger.Info("Listing all load-balancing policies.")
	return m.lbPolicies.GetAll()
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
	return h.manager.CreateLBPolicy(object.(*store.LBPolicy))
}

// Update an load-balancing policy.
func (h *lbPolicyHandler) Update(object any) error {
	return h.manager.UpdateLBPolicy(object.(*store.LBPolicy))
}

// Delete a load-balancing policy.
func (h *lbPolicyHandler) Delete(name any) (any, error) {
	return h.manager.DeleteLBPolicy(name.(string))
}

// Get an load-balancing policy.
func (h *lbPolicyHandler) Get(name string) (any, error) {
	policy := h.manager.GetLBPolicy(name)
	if policy == nil {
		return nil, nil
	}
	return lbPolicyToAPI(policy), nil
}

// List all load-balancing policies.
func (h *lbPolicyHandler) List() (any, error) {
	policies := h.manager.GetAllLBPolicies()
	apiPolicies := make([]*api.Policy, len(policies))
	for i, policy := range policies {
		apiPolicies[i] = lbPolicyToAPI(policy)
	}
	return apiPolicies, nil
}
