package k8sshim

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.ibm.com/mbg-agent/pkg/policyEngine/policytypes"
)

const (
	GroupName                        = "clusterlink"
	Version                          = "v1alpha1"
	PrivilegedConnectivityPolicyKind = "PrivilegedConnectivityPolicy"
	ConnectivityPolicyKind           = "ConnectivityPolicy"
)

// PrivilegedConnectivityPolicy represents a high-priority connectivity policy which takes precedence
// over a regular connectivity policy. It defines allowed/denied connectivity between two sets of workloads.
// Among all instances of PrivilegedConnectivityPolicy, instances with Spec.Action==PolicyActionDeny take
// precedence over instances with Spec.Action==PolicyActionAllow
type PrivilegedConnectivityPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ConnectivityPolicySpec `json:"spec"`
}

// ConnectivityPolicy represents a lower-priority connectivity policy.
// It defines allowed/denied connectivity between two sets of workloads.
// Among all instances of ConnectivityPolicy, instances with Spec.Action==PolicyActionDeny take
// precedence over instances with Spec.Action==PolicyActionAllow
type ConnectivityPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ConnectivityPolicySpec `json:"spec"`
}

// ConnectivityPolicySpec is a common spec for both PrivilegedConnectivityPolicy and ConnectivityPolicy
type ConnectivityPolicySpec struct {
	Action          policytypes.PolicyAction
	From            policytypes.WorkloadSetOrSelectorList `json:"from"`
	To              policytypes.WorkloadSetOrSelectorList `json:"to"`
	ConnectionAttrs []ConnectionAttrs                     `json:"connectionAttrs,omitempty"`
}

// ConnectionAttrs describes the combination of protocol and port used by a given connection
type ConnectionAttrs struct {
	Protocol string `json:"protocol"`       // TODO: only string or also int?
	Port     *int32 `json:"port,omitempty"` // if set to nil, all ports are allowed
}

// ToInternal converts a PrivilegedConnectivityPolicy into the built-in (non-k8s) ConnectivityPolicy type
func (pcp *PrivilegedConnectivityPolicy) ToInternal() *policytypes.ConnectivityPolicy {
	return &policytypes.ConnectivityPolicy{
		Name:       pcp.Name,
		Privileged: true,
		Action:     pcp.Spec.Action,
		From:       pcp.Spec.From,
		To:         pcp.Spec.To,
	}
}

// ToInternal converts a ConnectivityPolicy into the built-in (non-k8s) ConnectivityPolicy type
func (pcp *ConnectivityPolicy) ToInternal() *policytypes.ConnectivityPolicy {
	return &policytypes.ConnectivityPolicy{
		Name:       pcp.Name,
		Privileged: false,
		Action:     pcp.Spec.Action,
		From:       pcp.Spec.From,
		To:         pcp.Spec.To,
	}
}
