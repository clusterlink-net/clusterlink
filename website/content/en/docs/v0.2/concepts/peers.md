---
title: Peers
description: Defining ClusterLink peers as part of a fabric
weight: 20
---

A *Peer* represents a location, such as a Kubernetes cluster, participating in a
 [fabric][concept-fabric]. Each peer may host one or more [services][concept-service]
 that it wishes to share with other peers. A peer is managed by a peer administrator,
 which is responsible for running the ClusterLink control and data planes. The
 administrator will typically deploy the ClusterLink components by configuring
 the [deployment Custom Resource (CR)][operator-cr]. The administrator may also wish
 to provide coarse-grained access policies (and often do) in accordance with high level corporate
 policies (e.g., "production peers should only communicate with other production peers").

Once a peer has been added to a fabric, it can communicate with any other peer
 belonging to it. All configuration relating to service sharing (e.g., the exporting
 and importing of services, and the setting of fine grained application policies) can be
 done with lowered privileges (e.g., by users, such as application owners). Remote peers are
 represented by peer Custom Resource Definition (CRDs). Each Peer CR instance
 defines a remote cluster and the network endpoints of its ClusterLink gateways.

## Prerequisites

The following sections assume that you have access to the ClusterLink CLI and one or more
 peers (i.e., clusters) where you'll deploy ClusterLink. The CLI can be downloaded
 from the ClusterLink [releases page on GitHub](https://github.com/clusterlink-net/clusterlink/releases/latest).
 It also assumes that you have access to the [previously created][concept-fabric-new]
 fabric CA files.

## Initializing a new peer

{{< notice warning >}}
Creating a new peer is a **fabric administrator** level operation and should be appropriately
 protected.
{{< /notice >}}

### Create a new peer certificate

To create a new peer certificate belonging to a fabric, confirm that the fabric Certificate Authority (CA) files
 are available in the current working directory, and then execute the following CLI command:

```sh
clusterlink create peer-cert --name <peer_name> --fabric <fabric_name>
```

{{< notice tip >}}
The fabric CA files (certificate and private key) are expected to be in a subdirectory
 (i.e., `./<fabric_name>/cert.name` and `./<fabric_name>/key.pem`).
{{< /notice >}}

This will create the certificate and private key files (`cert.pem` and
 `key.pem`, respectively) of the new peer. By default, the files are
 created in a subdirectory named `<peer_name>` under the subdirectory of the fabric `<fabric_name>`.
 You can override the default by setting the `--output <path>` option.

{{< notice info >}}
You will need the CA certificate (but **not** the CA private key) and the peer certificate
 and private in the next step. They can be provided out of band (e.g., over email) to the
 peer administrator.
{{< /notice >}}

## Deploy ClusterLink to a new peer

{{< notice info >}}
This operation is typically done by a local **peer administrator**, usually different
 than the **fabric administrator**.
{{< /notice >}}

Before proceeding, ensure that the CA certificate (the CA private key is not needed),
 and the peer certificate and key files which were created in the previous step are
 in the current working directory.

### Install the ClusterLink deployment operator

Install the ClusterLink operator by running the following command:

```sh
clusterlink deploy peer --name <peer_name> --fabric <fabric_name>
```

The command assumes that kubectl is set to the correct context and credentials
 and that the certificates were created in respective sub-directories
 under the current working directory.
 If they were not, add the `--path <path>` CLI option to set the correct path.

This command will deploy the ClusterLink deployment CRDs using the current
 `kubectl` context. The operation requires cluster administrator privileges
 in order to install CRDs into the cluster.
 The ClusterLink operator is installed to the `clusterlink-operator` namespace.
 The CA, peer certificate, and private key are set as K8s secrets
 in the namespace where ClusterLink components are installed, which by default is
 `clusterlink-system`. You can confirm the successful completion of this step
 using the following commands:

```sh
kubectl get crds
kubectl get secret --namespace clusterlink-system
```

{{% expand summary="Example output" %}}

```sh
$ kubectl get crds
NAME                                       CREATED AT
accesspolicies.clusterlink.net             2024-04-07T12:08:24Z
exports.clusterlink.net                    2024-04-07T12:08:24Z
imports.clusterlink.net                    2024-04-07T12:08:24Z
instances.clusterlink.net                  2024-04-07T12:08:24Z
peers.clusterlink.net                      2024-04-07T12:08:24Z
privilegedaccesspolicies.clusterlink.net   2024-04-07T12:08:24Z

$ kubectl get secret --namespace clusterlink-system
NAME              TYPE     DATA   AGE
cl-controlplane   Opaque   2      19h
cl-dataplane      Opaque   2      19h
cl-fabric         Opaque   1      19h
cl-peer           Opaque   1      19h
```

{{% /expand %}}

### Deploy ClusterLink via the operator and ClusterLink CR

After the operator is installed, you can deploy ClusterLink by applying
 the ClusterLink CR. This will cause the ClusterLink operator to
 attempt reconciliation of the actual and intended ClusterLink deployment.
 By default, the operator will install the ClusterLink control and data plane
 components into a dedicated and privileged namespace (defaults to `clusterlink-system`).
 Configurations affecting the entire peer, such as the list of known peers, are also maintained
 in the same namespace.

Refer to the [operator documentation][operator-cli-flags] for a description
 of the ClusterLink CR fields.

## Add or remove peers

{{< notice info >}}
This operation is typically done by a local **peer administrator**, usually different
 than the **fabric administrator**.
{{< /notice >}}

Managing peers is done by creating, deleting and updating peer CRs
 in the dedicated ClusterLink namespace (typically, `clusterlink-system`). Peers are
 added to the ClusterLink namespace by the peer administrator. Information
 regarding peer gateways and attributes is communicated out of band (e.g., provided
 by the fabric or remote peer administrator over email). In the future, these may
 be configured via a management plane.

{{% expand summary="Peer Custom Resource" %}}

```go
type Peer struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec PeerSpec `json:"spec"`
    Status PeerStatus `json:"status,omitempty"`
}


type PeerSpec struct {
    Gateways []Endpoint `json:"gateways"`
}

type PeerStatus struct {
    Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type Endpoint struct {
    Host string `json:"host"`
    Port uint16 `json:"port"`
}
```

{{% /expand %}}

There are two fundamental attributes in the peer CRD: the peer name and the list of
 ClusterLink gateway endpoints through which the remote peer's services are available.
 Peer names are unique and must align with the Subject name present in their certificate
 during connection establishment. The name is used by importers in referencing an export
 (see [Services][concept-service] for details).

Gateway endpoint would typically be implemented via a `NodePort` or `LoadBalancer`
 K8s service. A `NodePort` service would typically be used in local deployments
 (e.g., when running in kind clusters during development) and a `LoadBalancer` service
 would be used in cloud based deployments. These can be automatically configured and
 created via the [ClusterLink CR][concept-peer-deploy-via-cr].
 The peer's status section includes a `Reachable` condition indicating whether the peer is currently reachable,
 and in case it is not reachable, the last time it was.

{{% expand summary="Example YAML for `kubectl apply -f <peer_file>`" %}}
{{< readfile file="/static/files/peer_crd_sample.yaml" code="true" lang="yaml" >}}
{{% /expand %}}

## Related tasks

Once a peer has been created and initialized with the ClusterLink control and data
 planes as well as one or more remote peers, you can proceed with configuring
 [services][concept-service] and [policies][concept-policy].
 For a complete end-to-end use case, refer to the [iperf tutorial][tutorial-iperf].

[concept-fabric]: {{< relref "fabric" >}}
[concept-fabric-new]: {{< relref "fabric#create-a-new-fabric-ca" >}}
[concept-service]: {{< relref "services" >}}
[concept-policy]: {{< relref "policies" >}}
[operator-cr]: {{< relref "../tasks/operator#deploy-cr-instance" >}}
[operator-cli-flags]: {{< relref "../tasks/operator#commandline-flags" >}}
[concept-peer-deploy-via-cr]: {{< relref "peers#deploy-clusterlink-via-the-operator-and-clusterlink-cr" >}}
[tutorial-iperf]: {{< relref "../tutorials/iperf" >}}
