# Scenario with iPerf3 Connectivity between 2 clusters
In this test we check iPerf3 connectivity between different Kind cluster using ClusterLink.  
This setup use two Kind clusters- 
1) Cluster 1 - contains the gateway (Control-plane/Data-plane) and iPerf3 client.
2) Cluster 2 - contains the gateway (Control-plane/Data-plane) and iPerf3 server.

## Pre-requires installations
To run a Kind test, check all pre-requires are installed (Go, docker, Kubectl, Kind):

    export PROJECT_DIR=`git rev-parse --show-toplevel`
    cd $PROJECT_DIR
    make prereqs

Following are the steps to run the iperf3 connectivity scenario (or you could also run the scripts towards the end):
### Step 1: Build docker image
Build project docker image:
    
    make docker-build

Install local control (gwctl) for the gateway:

    sudo make install

### Step 2: Create Kind clusters with ClusterLink gateway
In this step, we build the Kind cluster with the gateway image.
Create certificates and deployments files:
1) Create folder for all deployments and yaml files:

        export DEPLOY_DIR=~/clusterlink-deployment
        mkdir -p $DEPLOY_DIR
        cd $DEPLOY_DIR

2) Create fabric certificates (includes the Root CA certificate fo all the gateways):

        $PROJECT_DIR/bin/cl-adm create fabric

3) Create peer1 certificates and deployment file:

        $PROJECT_DIR/bin/cl-adm create peer --name peer1 --namespace default

4) Create peer2 certificates and deployment file:

        $PROJECT_DIR/bin/cl-adm create peer --name peer2 --namespace default

Build the first Kind cluster with gateway, and iperf3-client:
1) Create a Kind cluster with the gateway image:

        cd $PROJECT_DIR
        kind create cluster --name=peer1
        kind load docker-image cl-controlplane cl-dataplane cl-go-dataplane gwctl --name=peer1

2) Deploy ClusterLink gateway deployment:

        kubectl apply -f $DEPLOY_DIR/peer1/k8s.yaml

3) Expose ClusterLink gateway port using K8s nodeport service:

        kubectl apply -f $PROJECT_DIR/demos/utils/manifests/kind/cl-svc.yaml

4) Create an iPerf3-client deployment: 
 
        kind load docker-image mlabbe/iperf3 --name=peer1
        kubectl create -f $PROJECT_DIR/demos/iperf3/testdata/manifests/iperf3-client/iperf3-client.yaml

Build the second Kind cluster with gateway, and iperf3-client:
1) Create a Kind cluster with the gateway image:

        kind create cluster --name=peer2
        kind load docker-image cl-controlplane cl-dataplane cl-go-dataplane gwctl --name=peer2

2) Deploy Clustrelink gateway deployment:

        kubectl apply -f $DEPLOY_DIR/peer2/k8s.yaml

3) Expose ClusterLink gateway port using K8s nodeport service:
    
        kubectl apply -f $PROJECT_DIR/demos/utils/manifests/kind/cl-svc.yaml

3) Create an iPerf3-server deployment:

        kind load docker-image mlabbe/iperf3 --name=peer2
        kubectl create -f $PROJECT_DIR/demos/iperf3/testdata/manifests/iperf3-server/iperf3.yaml

Check that container statuses are Running.

        kubectl rollout status deployment cl-controlplane
        kubectl rollout status deployment cl-dataplane
        kubectl wait --for=condition=ready pod -l app=gwctl 

### Step 3: Start gwctl
In this step, we set up the gwctl.
First, Initialize the Gateways IPs:
    
    kubectl config use-context kind-peer1
    export PEER1_IP=`kubectl get nodes -o "jsonpath={.items[0].status.addresses[0].address}"`
    kubectl config use-context kind-peer2
    export PEER2_IP=`kubectl get nodes -o "jsonpath={.items[0].status.addresses[0].address}"`

Initialize gwctl CLI for peer1:

    gwctl init --id "peer1" --gwIP $PEER1_IP --gwPort 30443 --certca $DEPLOY_DIR/cert.pem --cert $DEPLOY_DIR/peer1/gwctl/cert.pem --key $DEPLOY_DIR/peer1/gwctl/key.pem

Initialize gwctl CLI for peer2:

    gwctl init --id "peer2" --gwIP $PEER2_IP --gwPort 30443 --certca $DEPLOY_DIR/cert.pem --cert $DEPLOY_DIR/peer2/gwctl/cert.pem --key $DEPLOY_DIR/peer2/gwctl/key.pem

Note : Another approach is to use the gwctl inside the kind cluster. In this case, the gwctl is already set up and deployed.
(If you are using macOS this is the better approach). 
In this case, you should replace the gwctl command with:

    kubectl exec -i <gwctl_pod> -- gwctl <command>

    See example in the next step.

### Step 4: Peers communication
In this step, we add a peer for each gateway using the gwctl:

    gwctl create peer --myid peer1 --name peer2 --host $PEER2_IP --port 30443
    gwctl create peer --myid peer2 --name peer1 --host $PEER1_IP --port 30443

When running Kind cluster on macOS run instead the following: 
    
    kubectl config use-context kind-peer1
    export GWCTL1=`kubectl get pods -l app=gwctl -o custom-columns=:metadata.name --no-headers`
    kubectl exec -i $GWCTL1 -- gwctl create peer --myid peer1 --name peer2 --host $PEER2_IP --port 30443

    kubectl config use-context kind-peer2
    export GWCTL2=`kubectl get pods -l app=gwctl -o custom-columns=:metadata.name --no-headers`
    kubectl exec -i $GWCTL2 -- gwctl create peer --myid peer2 --name peer1 --host $PEER1_IP --port 30443


### Step 5: Export a service
In this step, we add the iperf3 server to the cluster as an exported service that can be accessed from remote peers.  
Export the iperf3-server service to the Cluster 2 gateway:

    gwctl create export --myid peer2 --name iperf3-server --host iperf3-server --port 5000

When running Kind cluster on macOS run instead the following: 

    kubectl exec -i $GWCTL2 -- gwctl create export --myid peer2 --name iperf3-server --host iperf3-server --port 5000

Note: iperf3-client doesn't need to be added since it is not exported.

### Step 6: import iperf3 server service from Cluster 2
In this step, we import the iperf3-server service from Cluster 2 gateway to Cluster 1 gateway
First, we specify which service we want to import and specify the local k8s endpoint (host:port) that will create for this service:

    gwctl create import --myid peer1 --name iperf3-server --host iperf3-server --port 5000

When running Kind cluster on macOS run instead the following:

    kubectl config use-context kind-peer1
    kubectl exec -i $GWCTL1-- gwctl create import --myid peer1 --name iperf3-server --port 5000

Second, we specify the peer we want to import the service:

    gwctl create binding --myid peer1 --import iperf3-server --peer peer2

When running Kind cluster on macOS run instead the following:
 
    kubectl config use-context kind-peer1
    kubectl exec -i $GWCTL1 -- gwctl create binding --myid peer1 --import iperf3-server --peer peer2

### Step 7: Create access policy
In this step, we create a policy that allow to all traffic from peer1 and peer2:

    gwctl --myid peer1 create policy --type access --policyFile $PROJECT_DIR/pkg/policyengine/examples/allowAll.json 
    gwctl --myid peer2 create policy --type access --policyFile $PROJECT_DIR/pkg/policyengine/examples/allowAll.json 

When running Kind cluster on macOS run instead the following:

    kubectl config use-context kind-peer1
    kubectl cp $PROJECT_DIR/pkg/policyengine/examples/allowAll.json gwctl:/tmp/allowAll.json 
    kubectl exec -i $GWCTL1 -- gwctl create policy --type access --policyFile /tmp/allowAll.json 
    kubectl config use-context kind-peer2
    kubectl cp $PROJECT_DIR/pkg/policyengine/examples/allowAll.json gwctl:/tmp/allowAll.json 
    kubectl exec -i $GWCTL2 -- gwctl create policy --type access --policyFile /tmp/allowAll.json 

### Final Step : Test Service connectivity
Start the iperf3 test from cluster 1:

    kubectl config use-context kind-peer1
    export IPERF3CLIENT=`kubectl get pods -l app=iperf3-client -o custom-columns=:metadata.name --no-headers`
    kubectl exec -i $IPERF3CLIENT -- iperf3 -c iperf3-server --port 5000

### Cleanup
Delete all Kind clusters:

    kind delete cluster --name=peer1
    kind delete cluster --name=peer2
