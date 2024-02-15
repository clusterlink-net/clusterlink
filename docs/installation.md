# Installation guide for ClusterLink

The ClusterLink gateway contains three main components: the control-plane, data-plane, and gwctl (more details can be found [here](../README.md#what-is-clusterlink)).

ClusterLink can be deployed in any K8s based cluster (e.g., google GKE, amazon EKS, IBM IKS, KIND etc.).

To deploy the ClusterLink gateway in a K8s cluster, follow these steps:

1. Create and deploy certificates and the ClusterLink deployment YAML file.
2. Expose the ClusterLink deployment to a public IP.
3. Optionally, create a central gwctl component to control one or more gateways.

Before you begin, please build the project according to the [instructions](../README.md#building-clustelink).

## 1. ClusterLink deployment and certificates

In this step, we generate the ClusterLink certificates and deployment files.

1) Create a folder (eg. DEPLOY_DIR) for all deployments and yaml files:

        export PROJECT_DIR=`git rev-parse --show-toplevel`
        export DEPLOY_DIR=~/clusterlink-deployment
        mkdir -p $DEPLOY_DIR
        cd $DEPLOY_DIR

2) Create fabric certificate, which is the root CA certificate for all the clusters:

        $PROJECT_DIR/bin/cl-adm create fabric

    Note: This step needs to be done only once per fabric.
3) Create certificates and a deployment file for the cluster (e.g., peer1):

        $PROJECT_DIR/bin/cl-adm create peer --name peer1 --namespace default

    Note: By default, the images are pulled from ghcr.io/clusterlink-net. To change the container registry, you can use the ```--container-registry``` flag.  
    This step needs to be performed for every K8s cluster.

4) Apply ClusterLink deployment:
   
        kubectl apply -f $DEPLOY_DIR/peer1/k8s.yaml

5) Verify that all components (cl-controlplane, cl-dataplane, gwctl) are set up and ready.
   
        kubectl rollout status deployment cl-controlplane
        kubectl rollout status deployment cl-dataplane
        kubectl wait --for=condition=ready pod -l app=gwctl 
        
    Expected output:
    
        deployment "cl-controlplane" successfully rolled out
        deployment "cl-dataplane" successfully rolled out
        pod/gwctl condition met

## 2. Expose the ClusterLink deployment to a public IP

Create a public IP for accessing the ClusterLink gateway.  
* For a testing environment (e.g., KIND), create a K8s nodeport service:
```
echo "
apiVersion: v1
kind: Service
metadata:
  name: cl-svc
spec:
  type: NodePort
  selector:
    app: cl-dataplane
  ports:
  - port: 443
    targetPort: 443
    nodePort: 30443
    protocol: TCP
    name: http
" | kubectl apply -f -

export PEER1_IP=`kubectl get nodes -o "jsonpath={.items[0].status.addresses[0].address}"`
export PEER1_PORT=30443
```
* For a operational K8s cluster environment (e.g., google GKE, amazon EKS, IBM IKS, etc.):

1. First, create a K8s LoadBalancer service:

        kubectl expose deployment cl-dataplane --name=cl-dataplane-load-balancer --port=443 --target-port=443 --type=LoadBalancer

2. Retrieve the LoadBalancer IP when it is allocated:

        export PEER1_IP=$(kubectl get svc -l app=cl-dataplane -o jsonpath="{.items[0].status.loadBalancer.ingress[0].ip}")
        export PEER1_PORT=443

Now, the ClusterLink gateway can be accessed through `$PEER1_IP` at port `$PEER1_PORT`.

## 3. Create a central gwctl (optional)
By default for each K8s cluster a gwctl pod is created that use REST APIs to send control messages to the
ClusterLink gateway, using the command:

    kubectl exec -i <gwctl_pod> -- gwctl <command>

To create a single gwctl that controls one or more ClusterLink gateways, follow these steps:

1. Install the local control (gwctl):

        sudo make install

2. Initialize the gwctl CLI for the cluster (e.g., peer1):

        gwctl init --id peer1 --gwIP $PEER1_IP --gwPort $PEER1_PORT --certca $DEPLOY_DIR/cert.pem --cert $DEPLOY_DIR/peer1/gwctl/cert.pem --key $DEPLOY_DIR/peer1/gwctl/key.pem

3. To run gwctl command:

        gwctl --myid peer1 <command>

## Additional setup modes

### Debug mode

To run ClusterLink components in debug mode, use ```--log-level``` flag when creating the ClusterLink deployment

    $PROJECT_DIR/bin/cl-adm create peer --name peer1 --log-level debug

### Running self-built images

To create and run a local image, first build the project's local images:

    make docker-build

Change the container-registry in the peer deployment to use the local image:

    $PROJECT_DIR/bin/cl-adm create peer --name peer1 --container-registry=""

### Running on a different namespace

To run ClusterLink components in a different namespace, use the `--namespace flag` when creating the ClusterLink deployment.

    $PROJECT_DIR/bin/cl-adm create peer --name peer1 --namespace <namespace>
