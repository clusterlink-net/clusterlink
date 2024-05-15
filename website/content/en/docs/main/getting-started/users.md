---
title: Users
description: Installing and configuring a basic ClusterLink deployment
weight: 22
---

This guide will give you a quick start on installing and setting up ClusterLink on a Kubernetes cluster.

## Prerequisites

Before you start, you must have access to a Kubernetes cluster.
For example, you can set up a local environment using [kind](https://kind.sigs.k8s.io/docs/user/quick-start/).

## Installation

1. {{< anchor install-cli>}}To install ClusterLink on Linux or Mac, use the installation script:

   ```sh
   curl -L https://github.com/clusterlink-net/clusterlink/releases/latest/download/clusterlink.sh | sh -
   ```

1. Check the installation by running the command:

   ```sh
   clusterlink --version
   ```

## Setup

To set up ClusterLink on a Kubernetes cluster, follow these steps:

1. {{< anchor create-fabric-ca >}}Create the fabric's certificate authority (CA) certificate and private key:

   ```sh
   clusterlink create fabric --name <fabric_name>
   ```

   The ClusterLink fabric is defined as all K8s clusters (peers) that install ClusterLink gateways
    and can share services between the clusters, enabling communication among those services.
    This command will create the CA files `cert.pem` and `key.pem` in a directory named <fabric_name>.
    The `--name` option is optional, and by default, "default_fabric" will be used.

1. {{< anchor create-peer-certs >}}Create a peer (cluster) certificate:

   ```sh
   clusterlink create peer-cert --name <peer_name> --fabric <fabric_name>
   ```

   This command will create the certificate files `cert.pem` and `key.pem`
    in a directory named `<fabric_name>`/`<peer_name>`.
    The `--path <path>` flag can be used to change the directory location.
    Here too, the `--name` option is optional, and by default, "default_fabric" will be used.

**All the peer certificates in the fabric should be created from the same fabric CA files in step 1.**

1. {{< anchor install-cl-operator >}}Install ClusterLink deployment:

   ```sh
   clusterlink deploy peer --name <peer_name> --fabric <fabric_name>
   ```

   This command will deploy the ClusterLink operator on the `clusterlink-operator` namespace
    and convert the peer certificates to secrets in the namespace where ClusterLink components will be installed.
    By default, the `clusterlink-system` namespace is used.
    in addition it will create a ClusterLink instance custom resource object and deploy it to the operator.
    The operator will then create the ClusterLink components in the `clusterlink-system` namespace and enable ClusterLink in the cluster.
    The command assumes that `kubectl` is set to the correct peer (K8s cluster)
    and that the certificates were created by running the previous command on the same working directory.
    If they were not, use the flag `--path <path>` for pointing to the working directory
    that was used in the previous command.
    The `--fabric` option is optional, and by default, "default_fabric" will be used.
    For more details and deployment configuration see [ClusterLink deployment operator][].
{{< notice note >}}
To set up ClusterLink on another cluster, create another set of peer certificates (step 2).
Deploy ClusterLink in a console with access to the cluster (step 3).
{{< /notice >}}

## Try it out

Check out the [ClusterLink Tutorials](tutorials) for setting up multi-cluster connectivity
 for applications using two or more clusters.

## Uninstall ClusterLink

1. To remove a ClusterLink instance from the cluster, please delete the ClusterLink instance custom resource.
   The ClusterLink operator will subsequently remove all instance components (control-plane, data-plane, and ingress service).

   ```sh
   kubectl delete instances.clusterlink.net  -A --all
   ```

2. To completely remove ClusterLink from the cluster, including the operator, CRDs, namespaces, and instances,
   use the following command:

   ```sh
   clusterlink delete peer --name peer1
   ```

{{< notice note >}}
This command  using the current `kubectl` context.
{{< /notice >}}

3. To uninstall the ClusterLink CLI, use the following command:

   ```sh
   rm `which clusterlink`
   ```

[kind]: https://kind.sigs.k8s.io/)
[ClusterLink deployment operator]: {{< relref "../tasks/operator" >}}
[ClusterLink tutorials]: {{< relref "../tutorials/" >}}
