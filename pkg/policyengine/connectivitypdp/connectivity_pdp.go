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

	"github.com/clusterlink-net/clusterlink/pkg/policyengine/policytypes"
)

// PDP is the main object to maintain a set of connectivity policies and decide
// whether a given connection is allowed or denied by these policies.
type PDP struct {
	privilegedPolicies policyTier
	regularPolicies    policyTier
}

// policyTier holds a set of ConnectivityPolicies, split into deny policies and allow policies
// Within a tier, no two policies can have the same name, even if one is deny and the other is allow.
type policyTier struct {
	denyPolicies  connPolicyMap
	allowPolicies connPolicyMap
	lock          sync.RWMutex
}

type connPolicyMap map[string]*policytypes.ConnectivityPolicy // map from policy name to the policy

// DestinationDecision describes the PDP decision on a given destination (w.r.t, to a given source),
// including the deciding policy, if any.
// Calling PDP.Decide() with a source workload and a slice of destinations workloads,
// returns a slice of corresponding DestinationDecisions.
type DestinationDecision struct {
	Destination     policytypes.WorkloadAttrs
	Decision        policytypes.PolicyDecision
	MatchedBy       string // The name of the policy that matched the connection and took the decision
	PrivilegedMatch bool   // Whether the policy that took the decision was privileged
}

const DefaultDenyPolicyName = "<default deny>"

// NewPDP constructs a new PDP.
func NewPDP() *PDP {
	return &PDP{
		privilegedPolicies: newPolicyTier(),
		regularPolicies:    newPolicyTier(),
	}
}

// Returns a slice of copies of the policies stored in the PDP.
func (pdp *PDP) GetPolicies() []policytypes.ConnectivityPolicy {
	return append(pdp.privilegedPolicies.getPolicies(), pdp.regularPolicies.getPolicies()...)
}

// AddOrUpdatePolicy adds a ConnectivityPolicy to the PDP.
// If a policy with the same name and the same privilege already exists in the PDP,
// it is updated (including updating the Action field).
// Invalid policies return an error.
func (pdp *PDP) AddOrUpdatePolicy(policy *policytypes.ConnectivityPolicy) error {
	if err := policy.Validate(); err != nil {
		return err
	}

	if policy.Privileged {
		pdp.privilegedPolicies.addPolicy(policy)
	} else {
		pdp.regularPolicies.addPolicy(policy)
	}
	return nil
}

// DeletePolicy deletes a ConnectivityPolicy with the given name and privilege from the PDP.
// If no such ConnectivityPolicy exists in the PDP, an error is returned.
func (pdp *PDP) DeletePolicy(policyName string, privileged bool) error {
	if privileged {
		return pdp.privilegedPolicies.deletePolicy(policyName)
	}
	return pdp.regularPolicies.deletePolicy(policyName)
}

// Decide makes allow/deny decisions for the queried connections between src and each of destinations in dests.
// The decision, as well as the deciding policy, are recorded in the returned slice of DestinationDecision structs.
// The order of destinations in dests is preserved in the returned slice.
func (pdp *PDP) Decide(src policytypes.WorkloadAttrs, dests []policytypes.WorkloadAttrs) ([]DestinationDecision, error) {
	decisions := make([]DestinationDecision, len(dests))
	for i, dest := range dests {
		decisions[i] = DestinationDecision{Destination: dest}
	}

	allDestsDecided, err := pdp.privilegedPolicies.decide(src, decisions)
	if err != nil {
		return nil, err
	}
	if allDestsDecided {
		return decisions, nil
	}

	allDestsDecided, err = pdp.regularPolicies.decide(src, decisions)
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
		if dd.Decision == policytypes.DecisionUndecided {
			dd.Decision = policytypes.DecisionDeny
			dd.MatchedBy = DefaultDenyPolicyName
			dd.PrivilegedMatch = false
		}
	}
}

func newPolicyTier() policyTier {
	return policyTier{
		denyPolicies:  connPolicyMap{},
		allowPolicies: connPolicyMap{},
	}
}

func (pt *policyTier) getPolicies() []policytypes.ConnectivityPolicy {
	return append(pt.denyPolicies.getPolicies(), pt.allowPolicies.getPolicies()...)
}

// addPolicy adds a ConnectivityPolicy to the given tier, based on its action.
// Note that within a tier, no two policies can have the same name, even if one is deny and the other is allow.
func (pt *policyTier) addPolicy(policy *policytypes.ConnectivityPolicy) {
	pt.lock.Lock()
	defer pt.lock.Unlock()
	//nolint:errcheck // ignore return value as we just want to make sure non exists
	_ = pt.unsafeDeletePolicy(policy.Name) // delete an existing policy with the same name, if it exists
	if policy.Action == policytypes.ActionDeny {
		pt.denyPolicies[policy.Name] = policy
	} else {
		pt.allowPolicies[policy.Name] = policy
	}
}

// deletePolicy deletes a ConnectivityPolicy with the given name from the given tier.
// If no such ConnectivityPolicy exists in the tier, an error is returned.
func (pt *policyTier) deletePolicy(policyName string) error {
	pt.lock.Lock()
	defer pt.lock.Unlock()
	return pt.unsafeDeletePolicy(policyName)
}

// unsafeDeletePolicy does the actual deleting of the given policy, but without locking.
// Do not use directly.
func (pt *policyTier) unsafeDeletePolicy(policyName string) error {
	var okDeny, okAllow bool
	if _, okDeny = pt.denyPolicies[policyName]; okDeny {
		delete(pt.denyPolicies, policyName)
	}
	if _, okAllow = pt.allowPolicies[policyName]; okAllow {
		delete(pt.allowPolicies, policyName)
	}
	if !okDeny && !okAllow {
		return fmt.Errorf("failed deleting ConnectivityPolicy %s", policyName)
	}
	return nil
}

// decide first checks whether any of the tier's deny policies matches any of the not-yet-decided connections
// between src and each of the destinations in dests. If one policy does, the relevant DestinationDecision will
// be updated to reflect the connection been denied.
// The function then checks whether any of the tier's allow policies matches any of the remaining undecided connections,
// and will similarly update the relevant DestinationDecision of any matching connection.
// returns whether all destinations were decided and an error (if occurred).
func (pt *policyTier) decide(src policytypes.WorkloadAttrs, dests []DestinationDecision) (bool, error) {
	pt.lock.RLock() // allowing multiple simultaneous calls to decide() to be served
	defer pt.lock.RUnlock()
	allDecided, err := pt.denyPolicies.decide(src, dests)
	if err != nil {
		return false, err
	}
	if allDecided {
		return true, nil
	}

	allDecided, err = pt.allowPolicies.decide(src, dests)
	if err != nil {
		return false, err
	}
	return allDecided, nil
}

func (cpm connPolicyMap) getPolicies() []policytypes.ConnectivityPolicy {
	res := []policytypes.ConnectivityPolicy{}
	for _, p := range cpm {
		res = append(res, *p)
	}
	return res
}

// decide iterates over all policies in a connPolicyMap and checks if they make a connectivity decision (allow/deny)
// on the not-yet-decided connections between src and each of the destinations in dests.
// returns whether all destinations were decided and an error (if occurred).
func (cpm connPolicyMap) decide(src policytypes.WorkloadAttrs, dests []DestinationDecision) (bool, error) {
	// for when there are no policies in cpm (some destinations are undecided, otherwise we shouldn't be here)
	allDecided := false
	for _, policy := range cpm {
		allDecided = true // assume all destinations were decided, unless we find a destination which is not
		for i := range dests {
			dest := &dests[i]
			if dest.Decision == policytypes.DecisionUndecided {
				decision, err := policy.Decide(src, dest.Destination)
				if err != nil {
					return false, err
				}
				if decision == policytypes.DecisionUndecided {
					allDecided = false // policy didn't match dest - not all dests are decided
				} else { // policy matched - we now have a decision for dest
					dest.Decision = decision
					dest.MatchedBy = policy.Name
					dest.PrivilegedMatch = policy.Privileged
				}
			}
		}
		if allDecided {
			break
		}
	}
	return allDecided, nil
}
