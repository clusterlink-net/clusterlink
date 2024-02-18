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

type lbPolicyHandler struct {
	cp *controlplane.Instance
}

func lbPolicyToAPI(policy *store.LBPolicy) *api.Policy {
	return &policy.Policy
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
