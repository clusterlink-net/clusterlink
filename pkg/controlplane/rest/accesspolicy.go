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

type accessPolicyHandler struct {
	cp *controlplane.Instance
}

func accessPolicyToAPI(policy *store.AccessPolicy) *api.Policy {
	return &policy.Policy
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
