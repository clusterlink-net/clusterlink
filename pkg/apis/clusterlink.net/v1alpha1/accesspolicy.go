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

// Copyright (c) 2022 The ClusterLink Authors.
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

// Copyright (C) The ClusterLink Authors.
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

package v1alpha1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced

// AccessPolicy defines whether a set of connections should be allowed or denied.
// If multiple AccessPolicy objects match a given connection, deny policies take
// precedence over allow policies.
type AccessPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec represents the attributes of the exported service.
	Spec AccessPolicySpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// PrivilegedAccessPolicy is the cluster-scoped version of AccessPolicy.
// PrivilegedAccessPolicies are intended to be used by cluster admins, and take precedence over AccessPolicies.
// Within each tier, deny policies take precedence over allow policies.
type PrivilegedAccessPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec represents the attributes of the exported service.
	Spec AccessPolicySpec `json:"spec,omitempty"`
}

// AccessPolicyAction specifies whether an AccessPolicy allows or denies
// the connection specified by its 'From' and 'To' fields.
type AccessPolicyAction string

const (
	AccessPolicyActionAllow AccessPolicyAction = "allow"
	AccessPolicyActionDeny  AccessPolicyAction = "deny"
)

// WorkloadSetOrSelectorList is a collection of WorkloadSetOrSelector objects.
type WorkloadSetOrSelectorList []WorkloadSetOrSelector

// WorkloadSetOrSelector describes a set of workloads, based on their attributes (labels).
// Exactly one of the two fields should be non-empty.
type WorkloadSetOrSelector struct {
	// WorkloadSets allows specifying predefined sets of workloads - not yet supported.
	WorkloadSets []string `json:"workloadSets,omitempty"`
	// WorkloadSelector is a K8s-style label selector, selecting Pods and Services according to their labels.
	WorkloadSelector *metav1.LabelSelector `json:"workloadSelector,omitempty"`
}

// AccessPolicySpec specifies the connections AccessPolicy and PrivilegedAccessPolicy make decisions on
// as well as the policy's decision on these connection.
type AccessPolicySpec struct {
	// Action specifies whether the policy allows or denies connections matching its From and To fields.
	Action AccessPolicyAction `json:"action"`
	// From specifies the set of source workload to which this policy refers.
	From WorkloadSetOrSelectorList `json:"from"`
	// To specifies the set of destination services to which this policy refers.
	To WorkloadSetOrSelectorList `json:"to"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// PrivilegedAccessPolicyList is a list of PrivilegedAccessPolicyList objects.
type PrivilegedAccessPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of access policy objects.
	Items []PrivilegedAccessPolicy `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced

// AccessPolicyList is a list of AccessPolicy objects.
type AccessPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of access policy objects.
	Items []AccessPolicy `json:"items"`
}

// Validate returns an error if the given AccessPolicy is invalid. Otherwise, returns nil.
func (p *AccessPolicySpec) Validate() error {
	if p.Action != AccessPolicyActionAllow && p.Action != AccessPolicyActionDeny {
		return fmt.Errorf("unsupported policy actions %s", p.Action)
	}
	if len(p.From) == 0 {
		return fmt.Errorf("empty From field is not allowed")
	}
	if err := p.From.validate(); err != nil {
		return err
	}
	if len(p.To) == 0 {
		return fmt.Errorf("empty To field is not allowed")
	}
	return p.To.validate()
}

func (wsl WorkloadSetOrSelectorList) validate() error {
	for i := range wsl {
		if err := wsl[i].validate(); err != nil {
			return err
		}
	}

	return nil
}

func (wss *WorkloadSetOrSelector) validate() error {
	if len(wss.WorkloadSets) > 0 && wss.WorkloadSelector != nil ||
		len(wss.WorkloadSets) == 0 && wss.WorkloadSelector == nil {
		return fmt.Errorf("exactly one of WorkloadSets or WorkloadSelector must be set")
	}
	if len(wss.WorkloadSets) > 0 {
		return fmt.Errorf("workload sets are not yet supported")
	}
	_, err := metav1.LabelSelectorAsSelector(wss.WorkloadSelector)
	return err
}

func init() {
	SchemeBuilder.Register(&AccessPolicy{}, &PrivilegedAccessPolicy{}, &AccessPolicyList{}, &PrivilegedAccessPolicyList{})
}
