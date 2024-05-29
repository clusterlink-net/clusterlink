
1. Create the fabric and peer certificates for the clusters:

    *Client cluster*:

    ```sh
    clusterlink create fabric
    clusterlink create peer-cert --name client
    ```

    *Server cluster*:

    ```sh
    clusterlink create peer-cert --name server
    ```

    All peer certificates (i.e., for the `client` and `server` clusters, in this tutorial) should be created from the same fabric CA files.
    In this tutorial, we assume the server has access to the Fabric certificate created in the `default_fabric` folder.
    In this tutorial, we assume the `server` cluster creation has access to the fabric certificate stored in the
    `default_fabric` folder. If it doesn't, the fabric certificate should be copied from the `client` to the `server`.

    For more details regarding fabric and peer see [core concepts][].

2. Deploy ClusterLink on each cluster:

    *Client cluster*:

    ```sh
    clusterlink deploy peer --name client --ingress=NodePort --ingress-port=30443
    ```

    *Server cluster*:

    ```sh
    clusterlink deploy peer --name server --ingress=NodePort --ingress-port=30443
    ```

   This tutorial uses **NodePort** to create an external access point for the kind clusters.
    By default `deploy peer` creates an ingress of type **LoadBalancer**,
    which is more suitable for Kubernetes clusters running in the cloud.

3. Verify that ClusterLink control and data plane components are running:

    It may take a few seconds for the deployments to be successfully created.

    *Client cluster*:

    ```sh
    kubectl rollout status deployment cl-controlplane -n clusterlink-system
    kubectl rollout status deployment cl-dataplane -n clusterlink-system
    ```

    *Server cluster*:

    ```sh
    kubectl rollout status deployment cl-controlplane -n clusterlink-system
    kubectl rollout status deployment cl-dataplane -n clusterlink-system
    ```

    {{% expand summary="Sample output" %}}

    deployment "cl-controlplane" successfully rolled out
    deployment "cl-dataplane" successfully rolled out

    {{% /expand %}}

[core concepts]: {{< relref "../../concepts/_index.md" >}}
