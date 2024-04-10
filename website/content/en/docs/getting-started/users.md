---
title: Users
description: Installing and configuring a basic ClusterLink deployment
weight: 22
---

This guide will give you a quick start on installing and setting up the ClusterLink on a Kubernetes cluster.

## Prerequisites

Before you start, you must have access to a Kubernetes cluster.
For example, you can set up a local environment using the [kind](https://kind.sigs.k8s.io/) project.

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

1. {{< anchor create-fabric-ca >}}Create the fabric's CA certificate and private key:

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
    The `--name` option is optional, and by default, "default_fabric" will be used.

1. {{< anchor install-cl-operator >}}Install ClusterLink deployment operator:

   ```sh
   clusterlink deploy peer --autostart --name <peer_name> --fabric <fabric_name>
   ```

   This command will deploy the ClusterLink operator on the `clusterlink-operator` namespace
    and convert the peer certificates to secrets in this namespace where ClusterLink components will be installed.
    By default, the `clusterlink-system` namespace is used.
    The command assumes that `kubectl` is set to the correct peer (K8s cluster)
    and that the certificates were created by running the previous command on the same working directory.
    If they were not, use the flag `--path <path>` for pointing to the working directory
    that was used in the previous command.
    The `--fabric` option is optional, and by default, "default_fabric" will be used.
    The `--autostart` option will deploy the ClusterLink components in the `clusterlink-system` namespace,
    and enable ClusterLink in the cluster.
    For more details and deployment configuration see [ClusterLink deployment operator]({{< ref "operator" >}}).

{{< notice note >}}
To deploy ClusterLink on another cluster, repeat steps 2-3 in a console with access to the cluster.
{{< /notice >}}

## Try it out

Check out the [ClusterLink Tutorials]({{< ref "tutorials" >}}) for setting up
multi-cluster connectivity for applications using two or more clusters.

## Uninstall ClusterLink

To uninstall the ClusterLink CLI, use the following command:

```sh
rm `which clusterlink`
```
