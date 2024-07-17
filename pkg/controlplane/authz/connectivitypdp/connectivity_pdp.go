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

package connectivitypdp

import (
	"fmt"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"

	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
)

// Decision represents an AccessPolicy decision on a given connection.
type Decision int

const (
	DecisionUndecided Decision = iota
	DecisionAllow
	DecisionDeny
)

// WorkloadAttrs are the actual key-value attributes attached to any given workload.
type WorkloadAttrs map[string]string

// PDP is the main object to maintain a set of access policies and decide
// whether a given connection is allowed or denied by these policies.
type PDP struct {
	privilegedPolicies policyTier
	regularPolicies    policyTier
}

// policyTier holds a set of AccessPolicies, split into deny policies and allow policies
// Within a tier, no two policies can have the same name, even if one is deny and the other is allow.
type policyTier struct {
	privileged    bool
	denyPolicies  connPolicyMap
	allowPolicies connPolicyMap
	lock          sync.RWMutex
}

type connPolicyMap map[types.NamespacedName]*v1alpha1.AccessPolicySpec // map from policy name to the policy

// DestinationDecision describes the PDP decision on a given destination (w.r.t, to a given source),
// including the deciding policy, if any.
// Calling PDP.Decide() with a source workload and a slice of destinations workloads,
// returns a slice of corresponding DestinationDecisions.
type DestinationDecision struct {
	Destination     WorkloadAttrs
	Decision        Decision
	MatchedBy       string // The name of the policy that matched the connection and took the decision
	PrivilegedMatch bool   // Whether the policy that took the decision was privileged
}

const DefaultDenyPolicyName = "<default deny>"

// NewPDP constructs a new PDP.
func NewPDP() *PDP {
	return &PDP{
		privilegedPolicies: newPolicyTier(true),
		regularPolicies:    newPolicyTier(false),
	}
}

// GetPrivilegedPolicies returns a slice of copies of the non-privileged policies stored in the PDP.
func (pdp *PDP) GetPrivilegedPolicies() []v1alpha1.PrivilegedAccessPolicy {
	pols := pdp.privilegedPolicies.getPolicies()
	res := []v1alpha1.PrivilegedAccessPolicy{}
	for polName, polSpec := range pols {
		res = append(res, v1alpha1.PrivilegedAccessPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: polName.Name, Namespace: polName.Namespace},
			Spec:       *polSpec,
		})
	}
	return res
}

// GetPolicies returns a slice of copies of the non-privileged policies stored in the PDP.
func (pdp *PDP) GetPolicies() []v1alpha1.AccessPolicy {
	pols := pdp.regularPolicies.getPolicies()
	res := []v1alpha1.AccessPolicy{}
	for polName, polSpec := range pols {
		res = append(res, v1alpha1.AccessPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: polName.Name, Namespace: polName.Namespace},
			Spec:       *polSpec,
		})
	}
	return res
}

// DependsOnClientAttrs returns whether the PDP holds a policy which depends on attributes
// the client workload (From field) may or may not have.
func (pdp *PDP) DependsOnClientAttrs() bool {
	return pdp.privilegedPolicies.dependsOnClientAttrs() ||
		pdp.regularPolicies.dependsOnClientAttrs()
}

// AddOrUpdatePolicy adds an AccessPolicy to the PDP.
// If a policy with the same name already exists in the PDP,
// it is updated (including updating the Action field).
// Invalid policies return an error.
func (pdp *PDP) AddOrUpdatePolicy(policy *AccessPolicy) error {
	if err := policy.spec.Validate(); err != nil {
		return err
	}

	if policy.privileged {
		pdp.privilegedPolicies.addPolicy(policy.name, &policy.spec)
	} else {
		pdp.regularPolicies.addPolicy(policy.name, &policy.spec)
	}
	return nil
}

// DeletePolicy deletes an AccessPolicy with the given name and privilege from the PDP.
// If no such AccessPolicy exists in the PDP, an error is returned.
func (pdp *PDP) DeletePolicy(policyName types.NamespacedName, privileged bool) error {
	if privileged {
		return pdp.privilegedPolicies.deletePolicy(policyName)
	}
	return pdp.regularPolicies.deletePolicy(policyName)
}

// Decide makes allow/deny decisions for the queried connection between src and dest.
// The decision, as well as the deciding policy, is recorded in the returned DestinationDecision struct.
func (pdp *PDP) Decide(src, dest WorkloadAttrs, ns string) (*DestinationDecision, error) {
	decision := DestinationDecision{Destination: dest}

	decided, err := pdp.privilegedPolicies.decide(src, &decision, ns)
	if err != nil {
		return nil, err
	}
	if decided {
		return &decision, nil
	}

	decided, err = pdp.regularPolicies.decide(src, &decision, ns)
	if err != nil {
		return nil, err
	}
	if decided {
		return &decision, nil
	}

	// for an undecided destination (no policy matched) set the default deny action
	decision.Decision = DecisionDeny
	decision.MatchedBy = DefaultDenyPolicyName
	decision.PrivilegedMatch = false
	return &decision, nil
}

func newPolicyTier(privileged bool) policyTier {
	return policyTier{
		privileged:    privileged,
		denyPolicies:  connPolicyMap{},
		allowPolicies: connPolicyMap{},
	}
}

func (pt *policyTier) getPolicies() connPolicyMap {
	res := pt.denyPolicies

	for key, val := range pt.allowPolicies {
		res[key] = val
	}
	return res
}

// dependsOnClientAttrs returns whether any of the tier's policies has a From field which depends on specific attributes.
func (pt *policyTier) dependsOnClientAttrs() bool {
	return pt.denyPolicies.dependsOnClientAttrs() || pt.allowPolicies.dependsOnClientAttrs()
}

// addPolicy adds an access policy to the given tier, based on its action.
// Note that within a tier, no two policies can have the same name, even if one is deny and the other is allow.
func (pt *policyTier) addPolicy(policyName types.NamespacedName, policySpec *v1alpha1.AccessPolicySpec) {
	pt.lock.Lock()
	defer pt.lock.Unlock()
	//nolint:errcheck // ignore return value as we just want to make sure non exists
	_ = pt.unsafeDeletePolicy(policyName) // delete an existing policy with the same name, if it exists
	if policySpec.Action == v1alpha1.AccessPolicyActionDeny {
		pt.denyPolicies[policyName] = policySpec
	} else {
		pt.allowPolicies[policyName] = policySpec
	}
}

// deletePolicy deletes a AccessPolicy with the given name from the given tier.
// If no such AccessPolicy exists in the tier, an error is returned.
func (pt *policyTier) deletePolicy(policyName types.NamespacedName) error {
	pt.lock.Lock()
	defer pt.lock.Unlock()
	return pt.unsafeDeletePolicy(policyName)
}

// unsafeDeletePolicy does the actual deleting of the given policy, but without locking.
// Do not use directly.
func (pt *policyTier) unsafeDeletePolicy(policyName types.NamespacedName) error {
	var okDeny, okAllow bool
	if _, okDeny = pt.denyPolicies[policyName]; okDeny {
		delete(pt.denyPolicies, policyName)
	}
	if _, okAllow = pt.allowPolicies[policyName]; okAllow {
		delete(pt.allowPolicies, policyName)
	}
	if !okDeny && !okAllow {
		return fmt.Errorf("failed deleting AccessPolicy %s", policyName)
	}
	return nil
}

// decide first checks whether any of the tier's deny policies matches any of the not-yet-decided connection
// between src and dest. If one policy does, the DestinationDecision will
// be updated to reflect the connection been denied.
// If the connection is not decided, the function then checks whether any of the tier's allow policies matches,
// and will similarly update the DestinationDecision.
// returns whether the destination was decided and an error (if occurred).
func (pt *policyTier) decide(src WorkloadAttrs, dest *DestinationDecision, ns string) (bool, error) {
	pt.lock.RLock() // allowing multiple simultaneous calls to decide() to be served
	defer pt.lock.RUnlock()
	decided, err := pt.denyPolicies.decide(src, dest, pt.privileged, ns)
	if err != nil {
		return false, err
	}
	if decided {
		return true, nil
	}

	decided, err = pt.allowPolicies.decide(src, dest, pt.privileged, ns)
	if err != nil {
		return false, err
	}
	return decided, nil
}

// decide iterates over all policies in a connPolicyMap and checks if they make a connectivity decision (allow/deny)
// on the not-yet-decided connection between src and dest.
// returns whether the destination was decided and an error (if occurred).
func (cpm connPolicyMap) decide(src WorkloadAttrs, dest *DestinationDecision, privileged bool, ns string) (bool, error) {
	// for when there are no policies in cpm (some destinations are undecided, otherwise we shouldn't be here)
	for policyName, policy := range cpm {
		if !privileged && policyName.Namespace != ns { // Only consider non-privileged policies from the given namespace
			continue
		}

		decision, err := accessPolicyDecide(policy, src, dest.Destination)
		if err != nil {
			return false, err
		}
		if decision != DecisionUndecided { // policy matched - we now have a decision for dest
			dest.Decision = decision
			dest.MatchedBy = policyName.String()
			dest.PrivilegedMatch = privileged
			return true, nil
		}
	}

	return false, nil
}

// dependsOnClientAttrs returns whether any of the policies has a From field which depends on specific attributes.
func (cpm connPolicyMap) dependsOnClientAttrs() bool {
	for _, policySpec := range cpm {
		for i := range policySpec.From {
			selector := policySpec.From[i].WorkloadSelector
			if selector == nil {
				continue
			}
			if len(selector.MatchExpressions) > 0 || len(selector.MatchLabels) > 0 {
				return true
			}
		}
	}
	return false
}

// accessPolicyDecide returns a policy's decision on a given connection.
// If the policy matches the connection, a decision based on its Action is returned.
// Otherwise, it returns an "undecided" value.
func accessPolicyDecide(policy *v1alpha1.AccessPolicySpec, src, dest WorkloadAttrs) (Decision, error) {
	matches, err := accessPolicyMatches(policy, src, dest)
	if err != nil {
		return DecisionDeny, err
	}
	if matches {
		if policy.Action == v1alpha1.AccessPolicyActionAllow {
			return DecisionAllow, nil
		}
		return DecisionDeny, nil
	}
	return DecisionUndecided, nil
}

// accessPolicyMatches checks if a connection from a source with given labels
// to a destination with given labels, matches an AccessPolicy.
func accessPolicyMatches(policy *v1alpha1.AccessPolicySpec, src, dest WorkloadAttrs) (bool, error) {
	// Check if source matches any element of the policy's "From" field
	matched, err := WorkloadSetOrSelectorListMatches(&policy.From, src)
	if err != nil {
		return false, err
	}
	if !matched {
		return false, nil
	}

	// Check if destination matches any element of the policy's "To" field
	matched, err = WorkloadSetOrSelectorListMatches(&policy.To, dest)
	if err != nil {
		return false, err
	}
	return matched, nil
}

// checks whether a workload with the given labels matches any item in a slice of WorkloadSetOrSelectors.
func WorkloadSetOrSelectorListMatches(wsl *v1alpha1.WorkloadSetOrSelectorList, workloadAttrs WorkloadAttrs) (bool, error) {
	for i := range *wsl {
		matched, err := workloadSetOrSelectorMatches(&(*wsl)[i], workloadAttrs)
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
func workloadSetOrSelectorMatches(wss *v1alpha1.WorkloadSetOrSelector, workloadAttrs WorkloadAttrs) (bool, error) {
	// TODO: implement logic for WorkloadSet matching
	selector, err := metav1.LabelSelectorAsSelector(wss.WorkloadSelector)
	if err != nil {
		return false, err
	}

	return selector.Matches(labels.Set(workloadAttrs)), nil
}
