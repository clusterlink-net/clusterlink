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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true

// AccessPolicy defines whether a group of potential connections should be allowed or denied.
// If multiple AccessPolicy objects match a given connection, privileged policies
// take precedence over non-privileged, and within each tier deny policies take
// precedence over allow policies.
type AccessPolicy struct {
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

// AccessPolicySpec contains all attributes of an access policy.
type AccessPolicySpec struct {
	// Privileged is true if the policy has higher priority over non-privileged policies.
	Privileged bool `json:"privileged"`
	// Action specifies whether the policy allows or denies connections matching its From and To fields.
	Action AccessPolicyAction `json:"action"`
	// From specifies the set of source workload to which this policy refers.
	From WorkloadSetOrSelectorList `json:"from"`
	// To specifies the set of destination services to which this policy refers.
	To WorkloadSetOrSelectorList `json:"to"`
}

// +kubebuilder:object:root=true

// AccessPolicyList is a list of AccessPolicy objects.
type AccessPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of access policy objects.
	Items []AccessPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AccessPolicy{}, &AccessPolicyList{})
}
