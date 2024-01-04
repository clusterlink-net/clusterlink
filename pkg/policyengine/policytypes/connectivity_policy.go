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

package policytypes

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// ConnectivityPolicy defines whether a group of potential connections should be allowed or denied.
// If multiple ConnectivityPolicies match a given connection, privileged policies
// take precedence over non-privileged, and within each tier deny policies take
// precedence over allow policies.
type ConnectivityPolicy struct {
	Name       string                    `json:"name"`
	Privileged bool                      `json:"privileged"`
	Action     PolicyAction              `json:"action"`
	From       WorkloadSetOrSelectorList `json:"from"`
	To         WorkloadSetOrSelectorList `json:"to"`
}

// PolicyAction specifies whether a ConnectivityPolicy allows or denies the connection specified by its 'From' and 'To' fields.
type PolicyAction string

const (
	PolicyActionAllow PolicyAction = "allow"
	PolicyActionDeny  PolicyAction = "deny"
)

// PolicyDecision represents a ConnectivityPolicy decision on a given connection.
type PolicyDecision int

const (
	PolicyDecisionUndecided PolicyDecision = iota
	PolicyDecisionAllow
	PolicyDecisionDeny
)

// WorkloadSetOrSelectorList is a collection of WorkloadSetOrSelector objects.
type WorkloadSetOrSelectorList []WorkloadSetOrSelector

// WorkloadSetOrSelector describes a set of workloads, based on their attributes (labels)
// Exactly one of the two fields should be non-empty.
type WorkloadSetOrSelector struct {
	//nolint:tagliatelle // use camelCase, same as k8s convention
	WorkloadSets []string `json:"workloadSets,omitempty"`
	//nolint:tagliatelle // use camelCase, same as k8s convention
	WorkloadSelector *metav1.LabelSelector `json:"workloadSelector,omitempty"`
}

// WorkloadAttrs are the actual key-value attributes attached to any given workload.
type WorkloadAttrs map[string]string

// Validate returns an error if the given ConnectivityPolicy is invalid. Otherwise, returns nil.
func (cps *ConnectivityPolicy) Validate() error {
	if cps.Action != PolicyActionAllow && cps.Action != PolicyActionDeny {
		return fmt.Errorf("unsupported policy actions %s", cps.Action)
	}
	if len(cps.From) == 0 {
		return fmt.Errorf("empty From field is not allowed")
	}
	if err := cps.From.validate(); err != nil {
		return err
	}
	if len(cps.To) == 0 {
		return fmt.Errorf("empty To field is not allowed")
	}
	return cps.To.validate()
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

// Decide returns the receiver policy's decision on a given connection.
// If the policy matches the connection, a decision based on its Action is returned.
// Otherwise, it returns an "undecided" value.
func (cps *ConnectivityPolicy) Decide(src, dest WorkloadAttrs) (PolicyDecision, error) {
	matches, err := cps.Matches(src, dest)
	if err != nil {
		return PolicyDecisionDeny, err
	}
	if matches {
		if cps.Action == PolicyActionAllow {
			return PolicyDecisionAllow, nil
		}
		return PolicyDecisionDeny, nil
	}
	return PolicyDecisionUndecided, nil
}

// Matches checks if a connection from a source with given labels to a destination with given labels,
// matches a ConnectivityPolicy.
func (cps *ConnectivityPolicy) Matches(src, dest WorkloadAttrs) (bool, error) {
	// Check if source matches any element of the policy's "From" field
	matched, err := cps.From.matches(src)
	if err != nil {
		return false, err
	}
	if !matched {
		return false, nil
	}

	// Check if destination matches any element of the policy's "To" field
	matched, err = cps.To.matches(dest)
	if err != nil {
		return false, err
	}
	return matched, nil
}

// checks whether a workload with the given labels matches any item in a slice of WorkloadSetOrSelectors.
func (wsl WorkloadSetOrSelectorList) matches(workloadAttrs WorkloadAttrs) (bool, error) {
	for _, workloadSet := range wsl {
		matched, err := workloadSet.matches(workloadAttrs)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

// checks whether a workload with the given labels matches a WorkloadSetOrSelectors.
func (wss *WorkloadSetOrSelector) matches(workloadAttrs WorkloadAttrs) (bool, error) {
	// TODO: implement logic for WorkloadSet matching
	selector, err := metav1.LabelSelectorAsSelector(wss.WorkloadSelector)
	if err != nil {
		return false, err
	}

	return selector.Matches(labels.Set(workloadAttrs)), nil
}
