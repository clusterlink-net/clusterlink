---
title: iPerf3
description: Running basic connectivity between iPerf3 applications across two sites using ClusterLink.
---

In this tutorial we establish iPerf3 connectivity between two kind cluster using ClusterLink.
The tutorial uses two kind clusters:

1) Client cluster - runs ClusterLink along with an iPerf3 client.
2) Server cluster - runs ClusterLink along with an iPerf3 server.

## Install ClusterLink CLI

1. Install ClusterLink on Linux or Mac using the installation script:

    ```sh
    curl -L https://github.com/clusterlink-net/clusterlink/releases/latest/download/clusterlink.sh | sh -
    ```

1. Verify the installation:

    ```sh
    clusterlink --version
    ```

## Initialize clusters

Before you start, you must have access to two K8s clusters.
For example, in this tutorial we set up a local environment using the [kind](https://kind.sigs.k8s.io/) project.
To setup two kind clusters:

1. Install kind using [kind installation guide](https://kind.sigs.k8s.io/docs/user/quick-start).
2. Create a directory for all the tutorial files:

    ```sh
    mkdir iperf3-tutorial
    ```

3. Open two terminals in the tutorial directory and create a kind cluster in each terminal:

    Client cluster:

    ```sh
    cd iperf3-tutorial
    kind create cluster --name=client
    ```

    Server cluster:

    ```sh
    cd iperf3-tutorial
    kind create cluster --name=server
    ```

{{< notice note >}}
kind uses the prefix "kind", so the name of created clusters will be **kind-client** and **kind-server**.
{{< /notice >}}

4. Setup `KUBECONFIG` on each terminal to access the cluster:

    Client cluster:

    ```sh
    kubectl config use-context kind-client
    cp ~/.kube/config $PWD/config-client
    export KUBECONFIG=$PWD/config-client
    ```

    Server cluster:

    ```sh
    kubectl config use-context kind-server
    cp ~/.kube/config $PWD/config-server
    export KUBECONFIG=$PWD/config-server
    ```

{{< notice note >}}
You can run the tutorial in a single terminal and switch access between the clusters
using `kubectl config use-context kind-client` and `kubectl config use-context kind-server`.
{{< /notice >}}

## Deploy iPerf3 client and server

1. Install iPerf3 (client and server) on the clusters:

    Client cluster:

    ```sh
    export IPERF3_FILES=https://raw.githubusercontent.com/clusterlink-net/clusterlink/main/demos/iperf3/testdata/manifests
    kubectl apply -f $IPERF3_FILES/iperf3-client/iperf3-client.yaml
    ```

    Server cluster:

    ```sh
    export IPERF3_FILES=https://raw.githubusercontent.com/clusterlink-net/clusterlink/main/demos/iperf3/testdata/manifests
    kubectl apply -f $IPERF3_FILES/iperf3-server/iperf3.yaml
    ```

## Deploy ClusterLink

1. Create the fabric and peer certificates for the clusters:

    Client cluster:

    ```sh
    clusterlink create fabric
    clusterlink create peer-cert --name client
    ```

    Server cluster:

    ```sh
    clusterlink create peer-cert --name server
    ```

    For more details about fabric and peer concepts see [core concepts](https://clusterlink.net/docs/concepts).

2. Deploy ClusterLink on each cluster:

    Client cluster:

    ```sh
    clusterlink deploy peer --name client --autostart --ingress=NodePort --ingress-port=30443
    ```

    Server cluster:

    ```sh
    clusterlink deploy peer --name server --autostart --ingress=NodePort --ingress-port=30443
    ```

{{< notice note >}}
In this example, we use NodePort to create an external access point for the kind clusters.
By default `deploy peer` creates an ingress of type LoadBalancer,
which is more suitable for Kubernetes clusters running in the cloud.
{{< /notice >}}

3. Verify that ClusterLink controlplane and dataplane are running:

    Client cluster:

    ```sh
    kubectl rollout status deployment cl-controlplane -n clusterlink-system
    kubectl rollout status deployment cl-dataplane -n clusterlink-system
    ```

    Server cluster:

    ```sh
    kubectl rollout status deployment cl-controlplane -n clusterlink-system
    kubectl rollout status deployment cl-dataplane -n clusterlink-system
    ```

{{% expand summary="Output" %}}
It may take a few seconds for the deployments to be successfully created.

```sh
deployment "cl-controlplane" successfully rolled out
deployment "cl-dataplane" successfully rolled out
```

{{% /expand %}}

## Enable cross-cluster access

In this step, we enable connectivity access between the iPerf3 client and server.

1. First, add the peers to each cluster:

    Client cluster:

    ```sh
    export SERVER_IP=`docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' server-control-plane`
    curl -s $IPERF3_FILES/clusterlink/peer-server.yaml | envsubst | kubectl apply -f -
    ```

    Server cluster:

    ```sh
    export CLIENT_IP=`docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' client-control-plane`
    curl -s $IPERF3_FILES/clusterlink/peer-client.yaml | envsubst | kubectl apply -f -
    ```

{{% expand summary="Install **envsubst** for **macOS users**" %}}
In case `envsubst` does not exist, you can install it with:

```sh
brew install gettext
brew link --force gettext
```

{{% /expand %}}

<br>

{{< notice note >}}

The `CLIENT_IP` and `SERVER_IP` refers to the node IP of the peer kind cluster, which assigns the peer YAML file.
{{< /notice >}}

2. In the server cluster, export the iperf3-server service:

    Server cluster:

    ```sh
    kubectl apply -f $IPERF3_FILES/clusterlink/export-iperf3.yaml
    ```

3. In the client cluster, import the iperf3-server service from the server cluster:

    Client cluster:

    ```sh
    kubectl apply -f $IPERF3_FILES/clusterlink/import-iperf3.yaml
    ```

4. Create access policies on both clusters to allow connectivity:

    Client cluster:

    ```sh
    kubectl apply -f $IPERF3_FILES/clusterlink/allow-policy.yaml
    ```

    Server cluster:

    ```sh
    kubectl apply -f $IPERF3_FILES/clusterlink/allow-policy.yaml
    ```

    For more details about policies see [ClusterLink policies](https://clusterlink.net/docs/concepts/policies).

## Test service connectivity

Test the iperf3 connectivity between the clusters:

Client cluster:

```sh
export IPERF3CLIENT=`kubectl get pods -l app=iperf3-client -o custom-columns=:metadata.name --no-headers`
kubectl exec -i $IPERF3CLIENT -- iperf3 -c iperf3-server --port 5000
```

{{% expand summary="Output" %}}

```
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

1. Delete all kind clusters:
    Client cluster:

    ```sh
    kind delete cluster --name=client
    ```

    Server cluster:

    ```sh
    kind delete cluster --name=server
    ```

1. Remove tutorial directory:

    ```sh
    cd .. && rm -rf iperf3-tutorial
    ```

1. Unset environment variables:
    Client cluster:

    ```sh
    unset KUBECONFIG IPERF3_FILES IPERF3CLIENT
    ```

    Server cluster:

    ```sh
    unset KUBECONFIG IPERF3_FILES
    ```
