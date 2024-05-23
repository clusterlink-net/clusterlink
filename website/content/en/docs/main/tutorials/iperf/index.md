---
title: iPerf3
description: Running basic connectivity between iPerf3 applications across two sites using ClusterLink
---

In this tutorial we'll establish iPerf3 connectivity between two kind clusters using ClusterLink.
The tutorial uses two kind clusters:

1) Client cluster - runs ClusterLink along with an iPerf3 client.
2) Server cluster - runs ClusterLink along with an iPerf3 server.

## Install ClusterLink CLI

{{% include "../shared/cli-installation.md" %}}

## Initialize clusters

In this tutorial we set up a local environment using [kind][].
 You can skip this step if you already have access to existing clusters, just be sure to
 set KUBECONFIG accordingly.

To setup two kind clusters:

1. Install kind using [kind installation guide][].
1. Create a directory for all the tutorial files:

    ```sh
    mkdir iperf3-tutorial
    ```

1. Open two terminals in the tutorial directory and create a kind cluster in each terminal:

    *Client cluster*:

    ```sh
    cd iperf3-tutorial
    kind create cluster --name=client
    ```

    *Server cluster*:

    ```sh
    cd iperf3-tutorial
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

## Deploy iPerf3 client and server

Install iPerf3 (client and server) on the clusters:

*Client cluster*:

```sh
export TEST_FILES=https://raw.githubusercontent.com/clusterlink-net/clusterlink/main/demos/iperf3/testdata/manifests
kubectl apply -f $TEST_FILES/iperf3-client/iperf3-client.yaml
```

*Server cluster*:

```sh
export TEST_FILES=https://raw.githubusercontent.com/clusterlink-net/clusterlink/main/demos/iperf3/testdata/manifests
kubectl apply -f $TEST_FILES/iperf3-server/iperf3.yaml
```

## Deploy ClusterLink

{{% include "../shared/deploy-clusterlink.md" %}}

## Enable cross-cluster access

In this step, we enable connectivity access between the iPerf3 client and server.
 For each step, you have an example demonstrating how to apply the command from a
 file or providing the complete custom resource (CR) associated with the command.

Note that the provided YAML configuration files refer to environment variables
 (defined below) that should be set when running the tutorial. The values are
 replaced in the YAMLs using `envsubst` utility.

{{% expand summary="Installing `envsubst` on macOS" %}}
In case `envsubst` does not exist, you can install it with:

```sh
brew install gettext
brew link --force gettext
```

{{% /expand %}}

### Set-up peers

{{% include "../shared/peer.md" %}}

{{< notice note >}}
The `CLIENT_IP` and `SERVER_IP` refers to the node IP of the peer kind cluster, which assigns the peer YAML file.
{{< /notice >}}

### Export the iPerf server endpoint

In the server cluster, export the iperf3-server service:

*Server cluster*:

{{< tabpane text=true >}}
{{% tab header="File" %}}

```sh
kubectl apply -f $TEST_FILES/clusterlink/export-iperf3.yaml
```

{{% /tab %}}
{{% tab header="Full CR" %}}

```sh
echo "
apiVersion: clusterlink.net/v1alpha1
kind: Export
metadata:
  name: iperf3-server
  namespace: default
spec:
  port:  5000
" | kubectl apply -f -
```

{{% /tab %}}
{{< /tabpane >}}

### Set-up import

In the client cluster, import the iperf3-server service from the server cluster:

*Client cluster*:

{{< tabpane text=true >}}
{{% tab header="File" %}}

```sh
kubectl apply -f $TEST_FILES/clusterlink/import-iperf3.yaml
```

{{% /tab %}}
{{% tab header="Full CR" %}}

```sh
echo "
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
" | kubectl apply -f -
```

{{% /tab %}}
{{< /tabpane >}}

### Set-up access policies

{{% include "../shared/policy.md" %}}

## Test service connectivity

Test the iperf3 connectivity between the clusters:

*Client cluster*:

```sh
export IPERF3CLIENT=`kubectl get pods -l app=iperf3-client -o custom-columns=:metadata.name --no-headers`
kubectl exec -i $IPERF3CLIENT -- iperf3 -c iperf3-server --port 5000
```

{{% expand summary="Sample output" %}}

```sh
Connecting to host iperf3-server, port 5000
[  5] local 10.244.0.5 port 51666 connected to 10.96.46.198 port 5000
[ ID] Interval           Transfer     Bitrate         Retr  Cwnd
[  5]   0.00-1.00   sec   639 MBytes  5.36 Gbits/sec    0    938 KBytes
[  5]   1.00-2.00   sec   627 MBytes  5.26 Gbits/sec    0    938 KBytes
[  5]   2.00-3.00   sec   628 MBytes  5.26 Gbits/sec    0    938 KBytes
[  5]   3.00-4.00   sec   635 MBytes  5.33 Gbits/sec    0    938 KBytes
[  5]   4.00-5.00   sec   630 MBytes  5.29 Gbits/sec    0    938 KBytes
[  5]   5.00-6.00   sec   636 MBytes  5.33 Gbits/sec    0    938 KBytes
[  5]   6.00-7.00   sec   639 MBytes  5.36 Gbits/sec    0    938 KBytes
[  5]   7.00-8.00   sec   634 MBytes  5.32 Gbits/sec    0    938 KBytes
[  5]   8.00-9.00   sec   641 MBytes  5.39 Gbits/sec    0    938 KBytes
[  5]   9.00-10.00  sec   633 MBytes  5.30 Gbits/sec    0    938 KBytes
- - - - - - - - - - - - - - - - - - - - - - - - -
[ ID] Interval           Transfer     Bitrate         Retr
[  5]   0.00-10.00  sec  6.19 GBytes  5.32 Gbits/sec    0             sender
[  5]   0.00-10.00  sec  6.18 GBytes  5.31 Gbits/sec                  receiver

iperf Done.
```

{{% /expand %}}

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
    cd .. && rm -rf iperf3-tutorial
    ```

1. Unset the environment variables:

    *Client cluster*:

    ```sh
    unset KUBECONFIG TEST_FILES IPERF3CLIENT
    ```

    *Server cluster*:

    ```sh
    unset KUBECONFIG TEST_FILES
    ```

[kind]: https://kind.sigs.k8s.io/
[kind installation guide]: https://kind.sigs.k8s.io/docs/user/quick-start
[core concepts]: {{< relref "../../concepts/_index.md" >}}
