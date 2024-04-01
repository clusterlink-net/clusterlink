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
    in a directory named <fabric_name>/<peer_name>.
    The `--path <path>` flag can be used to change the directory location.
    The `--name` option is optional, and by default, "default_fabric" will be used.

1. {{< anchor install-cl-operator >}}Install ClusterLink deployment operator:

    ```sh
    clusterlink peer deploy --autostart --name <peer_name> --fabric <fabric_name>
    ```

    This command will deploy the ClusterLink operator on the `clusterlink-operator` namespace
    and convert the peer certificates to secrets in this namespace.
    The command assumes that `kubectl` is set to the correct peer (K8s cluster)
    and that the certificates were created by running the previous command on the same working directory.
    If they were not, use the flag `--path <path>` for pointing to the working directory
    that was used in the previous command.
    The `--fabric` option is optional, and by default, "default_fabric" will be used.
    The `--autostart` option will deploy the ClusterLink components in the `clusterlink-system` namespace,
    and the ClusterLink project will start running in the cluster.

To deploy ClusterLink on another cluster, please repeat steps 2-3 in a console with access to the cluster.

### Additional configurations

* Setting ClusterLink namespace:

    ```sh
    clusterlink peer deploy --autostart --name <peer_name> --fabric <fabric_name> --namespace <namespace>
    ```

    The `--namespace` parameter determines the namespace where the ClusterLink components are deployed.
    Note that you must set `--autostart`, and the namespace should already exist.

* Setting ClusterLink ingress type:

    ```sh
    clusterlink peer deploy --autostart --name <peer_name> --fabric <fabric_name> --ingress <ingress_type>
    ```

    The `--ingress` parameter controls the ClusterLink ingress type, with `LoadBalancer` being the default.
    If you're using a kind cluster, replace `<ingress_type>` with `NodePort`.
    For a cluster running in a cloud environment, use `LoadBalancer`.
    Note that you must set `--autostart` when you use this parameter.

* {{< anchor deploy-cr-instance >}}Full configuration setting using a ClusterLink K8s custom resource object:

    ```yaml
    kubectl apply -f - <<EOF
    apiVersion: clusterlink.net/v1alpha1
    kind: ClusterLink
    metadata:
    namespace: clusterlink-operator
    name: <peer_name>
    spec:
    ingress:
        type: <ingress_type>
    namespace: clusterlink-system
    EOF
    ```

    After the operator is installed, you can deploy ClusterLink by applying a ClusterLink object as shown above.
    The Clusterlink operator will create the ClusterLink gateway components in the `clusterlink-system` namespace.
    For more details and information about the ClusterLink custom resource,
    refer to the [operator documentation](https://github.com/clusterlink-net/clusterlink/blob/main/design-proposals/project-deployment.md#clusterlink-crd).

## Try it out

Check out the [ClusterLink Tutorials]({{< ref "tutorials" >}}) for setting up
 multi-cluster connectivity for applications using two or more clusters.
