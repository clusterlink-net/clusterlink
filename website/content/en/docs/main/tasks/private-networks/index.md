---
title: Private Networks
description: Running ClusterLink in a private network when the K8s cluster is behind a NAT or firewall.
---


This task involves connecting ClusterLink behind a NAT or firewall.
To connect the ClusterLink gateway, each peer should have a public IP that will be reachable from other peers to enable cross-cluster communications. In many cases, this is not possible because clusters are behind corporate NAT or firewalls that allow outgoing connections only. For this scenario, we will use the [Fast Reverse Proxy][] (FRP) open-source project to create reverse tunnels and connect all clusters behind a private network. With FRP, only one IP needs to be public to connect all the clusters in the fabric.

To create connectivity between the ClusterLink gateways, we need to set up one FRP server with a public IP and create an FRP client for each ClusterLink gateway, as illustrated below.

This task includes instructions on how to connect the peers using FRP. Instructions for creating full connectivity between applications to remote services can be found in the [Nginx tutorial][] and [iPerf3 tutorial][].

In this task, we will extend the peer connectivity instructions to use FRP.

## Create FRP Server

In this step, we will create the FRP server on the same cluster we use for ClusterLink, but it can be on any peer or Kubernetes cluster.

1. Create a configmap that contains the server configuration:

    ```sh
    echo "
    apiVersion: v1
    kind: ConfigMap
    metadata:
        name: frps-config
        namespace: clusterlink-system
    data:
        frps.toml: |
            bindPort = 4443
    " | kubectl apply -f -
    ```

    In this setup, we expose the FRP server pod on port `4443`.
2. Create FRP server deployment:

    ```sh
    echo "
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: frps
      namespace: clusterlink-system
    spec:
      replicas: 1
      selector:
        matchLabels:
          app: frps
      template:
        metadata:
          labels:
            app: frps
        spec:
          hostNetwork: true
          containers:
            - name: frps
              image: snowdreamtech/frps
              volumeMounts:
                - name: frps-config-volume
                  mountPath: /etc/frp/frps.toml
                  subPath: frps.toml
          volumes:
            - name: frps-config-volume
              configMap:
                name: frps-config
          restartPolicy: Always
    " | kubectl apply -f -
    ```

3. Create sn ingress service to expose the FRP server:

    ```sh
    echo "
    apiVersion: v1
    kind: Service
    metadata:
        name: clusterlink-frps
        namespace: clusterlink-system
    spec:
        type: NodePort
        selector:
            app: frps
        ports:
          - port: 4443
            targetPort: 4443
            nodePort: 30444
    " | kubectl apply -f -
    ```

    In this case, we use a `NodePort` service, but it can be other types like `LoadBalancer`.

## Create FRPs Clients

1. Set the `FRP_SERVER_IP` variable for each cluster:

    *Client cluster*:

    ```sh
    export FRP_SERVER_IP=`docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' client-control-plane`
    ```

    *Client cluster*:

    ```sh
    export FRP_SERVER_IP=`docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' client-control-plane`
    ```

2. Deploy FRP client configuration on each cluster:

    *Client cluster*:

    ```sh
    echo "
    apiVersion: v1
    kind: ConfigMap
    metadata:
        name: frpc-config
        namespace: clusterlink-system
    data:
        frpc.toml: |
            # Set server address
            serverAddr = \""${FRP_SERVER_IP}"\"
            serverPort = 30444

            [[proxies]]
            name = \"clusterlink-client\"
            type = \"stcp\"
            localIP = \"clusterlink.clusterlink-system.svc.cluster.local\"
            localPort = 443
            secretKey = \"abcdefg\"

            [[visitors]]
            name = \"clusterlink-client-to-server-visitor\"
            type = \"stcp\"
            serverName = \"clusterlink-server\"
            secretKey = \"abcdefg\"
            bindAddr = \"::\"
            bindPort = 6002
    " | kubectl apply -f -
    ```

    *Server cluster*:

    ```sh
    echo "
    apiVersion: v1
    kind: ConfigMap
    metadata:
        name: frpc-config
        namespace: clusterlink-system
    data:
        frpc.toml: |
            # Set server address
            serverAddr = \""${FRP_SERVER_IP}"\"
            serverPort = 30444

            [[proxies]]
            name = \"clusterlink-server\"
            type = \"stcp\"
            localIP = \"clusterlink.clusterlink-system.svc.cluster.local\"
            localPort = 443
            secretKey = \"abcdefg\"

            [[visitors]]
            name = \"clusterlink-server-to-client-visitor\"
            type = \"stcp\"
            serverName = \"clusterlink-client\"
            secretKey = \"abcdefg\"
            bindAddr = \"::\"
            bindPort = 6001
    " | kubectl apply -f -
    ```

    For each configuration, we first set the FRP server IP and port number.
    We create a `proxy` that connects to the ClusterLink gateway and establishes a reverse tunnel to allow other clients to connect.
    We also create an FRP `visitor` that specifies which other peers this client wants to connect to (you need to create a visitor for every peer you want to connect).

4. Create a K8s service that connects to the FRP client `visitor`, allowing ClusterLink to connect to it:

    *Client cluster*:

    ```sh
    echo '
    apiVersion: v1
    kind: Service
    metadata:
        name: server-peer-clusterlink
        namespace: clusterlink-system
    spec:
        type: ClusterIP
        selector:
            app: frpc
        ports:
            - port: 6002
              targetPort: 6002
    ' | kubectl apply -f -
     ```

    *Server cluster*:

    ```sh
    echo '
    apiVersion: v1
    kind: Service
    metadata:
        name: client-peer-clusterlink
        namespace: clusterlink-system
    spec:
        type: ClusterIP
        selector:
            app: frpc
        ports:
            - port: 6001
              targetPort: 6001
    ' | kubectl apply -f -
     ```

4. Create FRP client deployment for each cluster:

    *Client cluster*:

    ```sh
    echo "
    apiVersion: apps/v1
    kind: Deployment
    metadata:
        name: frpc
        namespace: clusterlink-system
    spec:
        replicas: 1
        selector:
            matchLabels:
                app: frpc
        template:
            metadata:
                labels:
                    app: frpc
            spec:
                containers:
                    - name: frpc
                      image: snowdreamtech/frpc
                      volumeMounts:
                        - name: frpc-config-volume
                          mountPath: /etc/frp
                volumes:
                  - name: frpc-config-volume
                    configMap:
                        name: frpc-config
                restartPolicy: Always
        " | kubectl apply -f -
    ```

    *Server cluster*:

    ```sh
    echo "
    apiVersion: apps/v1
    kind: Deployment
    metadata:
        name: frpc
        namespace: clusterlink-system
    spec:
        replicas: 1
        selector:
            matchLabels:
                app: frpc
        template:
            metadata:
                labels:
                    app: frpc
            spec:
                containers:
                    - name: frpc
                      image: snowdreamtech/frpc
                      volumeMounts:
                        - name: frpc-config-volume
                          mountPath: /etc/frp
                volumes:
                  - name: frpc-config-volume
                    configMap:
                        name: frpc-config
                restartPolicy: Always
        " | kubectl apply -f -
    ```

## Create Peer CRDs

1. Create Peer CRDs for each peer:

    *Client cluster*:

    ```sh
    echo "
    apiVersion: clusterlink.net/v1alpha1
    kind: Peer
    metadata:
        name: server
        namespace: clusterlink-system
    spec:
        gateways:
            - host: server-peer-clusterlink.clusterlink-system.svc.cluster.local
              port: 6002
    " | kubectl apply -f -
    ```

    *Server cluster*:

    ```sh
    echo "
    apiVersion: clusterlink.net/v1alpha1
    kind: Peer
    metadata:
        name: client
        namespace: clusterlink-system
    spec:
        gateways:
            - host: client-peer-clusterlink.clusterlink-system.svc.cluster.local
              port: 6001
    " | kubectl apply -f -
    ```

    To verify that the connectivity between the peers is established correctly,
    please check if the condition `PeerReachable` has been added to the peer CR status in each cluster.

    ```sh
    kubectl describe peers.clusterlink.net -A
    ```

    {{% expand summary="Sample output" %}}

    ```
    Name:         client
    Namespace:    clusterlink-system
    Labels:       <none>
    Annotations:  <none>
    API Version:  clusterlink.net/v1alpha1
    Kind:         Peer
    Metadata:
        Creation Timestamp:  2024-05-28T12:47:33Z
        Generation:          1
        Resource Version:    807
        UID:                 1fdeafff-707a-43e2-bb3a-826f003a42ed
    Spec:
        Gateways:
        Host:  172.18.0.4
        Port:  30443
    Status:
        Conditions:
        Last Transition Time:  2024-05-28T12:47:33Z
        Message:
        Reason:                Heartbeat
        Status:                True
        Type:                  PeerReachable
    ```

    {{% /expand %}}

## Connect Application Services

After creating the peer connectivity using FRP, continue to the next step of exporting services, importing services, and creating policies as described in the tutorials [Nginx tutorial][] and [iPerf3 tutorial][].

## Cleanup

To remove all FRP components:

1. Delete FRP server deployment, config-map and ingress service :

    ```sh
    kubectl delete deployments -n clusterlink-system frps
    kubectl delete services -n clusterlink-system clusterlink-frps
    kubectl delete configmaps -n clusterlink-system frps-config

1. Delete FRP client deployment, config-map and ingress service  on each cluster:

    *Client cluster*:

    ```sh
    kubectl delete deployments -n clusterlink-system frpc
    kubectl delete services -n clusterlink-system server-peer-clusterlink
    kubectl delete configmaps -n clusterlink-system frpc-config
    ```

    *Server cluster*:

    ```sh
    kubectl delete deployments -n clusterlink-system frpc
    kubectl delete services -n clusterlink-system client-peer-clusterlink
    kubectl delete configmaps -n clusterlink-system frpc-config
    ```

[Nginx tutorial]: {{< relref "../../tutorials/nginx/_index.md" >}}
[iPerf3 tutorial]: {{< relref "../../tutorials/iperf/_index.md" >}}
[Fast Reverse Proxy]: https://github.com/fatedier/frp