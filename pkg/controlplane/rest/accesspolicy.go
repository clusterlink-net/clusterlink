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

package rest

import (
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/authz/connectivitypdp"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/store"
)

type accessPolicyHandler struct {
	manager *Manager
}

func toPDPPolicy(policy *store.AccessPolicy, namespace string) *connectivitypdp.AccessPolicy {
	return connectivitypdp.PolicyFromCR(&v1alpha1.AccessPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      policy.Name,
			Namespace: namespace,
		},
		Spec: policy.AccessPolicySpec,
	})
}

func accessPolicyToAPI(policy *store.AccessPolicy) *v1alpha1.AccessPolicy {
	if policy == nil {
		return nil
	}

	return &v1alpha1.AccessPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: policy.Name,
		},
		Spec: policy.AccessPolicySpec,
	}
}

// CreateAccessPolicy creates an access policy to allow/deny specific connections.
func (m *Manager) CreateAccessPolicy(policy *store.AccessPolicy) error {
	m.logger.Infof("Creating access policy '%s'.", policy.Name)

	if m.initialized {
		if err := m.acPolicies.Create(policy); err != nil {
			return err
		}
	}

	return m.authzManager.AddAccessPolicy(toPDPPolicy(policy, m.namespace))
}

// UpdateAccessPolicy updates an access policy to allow/deny specific connections.
func (m *Manager) UpdateAccessPolicy(policy *store.AccessPolicy) error {
	m.logger.Infof("Updating access policy '%s'.", policy.Name)

	err := m.acPolicies.Update(policy.Name, func(old *store.AccessPolicy) *store.AccessPolicy {
		return policy
	})
	if err != nil {
		return err
	}

	return m.authzManager.AddAccessPolicy(toPDPPolicy(policy, m.namespace))
}

// DeleteAccessPolicy removes an access policy to allow/deny specific connections.
func (m *Manager) DeleteAccessPolicy(name string) (*store.AccessPolicy, error) {
	m.logger.Infof("Deleting access policy '%s'.", name)

	policy, err := m.acPolicies.Delete(name)
	if err != nil {
		return nil, err
	}
	if policy == nil {
		return nil, nil
	}

	namespacedName := types.NamespacedName{
		Name:      name,
		Namespace: m.namespace,
	}
	if err := m.authzManager.DeleteAccessPolicy(namespacedName, false); err != nil {
		return nil, err
	}

	return policy, err
}

// GetAccessPolicy returns an access policy with the given name.
func (m *Manager) GetAccessPolicy(name string) *store.AccessPolicy {
	m.logger.Infof("Getting access policy '%s'.", name)
	return m.acPolicies.Get(name)
}

// GetAllAccessPolicies returns the list of all AccessPolicies.
func (m *Manager) GetAllAccessPolicies() []*store.AccessPolicy {
	m.logger.Info("Listing all access policies.")
	return m.acPolicies.GetAll()
}

// Decode an access policy.
func (h *accessPolicyHandler) Decode(data []byte) (any, error) {
	var policy v1alpha1.AccessPolicy
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil, fmt.Errorf("cannot decode access policy: %w", err)
	}

	if err := policy.Spec.Validate(); err != nil {
		return nil, err
	}

	return store.NewAccessPolicy(&policy), nil
}

// Create an access policy.
func (h *accessPolicyHandler) Create(object any) error {
	return h.manager.CreateAccessPolicy(object.(*store.AccessPolicy))
}

// Update an access policy.
func (h *accessPolicyHandler) Update(object any) error {
	return h.manager.UpdateAccessPolicy(object.(*store.AccessPolicy))
}

// Delete an access policy.
func (h *accessPolicyHandler) Delete(name any) (any, error) {
	return h.manager.DeleteAccessPolicy(name.(string))
}

// Get an access policy.
func (h *accessPolicyHandler) Get(name string) (any, error) {
	policy := h.manager.GetAccessPolicy(name)
	if policy == nil {
		return nil, nil
	}
	return accessPolicyToAPI(policy), nil
}

// List all access policies.
func (h *accessPolicyHandler) List() (any, error) {
	policies := h.manager.GetAllAccessPolicies()
	apiPolicies := make([]*v1alpha1.AccessPolicy, len(policies))
	for i, policy := range policies {
		apiPolicies[i] = accessPolicyToAPI(policy)
	}
	return apiPolicies, nil
}
