---
title: Access Policies
description: Controlling service access across peers
weight: 40
---

Access policies allow users and administrators fine-grained control over
 which client workloads may access which service. This is an important security
 mechanism for applying [micro-segmentation][], which is a basic requirement of [zero-trust][]
 systems. Another zero-trust principle, "Deny by default / Allow by exception," is also
 addressed by ClusterLink's access policies: a connection without an explicit policy allowing it
 will be dropped. Access policies can also be used for enforcing corporate security rules,
 as well as segmenting the fabric into trust zones.

ClusterLink's access policies are based on attributes that are attached to
 [peers][], [services][] and client workloads.
 Each attribute is a key/value pair, similar to how [labels][]
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
 is intended for cluster/peer administrators to set access rules which cannot be
 overridden by cluster users. High-priority policies are controlled by the
 `PrivilegedAccessPolicy` CRD, and are cluster scoped (i.e., have no namespace).
 Regular policies are intended for cluster users, such as application developers
 and owners, and are controlled by the `AccessPolicy` CRD. Regular policies are
 namespaced, and have an effect in their namespace only. That is, they do not
 affect connections to/from other namespaces.

For a connection to be established, both the ClusterLink gateway on the client
 side and the ClusterLink gateway on the service side must allow the connection.
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

## Prerequisites

The following assumes that you have `kubectl` access to two or more clusters where ClusterLink
 has already been [deployed and configured][].

### Creating access policies

Recall that a connection is dropped if it does not match any access policy.
 Hence, for a connection to be allowed, an access policy with an `allow` action
 must be created on both sides of the connection.
 Creating an access policy is accomplished by creating an `AccessPolicy` CR in
 the relevant namespace (see note above).
 Creating a high-priority access policy is accomplished by creating a `PrivilegedAccessPolicy` CR.
 Instances of `PrivilegedAccessPolicy` have no namespace and affect the entire cluster.

{{% expand summary="PrivilegedAccessPolicy and AccessPolicy Custom Resources" %}}

```go
type PrivilegedAccessPolicy struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec AccessPolicySpec `json:"spec,omitempty"`
}

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
 A connection's source must match one of the specified sources to be matched by the policy.
- **To** (WorkloadSetOrSelectorList array, required): specifies connection destinations.
 A connection's destination must match one of the specified destinations to be matched by the policy.

A `WorkloadSetOrSelector` object has two fields; exactly one of them must be specified.

- **WorkloadSets** (string array, optional) - an array of predefined sets of workload.
 Currently not supported.
- **WorkloadSelector** (LabelSelector, optional) - a [Kubernetes label selector][]
 defining a set of client workloads or a set of services, based on their
 attributes. An empty selector matches all workloads/services.

### Example policies
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

The following privileged policy denies incoming/outgoing connections originating from a cluster with a Peer named `testing`.
```yaml
apiVersion: clusterlink.net/v1alpha1
kind: PrivilegedAccessPolicy
metadata:
    name: deny-from-testing
spec:
    action: deny
    from:
    - workloadSelector:
        matchLabels:
            peer.clusterlink.net/name: testing
    to:
    - workloadSelector: {}
```

More examples are available on our repo under [examples/policies][].

### Available attributes
The following attributes (labels) are set by ClusterLink on each connection request, and can be used in access policies within a `workloadSelector`.
#### Peer attributes - set when running `clusterlink deploy peer`
* `peer.clusterlink.net/name` - Peer name as set by the `--name` flag
* `peer.clusterlink.net/labels.<label-key>` - Peer's labels, set by using `--label` flags
#### Client attributes - derived from Pod info, as retrieved from Kubernetes API. Only relevant in the `from` section of access policies
* `client.clusterlink.net/namespace` - Pod's Namespace
* `client.clusterlink.net/service-account` - Pod's Service Account
* `client.clusterlink.net/labels.<label-key>` - Pod's labels - an attribute for each Pod label with key `<label-key>`
#### Service attributes - derived from the Export CR. Only relevant in the `to` section of access policies
* `export.clusterlink.net/name` - Export name
* `export.clusterlink.net/namespace` - Export namespace

[peers]: {{< relref "peers" >}}
[services]: {{< relref "services" >}}
[micro-segmentation]: https://en.wikipedia.org/wiki/Microsegmentation_(network_security)
[zero-trust]: https://en.wikipedia.org/wiki/Zero_trust_security_model
[labels]: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
[deployed and configured]: {{< relref "../getting-started/users#setup" >}}
[Kuberenetes label selector]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#labelselector-v1-meta
[examples/policies]: https://github.com/clusterlink-net/clusterlink/tree/main/examples/policies
