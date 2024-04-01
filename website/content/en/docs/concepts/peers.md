---
title: Peers
description: Defining ClusterLink Peers as part of Fabric
weight: 20
---

A `Peer` represents a location, such as a Kubernetes cluster, participating in a
 [Fabric]({{< ref "fabric" >}}). Each peer may host one or more [Services]({{< ref "services" >}})
 it wishes to share with other peers. A peer is managed by a peer administrator,
 which is responsible for running the ClusterLink control and data planes. The
 administrator will typically deploy the ClusterLink components by configuring
 the [deployment CRD]({{< ref "users#deploy-crd-instance" >}}). They may also wish to provide
 (often) coarse-grained access policies in accordance with high level corporate
 policies (e.g., "production peers should only communicate with other production peers").

Once a Peer has been added to a Fabric, it can communicate with any other Peer
 belonging to it. All configuration relating to service sharing (e.g., the exporting
 and importing of Services, and the setting of fine grained application policies) can be
 done with lowered privileges (e.g., by users, such as application owners). Remote peers are
 represented by the `Peer` Custom Resource Definition (CRD). Each Peer CRD instance
 defines a remote cluster and the network endpoints of its ClusterLink gateways.

## Prerequisites

The following assume that you have access to the `clusterlink` CLI and one or more
 peers (i.e., clusters) where you'll deploy ClusterLink. The CLI can be downloaded
 from the ClusterLink [releases page on GitHub](https://github.com/clusterlink-net/clusterlink/releases/latest).
 It also assumes that you have access to the [previously created]({{< ref "fabric#create-a-new-fabric-ca" >}})
 Fabric CA files.

## Initializing a new Peer

{{< notice warning >}}
Creating a new Peer is a **Fabric administrator** level operation and should be appropriately
 protected.
{{< /notice >}}

### Create a new Peer certificate

To create a new Peer certificate belonging to a fabric, confirm that the Fabric CA files
 are available in the current working directory, and then execute the following CLI command:

```sh
clusterlink create peer-cert --name <peer_name> --fabric <fabric_name>
```

{{< notice tip >}}
The Fabric CA files (certificate and private key) are expected to be in a subdirectory (i.e., `./<fabric_name>/cert.name` and `./<fabric_name>/key.pem`).
{{< /notice >}}

This will create the certificate and private key files (`cert.pem` and
 `key.pem`, respectively) for the new peer. By default, the files are
 created in a subdirectory named `<peer_name>` under the subdirectory of the fabric `<fabric_name>`.
 You can override the default by setting the `--output <path>` option.

{{< notice info >}}
You will need the CA certificate (but **not** the CA private key) and the peer certificate
 and private in the next step. They can be provided out of band (e.g., over email) to the
 peer administrator.
{{< /notice >}}

## Deploy ClusterLink to a new Peer

{{< notice info >}}
This operation is typically done by a local **Peer administrator**, usually different
 than the **Fabric administrator**.
{{< /notice >}}

Before proceeding, ensure that the CA certificate (the CA private key is not needed),
 and the peer certificate and key files which were created in the previous step are
 in the current working directory.

### Install the ClusterLink deployment operator

Install the ClusterLink operator by running the following command

```sh
clusterlink peer init
```
<!-- TODO: is this the right command -->

The command assumes that kubectl is set to the correct context and credentials
and that the certificates were created in the local folder. If they were not,
add the `-f <path>` CLI option to set the correct path to the certificate files.

This command will deploy the ClusterLink deployment CRDs using the current
kubectl context. The operation requires cluster administrator privileges
in order to install CRDs into the cluster.
The ClusterLink operator is installed to the `clusterlink-operator` namespace
and the CA and peer certificate and key are set as Kubernetes secrets
in the namespace. You can confirm the successful completion of the step using
the following commands:

```sh
kubectl get crds
kubectl get secret --namespace clusterlink-operator
```

{{% expand summary="Example output" %}}

```sh
$ kubectl get crds
output of `kubectl get crds`
over multiple lines
$ kubectl get secret --namespace clusterlink-operator
multiline output of `kubectl get secret --namespace clusterlink-operator` command
...
```

{{% /expand %}}

### Deploy ClusterLink via the Operator and ClusterLink CRD

After the operator is installed, you can deploy ClusterLink by applying
 the ClusterLink instance CRD. This will cause the ClusterLink operator to
 attempt reconciliation of the actual and intended ClusterLink deployment.
 By default, the operator will install the ClusterLink control and data plane
 components into a dedicated and privileged namespace (defaults to `clusterlink-system`).
 Configurations affecting the entire peer, such as the list of known Peers, are also maintained
 in the same namespace.

Refer to the [getting started guide]({{< ref "users#setup" >}}) for a description
 of the ClusterLink instance CRD's fields.

<!-- TODO expand the sample CRD file? -->

## Add or remove Peers

{{< notice info >}}
This operation is typically done by a local **Peer administrator**, usually different
 than the **Fabric administrator**.
{{< /notice >}}

Managing peers is done by creating, deleting and updating Peer CRD instances
 in the dedicated ClusterLink namespace (typically, `clusterlink-system`). Peers are
 added to the ClusterLink namespace by the peer administrator. Information
 regarding peer gateways and attributes is communicated out of band (e.g., provided
 by the Fabric or remote Peer administrator over email). In the future, these may
 be configured via a management plane.

There are two fundamental attributes in the Peer CRD: the Peer's name and the list of
 ClusterLink gateway endpoints through which the remote peer's Services are available.
 Peer names are unique and must align with the Subject name present in their certificate
 during connection establishment. The name is used by importers in referencing an export
 (see [here]({{< ref "services" >}}) for details).

Gateway endpoint would typically be a implemented via a `NodePort` or `LoadBalancer`
 Kubernetes Service. A `NodePort` Service would typically be used in local deployments
 (e.g., when running in KIND clusters during development) and a `LoadBalancer` Service
 would be used in Cloud based deployments. These can be automatically configured and
 created via the [operator CRD]{{< ref "#deploy-clusterlink-via-the-operator-and-clusterlink-crd" >}}.
 Not having any gateways is an error and will be reported in the Peer's Status.
 In addition, the Status section includes other useful attributes, such a `Reachable`
 (or `Seen`) indicating whether the Peer is currently reachable, the last time it
 successfully responded to heartbeats, etc.

{{% expand summary="Example YAML for `kubectl apply -f <peer_file>`" %}}
{{< readfile file="/static/files/peer_crd_sample.yaml" code="true" lang="yaml" >}}
{{% /expand %}}

## Related tasks

Once a peer has been created and initialized with the ClusterLink control and data
 planes as well as one or more remote Peers, you can proceed with configuring
 [services]({{< ref "services" >}}) and [policies]({{< ref "policies" >}}).
 For a complete end to end use case, refer to [iperf toturial]({{< ref "iperf" >}}).
