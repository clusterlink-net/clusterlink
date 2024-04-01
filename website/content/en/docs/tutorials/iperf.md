---
title: iPerf3
description: Running basic connectivity between iPerf3 applications across two sites using ClusterLink.
---

In this tutorial we establish iPerf3 connectivity between two Kind cluster using ClusterLink.
The tutorial uses two Kind clusters-

1) Client cluster - contains iPerf3 client and ClusterLink.
2) Server cluster - contains iPerf3 server and ClusterLink.

## Install ClusterLink CLI

1. Install ClusterLink on Linux or Mac, use the installation script:

    ```sh
    curl -L https://github.com/clusterlink-net/clusterlink/releases/latest/download/clusterlink.sh | sh -
    ```

1. Check the installation by running the command:

    ```sh
    clusterlink --version
    ```

## Initialize Clusters

Before you start, you must have access to two Kubernetes clusters.
For example, In this tutorial we set up a local environment using the [Kind](https://kind.sigs.k8s.io/) project.
To setup two kind clusters:

1. Install Kind using [Kind installation guide](https://kind.sigs.k8s.io/docs/user/quick-start).
2. Create a folder for all the tutorial files:

    ```sh
    mkdir iperf3-tutorial
    ```

3. Open two terminals in the tutorial folder and create a Kind cluster in each terminal:

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
Kind add the prefix "kind" in the start sot the name of cluster that will be create will be **kind-client** and **kind-server**.
{{< /notice >}}

1. Setup `KUBECONFIG` on each terminal to access the cluster:

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
You can run the tutorial in a single terminal and switch access between the clusters using `kubectl config use-context kind-client` and `kubectl config use-context kind-server`.
{{< /notice >}}

## Deploy iPerf3 Client and Server

1. Install iPerf3 application (Client and server) on the clusters.

    Client cluster:

    ```sh
    export IPERF3_FILES=https://github.com/clusterlink-net/clusterlink/raw/main/demos/iperf3/testdata/manifests
    kubectl apply -f $IPERF3_FILES/iperf3-client/iperf3-client.yaml
    ```

    Server cluster:

    ```sh
    export IPERF3_FILES=https://github.com/clusterlink-net/clusterlink/raw/main/demos/iperf3/testdata/manifests
    kubectl apply -f $IPERF3_FILES/iperf3-server/iperf3.yaml
    ```

## Deploy ClusterLink

1. Create the fabric and peer certificates for the clusters.

    Client cluster:

    ```sh
    clusterlink create fabric
    clusterlink create peer-cert --name client
    ```

    Server cluster:

    ```sh
    clusterlink create peer-cert --name server
    ```

    For more details about fabric and peer concepts see [ClusterLink Core Concepts](https://clusterlink.net/docs/concepts/).

1. Deploy ClusterLink on each cluster:

    Client cluster:

    ```sh
    clusterlink deploy peer --name client --autostart --ingress=NodePort --ingress-port=30443
    ```

    Server cluster:

    ```sh
    clusterlink deploy peer --name server --autostart --ingress=NodePort --ingress-port=30443
    ```

{{< notice note >}}
In this example, we use NodePort to create an external access point for the Kind clusters. By default `deploy peer` creates an ingress of type LoadBalancer, which is more suitable for Kubernetes clusters running in the cloud.
{{< /notice >}}

1. Check that ClusterLink controlplane and dataplane are Running.

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

## Enable Access between the clusters

In this step, we enable connectivity access between the iPerf3 client and server.

1. First, we add the peers to each cluster:

    Client cluster:

    ```sh
    export SERVER_IP=`docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' server-control-plane`
    envsubst < $IPERF3_FILES/clusterlink/peer-server.yaml | kubectl apply -f -
    ```

    Server cluster:

    ```sh
    export CLIENT_IP=`docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' client-control-plane`
    envsubst < $IPERF3_FILES/clusterlink/peer-client.yaml | kubectl apply -f -
    ```

{{< notice note >}}
The `PEER_X_IP` refers to the node IP of the peer Kind Cluster, which assigns the peer YAML file
{{< /notice >}}

1. In the server cluster, we add the iperf3-server to clusterlink as an exported service that can be accessed from remote peers.

    Server cluster:

    ```sh
    kubectl apply -f $IPERF3_FILES/clusterlink/export-iperf3.yaml
    ```

1. In the client cluster, we import iperf3-server from the server cluster.

    Client cluster:

    ```sh
    kubectl apply -f $IPERF3_FILES/clusterlink/import-iperf3.yaml
    ```

1. We create access policies on both clusters to allow connectivity.

    Client cluster:

    ```sh
    kubectl apply -f $IPERF3_FILES/clusterlink/allow-policy.yaml
    ```

    Server cluster:

    ```sh
    kubectl apply -f $IPERF3_FILES/clusterlink/allow-policy.yaml
    ```

    For more details about policy see [ClusterLink policies](https://clusterlink.net/docs/concepts/policies)

## Test Service connectivity

Test the iperf3 connectivity between the clusters.

Client cluster:

```sh
export IPERF3CLIENT=`kubectl get pods -l app=iperf3-client -o custom-columns=:metadata.name --no-headers`
kubectl exec -i $IPERF3CLIENT -- iperf3 -c iperf3-server --port 5000
```

## Cleanup

1. Delete all Kind clusters.
    Client cluster:

    ```sh
    kind delete cluster --name=client
    ```

    Server cluster:

    ```sh
    kind delete cluster --name=server
    ```

1. Remove tutorial folder:

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
