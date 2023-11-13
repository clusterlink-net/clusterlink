# Installation Guide for ClusterLink
The ClusterLink gateway contains three main components: the control-plane, data-plane, and gwctl (more details can be found [here](../README.md#what-is-clusterlink)).

ClusterLink can be deployed in a local environment (Kind) or any remote K8s cluster environment (GKE, EKS, IBM-K8S, etc.).

To deploy the ClusterLink gateway in a K8s cluster, follow these steps:

1. Create and deploy certificates and the ClusterLink Deployment file.
2. Expose the ClusterLink deployment to a public IP.
3. Optionally, create a central gwctl component to control all gateways.

Before you begin, please clone and build the project according to the [instructions](../README.md#building-clustelink).
### 1. ClusterLink Deployment and certificates.
In this step, we build the ClusterLink certificates and deployment files.

1) Create a folder (eg. TEST_DIR) for all deployments and yaml files:

        export PROJECT_DIR=`git rev-parse --show-toplevel`
        export TEST_DIR=$PROJECT_DIR/bin/test/
        mkdir -p $TEST_DIR
        cd $TEST_DIR

2) Create Fabric certificates, including the Root CA certificate for all the clusters (This step needs to be done only once per fabric).:

        $PROJECT_DIR/bin/cl-adm create fabric

3) Create certificates and a deployment file for the cluster (e.g., peer1). This step needs to be performed for every K8s cluster.:

        $PROJECT_DIR/bin/cl-adm create peer --name peer1

    Note: By default, the images are pulled from ghcr.io/clusterlink-net. To change the container registry, you can use the ```--container-registry``` flag.

4) Apply ClusterLink deployment:
   
        kubectl apply -f $TEST_DIR/peer1/k8s.yaml


5) Verify that all components (cl-controlplane, cl-dataplane, gwctl) are set up and ready.
   
        kubectl get pods

### 2. Expose the ClusterLink deployment to a public IP
Create public IP for accessing the ClusterLink gateway.
For Local environment create a K8s nodeport service:

        kubectl apply -f $PROJECT_DIR/demos/utils/manifests/kind/cl-svc.yaml
        export PEER1_IP=`kubectl get nodes -o "jsonpath={.items[0].status.addresses[0].address}"`

For a remote environment:

1. First, create a K8s LoadBalancer service:
    ```
    kubectl expose deployment cl-dataplane --name=cl-dataplane-load-balancer --port=443 --target-port=443 --type=LoadBalancer
    ```

2. Retrieve the LoadBalancer IP when it is allocated:
    ```
    export PEER1_IP=$(kubectl get svc -l app=cl-dataplane -o jsonpath="{.items[0].status.loadBalancer.ingress[0].ip}")
    ```

Now, the ClusterLink gateway can be accessed through `$PEER1_IP` at port 443.

### 3. Create central gwctl (optional)
By default for each K8s cluster a gwctl pod is created that use REST APIs to send control messages to the
ClusterLink Gateway, using the command:

    kubectl exec -i <gwctl_pod> -- gwctl <the command>

To create a single gwctl that controls all ClusterLink gateways, follow these steps:

1. Install the local control (gwctl):

    ```
    sudo make install
    ```

2. Initialize the gwctl CLI for the cluster (e.g., peer1):
    ```
    gwctl init --id "peer1" --gwIP $PEER1_IP --gwPort 30443 --certca $TEST_DIR/cert.pem --cert $TEST_DIR/peer1/gwctl/cert.pem --key
    ```

## Additional Setup Modes
### Debug mode
To run ClusterLink component in debug mode, use ```--log-level``` flag when creating the ClusterLink deployment

    $PROJECT_DIR/bin/cl-adm create peer --name peer1 --log-level debug
### Local docker image
To create and run a local image, first build the project's local images:

    make docker-build
Change the container-registry in the peer deployment to use the local image:

    $PROJECT_DIR/bin/cl-adm create peer --name peer1 --container-registry=""
