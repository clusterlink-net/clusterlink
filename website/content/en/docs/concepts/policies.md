---
title: Access Policies
description: Controlling service access across peers
weight: 40
---

Access policies allow users and administrators fine-grained control over
 which client workloads may access which service. This is an important security
 mechanism for applying [micro-segmentation](https://en.wikipedia.org/wiki/Microsegmentation_(network_security)),
 which is a basic requirement of [zero-trust](https://en.wikipedia.org/wiki/Zero_trust_security_model)
 systems. Another zero-trust principle, "Deny by default / Allow by exception", is also
 addressed by ClusterLink's access policies: a connection without an explicit policy allowing it,
 will be dropped. Access policies can also be used for enforcing corporate security rules,
 as well as segmenting the fabric into trust zones.

ClusterLink's access policies are based on attributes that are attached to
 [peers]({{< ref "peers" >}}), [services]({{< ref "services" >}}) and client workloads.
 Each attribute is a key:value pair, similar to how
 [labels](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/)
 are used in Kubernetes. This approach, called ABAC (Attribute Based Access Control),
 allows referring to a set of entities in a single policy, rather than listing individual
 entity names. Using attributes is safer, more resilient to changes, and easier to
 control and audit. At the moment, a limited set of attributes is available to use.
 We plan to enrich this set in the future.

Every instance of an access policy either allows or denies a given set of connections.
This set is defined by specifying the sources and destinations of these connections.
Sources are defined in terms of the attributes attached to the client workloads.
Destinations are defined in terms of the attributes attached to the target services.
Both client workloads and target services may inherit some attributes from their hosting peer.

There are two tiers of access policies in ClusterLink. The high-priority tier
 is intended for cluster/Peer administrators to set access rules which cannot be
 overridden by cluster users. High-priority policies are controlled by the
 `PrivilegedAccessPolicy` CRD, and are cluster-scoped (i.e., have no namespace).
 Regular policies are intended for cluster users, such as application developers
 and owners, and are controlled by the `AccessPolicy` CRD. Regular policies are
 namespaced, and have an effect in their namespace only. That is, they do not
 affect connections to/from other namespaces.

For a connection to be established, both the ClusterLink gateway on the client
 side and the ClusterLink gateway on the Service side must allow the connection.
 Each gateway (independently) follows these steps to decide if the connection is allowed:

1. All instances of `PrivilegedAccessPolicy` in the cluster with `deny` action are considered.
 If the connection matches any of them, the connection is dropped.
1. All instances of `PrivilegedAccessPolicy` in the cluster with `allow` action are considered.
 If the connection matches any of them, the connection is allowed.
1. All instances of `AccessPolicy` in the relevant namespace with `deny` action are considered.
 If the connection matches any of them, the connection is dropped.
1. All instances of `AccessPolicy` in the relevant namespace with `allow` action are considered.
 If the connection matches any of them, the connection is allowed.
1. If the connection matched none of the above policies, the connection is dropped.

**Note**: The relevant namespace for a given connection is the namespace of
 the corresponding Import CR on the client side and the namespace of the corresponding
 Export on the service side.

**Note**: Creating and deleting instances of `PrivilegedAccessPolicy` is currently not supported.

## Prerequisites

The following assumes that you have `kubectl` access to two or more clusters where ClusterLink
 has already been [deployed and configured]({{% ref "users#setup" %}}).

### Creating access policies

Recall that a connection is dropped if it does not match any access policy.
 Hence, for a connection to be allowed, an access policy with an `allow` action
 must be created on both sides of the connection.
 Creating an access policy is accomplished by creating an `AccessPolicy` CR in
 the relevant namespace (see Note above).

{{% expand summary="Export Custom Resource" %}}

```go
type AccessPolicy struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec AccessPolicySpec `json:"spec,omitempty"`
}

type AccessPolicySpec struct {
    Action AccessPolicyAction      `json:"action"`
    From WorkloadSetOrSelectorList `json:"from"`
    To WorkloadSetOrSelectorList   `json:"to"`
}

type AccessPolicyAction string

const (
    AccessPolicyActionAllow AccessPolicyAction = "allow"
    AccessPolicyActionDeny  AccessPolicyAction = "deny"
)

type WorkloadSetOrSelectorList []WorkloadSetOrSelector

type WorkloadSetOrSelector struct {
    WorkloadSets []string                  `json:"workloadSets,omitempty"`
    WorkloadSelector *metav1.LabelSelector `json:"workloadSelector,omitempty"`
}
```

{{% /expand %}}

The `AccessPolicySpec` defines the following fields:

- **Action** (string, required): whether the policy allows or denies the
 specified connection. Value must be either `allow` or `deny`.
- **From** (WorkloadSetOrSelector array, required): specifies connection sources.
 A connection's source must match one of the specified sources to be matched by the policy
- **To** (WorkloadSetOrSelectorList array, required): specifies connection destinations.
 A connection's destination must match one of the specified destinations to be matched by the policy

A `WorkloadSetOrSelector` object has two fields; exactly one of them must be specified.

- **WorkloadSets** (string array, optional) - an array of predefined sets of workload.
 Currently not supported.
- **WorkloadSelector** (LabelSelector, optional) - a Kubernetes
 [label selector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#labelselector-v1-meta)
 defining a set of client workloads or a set of services, based on their
 attributes. An empty selector matches all workloads/services.

The following policy allows all incoming/outgoing connections in the `default` namespace.

```yaml
apiVersion: clusterlink.net/v1alpha1
kind: AccessPolicy
metadata:
    name: allow-all
    namespace: default
spec:
    action: allow
    from:
    - workloadSelector: {}
    to:
    - workloadSelector: {}
```

More examples are available [here](https://github.com/clusterlink-net/clusterlink/tree/main/pkg/policyengine/examples)
