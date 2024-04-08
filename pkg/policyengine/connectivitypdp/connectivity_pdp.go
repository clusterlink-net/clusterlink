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

// Returns a slice of copies of the non-privileged policies stored in the PDP.
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

// Returns a slice of copies of the non-privileged policies stored in the PDP.
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

// Decide makes allow/deny decisions for the queried connections between src and each of destinations in dests.
// The decision, as well as the deciding policy, are recorded in the returned slice of DestinationDecision structs.
// The order of destinations in dests is preserved in the returned slice.
func (pdp *PDP) Decide(src WorkloadAttrs, dests []WorkloadAttrs, ns string) ([]DestinationDecision, error) {
	decisions := make([]DestinationDecision, len(dests))
	for i, dest := range dests {
		decisions[i] = DestinationDecision{Destination: dest}
	}

	allDestsDecided, err := pdp.privilegedPolicies.decide(src, decisions, ns)
	if err != nil {
		return nil, err
	}
	if allDestsDecided {
		return decisions, nil
	}

	allDestsDecided, err = pdp.regularPolicies.decide(src, decisions, ns)
	if err != nil {
		return nil, err
	}
	if allDestsDecided {
		return decisions, nil
	}

	// For all undecided destination (for which no policy matched) set the default deny action
	denyUndecidedDestinations(decisions)
	return decisions, nil
}

func denyUndecidedDestinations(dest []DestinationDecision) {
	for i := range dest {
		dd := &dest[i]
		if dd.Decision == DecisionUndecided {
			dd.Decision = DecisionDeny
			dd.MatchedBy = DefaultDenyPolicyName
			dd.PrivilegedMatch = false
		}
	}
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

// decide first checks whether any of the tier's deny policies matches any of the not-yet-decided connections
// between src and each of the destinations in dests. If one policy does, the relevant DestinationDecision will
// be updated to reflect the connection been denied.
// The function then checks whether any of the tier's allow policies matches any of the remaining undecided connections,
// and will similarly update the relevant DestinationDecision of any matching connection.
// returns whether all destinations were decided and an error (if occurred).
func (pt *policyTier) decide(src WorkloadAttrs, dests []DestinationDecision, ns string) (bool, error) {
	pt.lock.RLock() // allowing multiple simultaneous calls to decide() to be served
	defer pt.lock.RUnlock()
	allDecided, err := pt.denyPolicies.decide(src, dests, pt.privileged, ns)
	if err != nil {
		return false, err
	}
	if allDecided {
		return true, nil
	}

	allDecided, err = pt.allowPolicies.decide(src, dests, pt.privileged, ns)
	if err != nil {
		return false, err
	}
	return allDecided, nil
}

// decide iterates over all policies in a connPolicyMap and checks if they make a connectivity decision (allow/deny)
// on the not-yet-decided connections between src and each of the destinations in dests.
// returns whether all destinations were decided and an error (if occurred).
func (cpm connPolicyMap) decide(src WorkloadAttrs, dests []DestinationDecision, privileged bool, ns string) (bool, error) {
	// for when there are no policies in cpm (some destinations are undecided, otherwise we shouldn't be here)
	allDecided := false
	for policyName, policy := range cpm {
		if !privileged && policyName.Namespace != ns { // Only consider non-privileged policies from the given namespace
			continue
		}
		allDecided = true // assume all destinations were decided, unless we find a destination which is not
		for i := range dests {
			dest := &dests[i]
			if dest.Decision == DecisionUndecided {
				decision, err := accessPolicyDecide(policy, src, dest.Destination)
				if err != nil {
					return false, err
				}
				if decision == DecisionUndecided {
					allDecided = false // policy didn't match dest - not all dests are decided
				} else { // policy matched - we now have a decision for dest
					dest.Decision = decision
					dest.MatchedBy = policyName.String()
					dest.PrivilegedMatch = privileged
				}
			}
		}
		if allDecided {
			break
		}
	}
	return allDecided, nil
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
