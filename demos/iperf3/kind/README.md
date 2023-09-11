# <ins>Scenario with iPerf3 Connectivity between 2 clusters<ins>
In this test we check iPerf3 connectivity between different Kind cluster using the MBG components.  
This setup use two Kind clusters- 
1) Cluster 1 - contains the gateway (Control-plane/Data-plane) and iPerf3 client.
2) Cluster 2 - contains the gateway (Control-plane/Data-plane) and iPerf3 server.


## <ins> Pre-requires installations <ins>
To run a Kind test, check all pre-requires are installed (Go, docker, Kubectl, Kind):

    export PROJECT_FOLDER=`git rev-parse --show-toplevel`
    cd $PROJECT_FOLDER
    make prereqs

Following are the steps to run the iperf3 connectivity scenario (or you could also run the scripts towards the end):
### <ins> Step 1: Build docker image <ins>
Build MBG docker image:
    
    make build
    make docker-build

Install local control (gwctl) for the gateway:
    
    sudo make install

### <ins> Step 2: Create Kind clusters with the gateway <ins>
In this step, we build the Kind cluster with the gateway image.  
Build the first Kind cluster with gateway, and iperf3-client:
1) Create a Kind cluster with the gateway image:

        kind create cluster --name=cluster1
        kind load docker-image mbg --name=cluster1

2) Create the gateway control plane and data plane deployment: 
    
        kubectl create -f $PROJECT_FOLDER/config/manifests/mbg/mbg.yaml
        kubectl apply -f $PROJECT_FOLDER/config/manifests/mbg/mbg-role.yaml
        kubectl create -f $PROJECT_FOLDER/config/manifests/mbg/dataplane.yaml 
        kubectl create service nodeport dataplane --tcp=443:443 --node-port=30443

3) Create an iPerf3-client deployment: 
   
        kind load docker-image mlabbe/iperf3 --name=cluster1
        kubectl create -f $PROJECT_FOLDER/demos/iperf3/testdata/manifests/iperf3-client/iperf3-client.yaml

Build the second Kind cluster with gateway, and iperf3-client:
1) Create a Kind cluster with the gateway image:

        kind create cluster --name=cluster2
        kind load docker-image mbg --name=cluster2

2) Create the gateway control plane and data plane deployment:
   
        kubectl create -f $PROJECT_FOLDER/config/manifests/mbg/mbg.yaml
        kubectl apply -f $PROJECT_FOLDER/config/manifests/mbg/mbg-role.yaml
        kubectl create -f $PROJECT_FOLDER/config/manifests/mbg/dataplane.yaml 
        kubectl create service nodeport dataplane --tcp=443:443 --node-port=30443

3) Create an iPerf3-server deployment:
   
        kind load docker-image mlabbe/iperf3 --name=cluster2
        kubectl create -f $PROJECT_FOLDER/demos/iperf3/testdata/manifests/iperf3-server/iperf3.yaml
        kubectl create service nodeport iperf3-server --tcp=5000:5000 --node-port=30001

Check that container statuses are Running.

    kubectl get pods

### <ins> Step 3: Start running MBG and gwctl <ins>
In this step, start to run the MBG and gwctl.  
First, Initialize the parameters of the test (pods' names and IPs):
    
    kubectl config use-context kind-cluster1
    export MBG1_CP=`kubectl get pods -l app=mbg -o custom-columns=:metadata.name --no-headers`
    export MBG1_DP=`kubectl get pods -l app=dataplane -o custom-columns=:metadata.name --no-headers`
    export MBG1IP=`kubectl get nodes -o "jsonpath={.items[0].status.addresses[0].address}"`
    export IPERF3CLIENT=`kubectl get pods -l app=iperf3-client -o custom-columns=:metadata.name --no-headers`

    kubectl config use-context kind-cluster2
    export MBG2_CP=`kubectl get pods -l app=mbg -o custom-columns=:metadata.name --no-headers`
    export MBG2_DP=`kubectl get pods -l app=dataplane -o custom-columns=:metadata.name --no-headers`
    export MBG2IP=`kubectl get nodes -o "jsonpath={.items[0].status.addresses[0].address}"`

Start the Gateway in Cluster 1:

    kubectl config use-context kind-cluster1
    kubectl exec -i $MBG1_CP -- ./controlplane start --id mbg1 --ip $MBG1IP --cport 30443 --cportLocal 443 --externalDataPortRange 30001 --dataplane mtls --certca ./mtls/ca.crt --cert ./mtls/mbg1.crt --key ./mtls/mbg1.key --startPolicyEngine=true --logFile=true --zeroTrust=false &
    kubectl exec -i $MBG1_DP -- ./dataplane --id mbg1 --dataplane mtls --certca ./mtls/ca.crt --cert ./mtls/mbg1.crt --key ./mtls/mbg1.key &


Initialize gwctl CLI:

    gwctl init --id "gwctl1" --gwIP $MBG1IP --gwPort 30443 --dataplane mtls --certca $PROJECT_FOLDER/demos/utils/mtls/ca.crt --cert $PROJECT_FOLDER/demos/utils//mtls/mbg1.crt --key $PROJECT_FOLDER/demos/utils/mtls/mbg1.key

Note : If you are using macOS to run the Kind cluster, instead of running gwctl in the macOS, it's better to run it within the individual Kind cluster in the following way. The subsequent gwctl commands need to be called from the respective KIND cluster.

    kubectl exec -i $MBG1_CP -- ./gwctl init --id "gwctl1" --gwIP $MBG1IP --gwPort 30443 --dataplane mtls --certca ./mtls/ca.crt --cert ./mtls/mbg1.crt --key ./mtls/mbg1.key
Start the Gateway in Cluster 2:

    kubectl config use-context kind-cluster2
    kubectl exec -i $MBG2_CP -- ./controlplane start --id mbg2 --ip $MBG2IP --cport 30443 --cportLocal 443  --dataplane mtls --certca ./mtls/ca.crt --cert ./mtls/mbg2.crt --key ./mtls/mbg2.key &
    kubectl exec -i $MBG2_DP -- ./dataplane --id mbg2 --dataplane mtls --certca $PROJECT/mtls/ca.crt --cert ./mtls/mbg2.crt --key ./mtls/mbg2.key &

Initialize gwctl CLI:

    gwctl init --id gwctl2 --gwIP $MBG2IP --gwPort 30443 --dataplane mtls --certca $PROJECT_FOLDER/demos/utils/mtls/ca.crt --cert $PROJECT_FOLDER/demos/utils/mtls/mbg2.crt --key $PROJECT_FOLDER/demos/utils/mtls/mbg2.key

When running Kind cluster on macOS run instead the following: 

    kubectl exec -i $MBG2_CP -- ./gwctl init --id "gwctl2" --gwIP $MBG2IP --gwPort 30443 --dataplane mtls --certca ./mtls/ca.crt --cert ./mtls/mbg2.crt --key ./mtls/mbg2.key

Note: The gateway certificate and key files are located in $PROJECT_FOLDER/demos/aux/mtls. The files are loaded to the gateway image (in step 1) and can be replaced.

### <ins> Step 4: Peers communication <ins>
In this step, we add a peer for each gateway using the gwctl:

    gwctl create peer --myid gwctl1 --name mbg2 --host $MBG2IP --port 30443
    gwctl create peer --myid gwctl2 --name mbg1 --host $MBG1IP --port 30443

When running Kind cluster on macOS run instead the following: 
    
    kubectl config use-context kind-cluster1
    kubectl exec -i $MBG1_CP -- ./gwctl create peer --myid gwctl1 --name mbg2 --host $MBG2IP --port 30443

    kubectl config use-context kind-cluster2
    kubectl exec -i $MBG2_CP -- ./gwctl create peer --myid gwctl2 --name mbg1 --host $MBG1IP --port 30443


### <ins> Step 5: Export a service <ins>
In this step, we add the iperf3 server to the cluster as an exported service that can be accessed from remote peers.  
Export the iperf3-server service to the Cluster 2 gateway:

    gwctl create export --myid gwctl2 --name iperf3-server --host iperf3-server --port 5000

When running Kind cluster on macOS run instead the following: 

    kubectl exec -i $MBG2_CP -- ./gwctl create export --myid gwctl2 --name iperf3-server --host iperf3-server --port 5000

Note: iperf3-client doesnt need to be added since it is not exported.

### <ins> Step 6: import iperf3 server service from Cluster 2 <ins>
In this step, we import the iperf3-server service from Cluster 2 gateway to Cluster 1 gateway
First, we specify which service we want to import and specify the local k8s endpoint (host:port) that will create for this service:

    gwctl create import --myid gwctl1 --name iperf3-server --host iperf3-server --port 5000

When running Kind cluster on macOS run instead the following:

    kubectl config use-context kind-cluster1
    kubectl exec -i $MBG1_CP -- ./gwctl create import --myid gwctl1 --name iperf3-server --host iperf3-server --port 5000

Second, we specify the peer we want to import the service:

    gwctl create binding --myid gwctl1 --import iperf3-server --peer mbg2

When running Kind cluster on macOS run instead the following:
 
    kubectl config use-context kind-cluster1
    kubectl exec -i $MBG1_CP -- ./gwctl create binding --myid gwctl1 --import iperf3-server --peer mbg2

### <ins> Final Step : Test Service connectivity <ins>
Start the iperf3 test from cluster 1:

    kubectl config use-context kind-cluster1
    kubectl exec -i $IPERF3CLIENT -- iperf3 -c iperf3-server --port 5000
    
### <ins> Debug <ins>
To see the control-plane state of the cluster:

    kubectl exec -i $MBG1_CP -- ./controlplane get state
To see the control-plane log of the cluster:

    kubectl exec -i $MBG1_CP -- ./controlplane get log

### <ins> Cleanup <ins>
Delete all Kind clusters:

    kind delete cluster --name=cluster1
    kind delete cluster --name=cluster2

### <ins> Automated Tests <ins>
To run the above tests as part of an automated script with three clusters.

    ./allinone.py
## Running the packaged Demo
Additionally, we have prepared individual scripts for a demo of the above scenario with policies.

### Start the Clusters and MBGs

    ./start_cluster_mbg.py -d mtls -m mbg1
    ./start_cluster_mbg.py -d mtls -m mbg2
    ./start_cluster_mbg.py -d mtls -m mbg3

### Connect the MBGs
The below command create connection between MBG1-MBG2 and MBG2-MBG3

    ./connect_mbgs.py

### Start iperf3 services at the three clusters 

    ./iperf3_service_create.py

### Expose iperf3-server server from Cluster 2 to other Clusters
    
    ./iperf3_service_expose.py

### Start iperf3 client from Cluster 1 to Cluster2

    ./iperf3_client_start.py -m mbg1
    ./iperf3_client_start.py -m mbg3

### Apply Policy to Block connection at MBG3
    ./apply_policy.py -m mbg3 -t deny
    ./apply_policy.py -m mbg3 -t show
    ./iperf3_client_start.py -m mbg3
    ./iperf3_client_start.py -m mbg1

### Apply Policy to Allow connection at MBG3
    ./apply_policy.py -m mbg3 -t allow
    ./apply_policy.py -m mbg3 -t show
    ./iperf3_client_start.py -m mbg3

### Apply Policy to Block connection at MBG2
    ./apply_policy.py -m mbg2 -t deny
    ./apply_policy.py -m mbg2 -t show
    ./iperf3_client_start.py -m mbg3

### Apply Policy to Allow connection at MBG3
    ./apply_policy.py -m mbg2 -t allow
    ./apply_policy.py -m mbg2 -t show
    ./iperf3_client-start.py -m mbg3
