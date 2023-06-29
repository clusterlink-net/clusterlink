package connectivitypdp

import (
	"fmt"
	"sync"

	"github.ibm.com/mbg-agent/pkg/policyEngine/policytypes"
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

// NewPDP constructs a new PDP
func NewPDP() *PDP {
	return &PDP{
		privilegedPolicies: newPolicyTier(),
		regularPolicies:    newPolicyTier(),
	}
}

// Returns a slice of copies of the policies stored in the PDP
func (pdp *PDP) GetPolicies() []policytypes.ConnectivityPolicy {
	return append(pdp.privilegedPolicies.getPolicies(), pdp.regularPolicies.getPolicies()...)
}

// AddOrUpdatePolicy adds a ConnectivityPolicy to the PDP.
// If a policy with the same name and the same privilege already exists in the PDP, it is updated (including updating the Action field).
// Invalid policies return an error.
func (pdp *PDP) AddOrUpdatePolicy(policy policytypes.ConnectivityPolicy) error {
	if err := policy.Validate(); err != nil {
		return err
	}

	if policy.Privileged {
		pdp.privilegedPolicies.addPolicy(&policy)
	} else {
		pdp.regularPolicies.addPolicy(&policy)
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

// Decide returns Allow/Deny for a given connection, based on the set of policies the PDP holds.
// src and dest are sets of attributes (labels) of the source and destination respectively.
func (pdp *PDP) Decide(src, dest policytypes.WorkloadAttrs) (policytypes.PolicyAction, error) {
	matched, decision, err := pdp.privilegedPolicies.decide(src, dest)
	if err != nil {
		return policytypes.PolicyActionDeny, err
	}
	if matched {
		return decision, nil
	}

	matched, decision, err = pdp.regularPolicies.decide(src, dest)
	if err != nil {
		return policytypes.PolicyActionDeny, err
	}
	if matched {
		return decision, nil
	}

	return policytypes.PolicyActionDeny, nil // no policy matched; default is deny
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
	_ = pt.unsafeDeletePolicy(policy.Name) // delete an existing policy with the same name, if it exists
	if policy.Action == policytypes.PolicyActionDeny {
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
// Do not use directly
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

// returns a triplet: whether a policy matched the inputs, the matching policy action, and an error (if occurred)
func (pt *policyTier) decide(src, dest policytypes.WorkloadAttrs) (bool, policytypes.PolicyAction, error) {
	pt.lock.RLock() // allowing multiple simultaneous calls to decide() to be served
	defer pt.lock.RUnlock()
	matched, err := pt.denyPolicies.matches(src, dest)
	if err != nil {
		return false, policytypes.PolicyActionDeny, err
	}
	if matched {
		return true, policytypes.PolicyActionDeny, nil
	}

	matched, err = pt.allowPolicies.matches(src, dest)
	if err != nil {
		return false, policytypes.PolicyActionDeny, err
	}
	if matched {
		return true, policytypes.PolicyActionAllow, nil
	}
	return false, policytypes.PolicyActionDeny, nil
}

func (cpm connPolicyMap) getPolicies() []policytypes.ConnectivityPolicy {
	res := []policytypes.ConnectivityPolicy{}
	for _, p := range cpm {
		res = append(res, *p)
	}
	return res
}

func (cpm connPolicyMap) matches(src, dest policytypes.WorkloadAttrs) (bool, error) {
	for _, policy := range cpm {
		matched, err := policy.Matches(src, dest)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}
