---
title: Getting Started
weight: 20
---

This guide will give you a quick start on installing and setting up the ClusterLink on a Kubernetes cluster.

## Prerequisites

Before you start, you must have access to a Kubernetes cluster. For example, you can set up a local environment using the [Kind](https://kind.sigs.k8s.io/) project.

## Installation

1. To install ClusterLink on Linux or Mac, use the installation script:

    ```sh
    curl -L https://github.com/clusterlink-net/clusterlink/releases/latest/clusterlink.sh | sh -
    ```

1. Move to the ClusterLink project folder:

    ```sh
    cd clusterlink
    ```

1. Export the ClusterLink CLI to your path:

    ```sh
    export PATH=$PWD/bin:$PATH
    ```

1. Check the installation by running the command:

    ```sh
    clusterlink version
    ```

## Setup

To set up ClusterLink on a Kubernetes cluster, follow these steps:

1. Create fabric certificates.

    The ClusterLink Fabric is defined as all Kubernetes clusters (sites) that install ClusterLink gateways and can share services between the clusters, enabling communication among those services.
    First, create the fabric Certificate Authority (CA):

    ```sh
    clusterlink create fabric --name <fabric_name>
    ```

    This command will create the CA files `<fabric_name>.cert` and `<fabric_name>.key` in the current folder.

1. Create site certificates.

    Create a site (cluster) certificate:

    ```sh
    clusterlink create site --name <site_name> --fabric <fabric_name>
    ```

    This command will create the certificate files `<site_name>.cert` and `<site_name>.key` in a folder named <site_name>. The `--output <path>` flag can be used to change the folder location.

1. Install ClusterLink deployment operator.

    To install ClusterLink on the site, first, install the ClusterLink deployment operator.

    ```sh
    clusterlink site init
    ```

    This command will deploy the ClusterLink deployment operator on the `clusterlink-operator` namespace and convert the site certificates to secrets in the namespace.
    The command assumes that kubectl is set to the correct site (Kubernetes cluster) and that the certificates were created in the local folder. If they were not, use the flag `-f <path>`.

1. Deploy clusterlink CRD instance.

    After the operator is installed, you can deploy ClusterLink by applying the ClusterLink instance CRD:

    ```yaml
    kubectl apply -f - <<EOF
    apiVersion: clusterlink.net/v1alpha1
    kind: ClusterLink
    metadata:
    namespace: clusterlink-operator
    name: <site_name>
    spec:
    ingress:
        type: <ingress_type>
    namespace: clusterlink-system
    EOF
    ```

    If you're using a Kind cluster, replace <ingress_type> with `NodePort`. For a cluster running in a cloud environment, use `LoadBalancer` instead.

    The instance CRD will create the ClusterLink gateway components in the `clusterlink-system` namespace.
    For more details and information about the ClusterLink instance CRD, refer to the [operator documentation](https://github.com/clusterlink-net/clusterlink/blob/main/design-proposals/project-deploymnet.md#clusterlink-crd).

To deploy ClusterLink on another cluster, please repeat steps 2-4 in the console with access to the cluster.

## Try it out

Check out the [ClusterLink Tutorials]({{< ref "tutorials" >}}) for setting up multi-cluster connectivity for applications using two or more clusters.
