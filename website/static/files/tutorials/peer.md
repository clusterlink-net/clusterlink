
Add the remote peer to each cluster:

  *Client cluster*:

  {{< tabpane text=true >}}
  {{% tab header="File" %}}

  ```sh
  export SERVER_IP=`docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' server-control-plane`
  curl -s $TEST_FILES/clusterlink/peer-server.yaml | envsubst | kubectl apply -f -
  ```

  {{% /tab %}}
  {{% tab header="Full CR" %}}

  ```sh
  export SERVER_IP=`docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' server-control-plane`
  echo "
  apiVersion: clusterlink.net/v1alpha1
  kind: Peer
  metadata:
    name: server
    namespace: clusterlink-system
  spec:
    gateways:
      - host: "${SERVER_IP}"
        port: 30443
  " | kubectl apply -f -
  ```

  {{% /tab %}}
  {{< /tabpane >}}

  *Server cluster*:

  {{< tabpane text=true >}}
  {{% tab header="File" %}}

  ```sh
  export CLIENT_IP=`docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' client-control-plane`
  curl -s $TEST_FILES/clusterlink/peer-client.yaml | envsubst | kubectl apply -f -
  ```

  {{% /tab %}}
  {{% tab header="Full CR" %}}

  ```sh
  export CLIENT_IP=`docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' client-control-plane`
  echo "
  apiVersion: clusterlink.net/v1alpha1
  kind: Peer
  metadata:
    name: client
    namespace: clusterlink-system
  spec:
    gateways:
      - host: "${CLIENT_IP}"
        port: 30443
  " | kubectl apply -f -
  ```

  {{% /tab %}}
  {{< /tabpane >}}

  The `CLIENT_IP` and `SERVER_IP` refers to the node IP of the peer kind cluster, which assigns the peer YAML file.

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
