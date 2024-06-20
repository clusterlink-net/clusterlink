---
title: Services
description: Sharing services
weight: 30
---

ClusterLink uses services as the unit of sharing between peers.
 One or more peers can expose an (internal) K8s Service to
 be consumed by other [peers][] in the [fabric][].
 A service is exposed by creating an *Export* CR referencing it in the
 source cluster. Similarly, the exported service can be made accessible to workloads
 in a peer by defining an *Import* CR in the destination cluster[^KEP-1645].
 Thus, service sharing is an explicit operation. Services are not automatically
 shared by peers in the fabric. Note that the exporting cluster must be
 [configured as a peer][] of the importing cluster.

{{< notice info >}}
Services sharing is done on a per namespace basis and does not require cluster wide privileges.
 It is intended to be used by application owners having access to their own namespaces only.
{{< /notice >}}

A service is shared using a logical name. The logical name does not have to match
 the actual Kubernetes Service name in the exporting cluster. Exporting a service
 does not expose cluster Pods or their IP addresses to the importing clusters.
 Any load balancing and scaling decisions are kept local in the exporting cluster.
 This reduces the amount, frequency and sensitivity of information shared between
 clusters. Similarly, the imported service can have any arbitrary name in the
 destination cluster, allowing independent choice of naming.

Orchestration of service sharing is the responsibility of users wishing to
 export or import it, and any relevant information (e.g, the exported service
 name and namespace) must be communicated out of band. In the future, this could
 be done by a centralized management plane.

<!-- TODO: image showing export/import (from >2 clusters?) -->

<!--
 TODO centralized management may apply simplification of sharing via policies
  (e.g., auto-expose)
  can also simplify some via clusterlink cli (e.g., allow-all policy default)
  Ask for inputs through the documentation?
  Put all of the above in footnote?
-->

## Prerequisites

The following assume that you have `kubectl` access to two or more clusters where ClusterLink
 has already been [deployed and configured][].

### Exporting a service

In order to make a service potentially accessible by other clusters, it must be
 explicitly configured for remote access via ClusterLink. Exporting is
 accomplished by creating an Export CR in the **same** namespace
 as the service being exposed. The CR acts as a marker for enabling
 remote access to the service via ClusterLink.

{{% expand summary="Export Custom Resource" %}}

```go
type Export struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec ExportSpec     `json:"spec,omitempty"`
    Status ExportStatus `json:"status,omitempty"`
}

type ExportSpec struct {
    Host string `json:"host,omitempty"`
    Port uint16 `json:"port,omitempty"`
}

type ExportStatus struct {
    Conditions []metav1.Condition `json:"conditions,omitempty"`
}
```

{{% /expand %}}

The ExportSpec defines the following fields:

- **Host** (string, optional): the name of the service being exported. The service
 must be defined in the same namespace as the Export CR. If empty,
 the export shall refer to a Kubernetes Service with the same name as the instance's
 `metadata.name`. It is an error to refer to a non-existent service or one that is
 not present in the local namespace. The error will be reflected in the CRD's status.
- **Port** (integer, required): the port number being exposed. If you wish to export
 a multi-port service[^multiport], you will need to define multiple Exports using
 the same `Host` value and a different `Port` each. This is aligned with ClusterLink's
 principle of being explicit in sharing and limiting exposure whenever possible.

Note that exporting a Service does not automatically make is accessible to other
 peers, but only enables *potential* access. To complete service sharing, you must
 define at least one [access control policy][concept-policy] that allows
 access in the exporting cluster.
 In addition, users in consuming clusters must still explicitly configure
 [service imports][] and [policies][] in their respective namespaces.

{{% expand summary="Example YAML for `kubectl apply -f <export_file>`" %}}

```yaml
apiVersion: clusterlink.net/v1alpha1
kind: Export
metadata:
  name: iperf3-server
  namespace: default
spec:
  port:  5000
```

{{% /expand %}}

### Importing a service

Exposing remote services to a peer is accomplished by creating an Import CR
 to a namespace. The CR represents the imported service and its
 available backends across all peers. In response to an Import CR, ClusterLink
 control plane will create a local Kubernetes Service selecting the ClusterLink
 data plane Pods. The use of native Kubernetes constructs, allows ClusterLink
 to work with any compliant cluster and CNI, transparently.

The Import instance creates the service endpoint in the same namespace as it is
 defined in. The created service will have the Import's `metadata.Name`. This
 allows maintaining independent names for services between peers. Alternately,
 you may use the same name for the import and related source exports.
 You can define multiple Import CRs for the same set of Exports in different
 namespaces. These are independent of each other.

{{% expand summary="Import Custom Resource" %}}

```go
type Import struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec ImportSpec     `json:"spec"`
    Status ImportStatus `json:"status,omitempty"`
}

type ImportSpec struct {
    Port uint16 `json:"port"`
    TargetPort uint16 `json:"targetPort,omitempty"`
    Sources []ImportSource `json:"sources"`
    LBScheme string `json:"lbScheme"`
}

type ImportSource struct {
    Peer string `json:"peer"`
    ExportName string `json:"exportName"`
    ExportNamespace string `json:"exportNamespace"`
}

type ImportStatus struct {
    Conditions []metav1.Condition `json:"conditions,omitempty"`
}
```

{{% /expand %}}

The ImportSpec defines the following fields:

- **Port** (integer, required): the imported, user facing, port number defined
 on the created service object.
- **TargetPort** (integer, optional): this is the internal listening port
 used by the ClusterLink data plane pods to represent the remote services. Typically the
 choice of TargetPort should be left to the ClusterLink control plane, allowing
 it to select a random and non-conflicting port, but there may be cases where
 you wish to assume responsibility for port selection (e.g., a-priori define
 local cluster Kubernetes NetworkPolicy object instances). This may result in
 [port conflicts][] as is done for NodePort services.
- **Sources** (source array, required): references to remote exports providing backends
 for the Import. Each reference names a different export through the combination of:
  - *Peer* (string, required): name of ClusterLink peer where the export is defined.
  - *ExportNamespace* (string, required): name of the namespace on the remote peer where
   the export is defined.
  - *ExportName* (string, required): name of the remote export.
- **LBScheme** (string, optional): load balancing method to select between different
 Sources defined. The default policy is `random`, but you could override it to use
 `round-robin` or `static` (i.e., fixed) assignment.

<!-- Importing multiport? It is not possible... Could use merge in future?
 perhaps, but might requires explicit service name so can merge correctly
 or use port set instead of individual port per export/import -->

As with exports, importing a service does not automatically make it accessible by
 workloads, but only enables *potential* access. To complete service sharing,
 you must define at least one [access control policy][] that
 allows access in the importing cluster. To grant access, a connection must be
 evaluated to "allow" by both egress (importing cluster) and ingress (exporting
 cluster) policies.

{{% expand summary="Example YAML for `kubectl apply -f <import_file>`" %}}

```yaml
apiVersion: clusterlink.net/v1alpha1
kind: Import
metadata:
  name: iperf3-server
  namespace: default
spec:
  port:       5000
  sources:
    - exportName:       iperf3-server
      exportNamespace:  default
      peer:             server
```

{{% /expand %}}


In certain cases, a service can be imported without creating another corresponding service at the imported side, but merging it along with a pre-existing service with the same `name`. This can be specified by adding the label `import.clusterlink.net/merge`, which is set to `true`. This would trigger the creation of an endpointslice which services requests to the imported service (by setting `kubernetes.io/service-name` to the imported service name).

{{% expand summary="Example YAML for `kubectl apply -f <import_file>`" %}}

```yaml
apiVersion: clusterlink.net/v1alpha1
kind: Import
metadata:
  name: iperf3-server
  namespace: default
  labels:
    import.clusterlink.net/merge: true
spec:
  port:       5000
  sources:
    - exportName:       iperf3-server
      exportNamespace:  default
      peer:             server
```
{{% /expand %}}

## Related tasks

Once a service is exported and imported by one or more clusters, you should
 configure [polices][] governing its access.
 For a complete end to end use case, refer to [iperf tutorial][].

[^KEP-1645]: While using similar terminology as the Kubernetes Multicluster Service
 enhancement proposal ([MCS KEP][]), the ClusterLink implementation intentionally
 differs from and is not compliant with the KEP (e.g., there is no `ClusterSet`
 and "name sameness" assumption).

[^multiport]: ClusterLink intentionally does not expose all service ports, as
 typically only a small subset in a multi-port service is meant to be user
 accessible, and other ports are service internal (e.g., ports used for internal
 service coordination and replication).

[fabric]: {{< relref "fabric" >}}
[peers]: {{< relref "peers" >}}
[configured as a peer]: {{< relref "peers#add-or-remove-peers" >}}
[policies]: {{< relref "policies" >}}
[service imports]: #importing-a-service
[port conflicts]: https://kubernetes.io/docs/concepts/services-networking/service/#avoid-nodeport-collisions
[access control policy]: {{< relref "policies" >}}
[iperf tutorial]: {{< relref "../tutorials/iperf" >}}
[deployed and configured]: {{< relref "../getting-started/users#setup" >}}
[MCS KEP]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-multicluster/1645-multi-cluster-services-api
