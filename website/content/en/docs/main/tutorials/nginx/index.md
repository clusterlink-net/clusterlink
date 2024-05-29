---
title: nginx
description: Running basic connectivity between nginx server and client across two clusters using ClusterLink.
---

In this tutorial, we'll establish connectivity across clusters using ClusterLink to access a remote nginx server.
The tutorial uses two kind clusters:

1) Client cluster - runs ClusterLink along with a client.
2) Server cluster - runs ClusterLink along with a nginx server.

## Install ClusterLink CLI

{{% readfile file="/static/files/tutorials/cli-installation.md" %}}

## Initialize clusters

This tutorial uses [kind][] as a local Kubernetes environment.
 You can skip this step if you already have access to existing clusters, just be sure to
 set KUBECONFIG accordingly.

To setup two kind clusters:

1. Install kind using [kind installation guide][].
1. Create a directory for all the tutorial files:

    ```sh
    mkdir nginx-tutorial
    ```

1. Open two terminals in the tutorial directory and create a kind cluster in each terminal:

    *Client cluster*:

    ```sh
    cd nginx-tutorial
    kind create cluster --name=client
    ```

    *Server cluster*:

    ```sh
    cd nginx-tutorial
    kind create cluster --name=server
    ```

   {{< notice note >}}
   kind uses the prefix `kind`, so the name of created clusters will be **kind-client** and **kind-server**.
   {{< /notice >}}

1. Setup `KUBECONFIG` on each terminal to access the cluster:

    *Client cluster*:

    ```sh
    kubectl config use-context kind-client
    cp ~/.kube/config $PWD/config-client
    export KUBECONFIG=$PWD/config-client
    ```

    *Server cluster*:

    ```sh
    kubectl config use-context kind-server
    cp ~/.kube/config $PWD/config-server
    export KUBECONFIG=$PWD/config-server
    ```

{{< notice tip >}}
You can run the tutorial in a single terminal and switch access between the clusters
using `kubectl config use-context kind-client` and `kubectl config use-context kind-server`.
{{< /notice >}}

## Deploy nginx client and server

Setup the ```TEST_FILES``` variable, and install nginx on the server cluster.

*Client cluster*:

```sh
export TEST_FILES=https://raw.githubusercontent.com/clusterlink-net/clusterlink/main/demos/nginx/testdata
```

*Server cluster*:

```sh
export TEST_FILES=https://raw.githubusercontent.com/clusterlink-net/clusterlink/main/demos/nginx/testdata
kubectl apply -f $TEST_FILES/nginx-server.yaml
```

## Deploy ClusterLink

{{% readfile file="/static/files/tutorials/deploy-clusterlink.md" %}}

## Enable cross-cluster access

In this step, we enable access between the client and server.
 For each step, you have an example demonstrating how to apply the command from a
 file or providing the complete custom resource (CR) associated with the command.

{{% readfile file="/static/files/tutorials/envsubst.md" %}}

### Set-up peers

{{% readfile file="/static/files/tutorials/peer.md" %}}

{{< notice note >}}
The `CLIENT_IP` and `SERVER_IP` refers to the node IP of the peer kind cluster, which assigns the peer YAML file.
{{< /notice >}}

### Export the nginx server endpoint

In the server cluster, export the nginx server service:

*Server cluster*:

{{< tabpane text=true >}}
{{% tab header="File" %}}

```sh
kubectl apply -f $TEST_FILES/clusterlink/export-nginx.yaml
```

{{% /tab %}}
{{% tab header="Full CR" %}}

```sh
echo "
apiVersion: clusterlink.net/v1alpha1
kind: Export
metadata:
  name: nginx
  namespace: default
spec:
  port:  80
" | kubectl apply -f -
```

{{% /tab %}}
{{< /tabpane >}}

### Set-up import

In the client cluster, import the nginx service from the server cluster:

*Client cluster*:

{{< tabpane text=true >}}
{{% tab header="File" %}}

```sh
kubectl apply -f $TEST_FILES/clusterlink/import-nginx.yaml
```

{{% /tab %}}
{{% tab header="Full CR" %}}

```sh
echo "
apiVersion: clusterlink.net/v1alpha1
kind: Import
metadata:
  name: nginx
  namespace: default
spec:
  port:       80
  sources:
    - exportName:       nginx
      exportNamespace:  default
      peer:             server
" | kubectl apply -f -
```

{{% /tab %}}
{{< /tabpane >}}

### Set-up access policies

{{% readfile file="/static/files/tutorials/allow-all-policy.md" %}}

## Test service connectivity

Test the connectivity between the clusters with a batch job of the ```curl``` command:

*Client cluster*:

```sh
kubectl apply -f $TEST_FILES/nginx-job.yaml
```

Verify the job succeeded:

```sh
kubectl logs jobs/curl-nginx-homepage
```

{{% readfile file="/static/files/tutorials/nginx/nginx-output.md" %}}

## Cleanup

1. Delete the kind clusters:
    *Client cluster*:

    ```sh
    kind delete cluster --name=client
    ```

    *Server cluster*:

    ```sh
    kind delete cluster --name=server
    ```

1. Remove the tutorial directory:

    ```sh
    cd .. && rm -rf nginx-tutorial
    ```

1. Unset the environment variables:
    *Client cluster*:

    ```sh
    unset KUBECONFIG TEST_FILES
    ```

    *Server cluster*:

    ```sh
    unset KUBECONFIG TEST_FILES
    ```

[kind]: https://kind.sigs.k8s.io/
[kind installation guide]: https://kind.sigs.k8s.io/docs/user/quick-start
