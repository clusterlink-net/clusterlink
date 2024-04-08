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
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/clusterlink-net/clusterlink/pkg/api"
	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/store"
	"github.com/clusterlink-net/clusterlink/pkg/policyengine/connectivitypdp"
	"k8s.io/apimachinery/pkg/types"
)

type accessPolicyHandler struct {
	manager *Manager
}

func accessPolicyToAPI(policy *store.AccessPolicy) *api.Policy {
	return &policy.Policy
}

// accessPolicyFromBlob unmarshals an AccessPolicy object encoded as json in a byte array.
func accessPolicyFromBlob(blob []byte) (*v1alpha1.AccessPolicy, error) {
	bReader := bytes.NewReader(blob)
	connPolicy := &v1alpha1.AccessPolicy{}
	err := json.NewDecoder(bReader).Decode(connPolicy)
	if err != nil {
		return nil, err
	}
	return connPolicy, nil
}

// CreateAccessPolicy creates an access policy to allow/deny specific connections.
func (m *Manager) CreateAccessPolicy(policy *store.AccessPolicy) error {
	m.logger.Infof("Creating access policy '%s'.", policy.Spec.Blob)

	if m.initialized {
		if err := m.acPolicies.Create(policy); err != nil {
			return err
		}
	}

	acPolicy, err := accessPolicyFromBlob(policy.Spec.Blob)
	if err != nil {
		m.logger.Errorf("failed decoding access policy %s", policy.Name)
		return err
	}
	return m.authzManager.AddAccessPolicy(connectivitypdp.PolicyFromCRD(acPolicy))
}

// UpdateAccessPolicy updates an access policy to allow/deny specific connections.
func (m *Manager) UpdateAccessPolicy(policy *store.AccessPolicy) error {
	m.logger.Infof("Updating access policy '%s'.", policy.Spec.Blob)

	err := m.acPolicies.Update(policy.Name, func(old *store.AccessPolicy) *store.AccessPolicy {
		return policy
	})
	if err != nil {
		return err
	}

	acPolicy, err := accessPolicyFromBlob(policy.Spec.Blob)
	if err != nil {
		m.logger.Errorf("failed decoding access policy %s", policy.Name)
		return err
	}
	return m.authzManager.AddAccessPolicy(connectivitypdp.PolicyFromCRD(acPolicy))
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

	acPolicy, err := accessPolicyFromBlob(policy.Spec.Blob)
	if err != nil {
		m.logger.Errorf("failed decoding access policy %s", policy.Name)
		return nil, err
	}

	polName := types.NamespacedName{Namespace: acPolicy.Namespace, Name: acPolicy.Name}
	if err := m.authzManager.DeleteAccessPolicy(polName, false); err != nil {
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
	apiPolicies := make([]*api.Policy, len(policies))
	for i, policy := range policies {
		apiPolicies[i] = accessPolicyToAPI(policy)
	}
	return apiPolicies, nil
}
