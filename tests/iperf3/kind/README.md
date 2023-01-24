# <ins>iPerf3 Connectivity and Performance Test<ins>
In this test we check iPerf3 connectivity between different kind cluster using the MBG components.  
This setup use two Kind clusters- 
1) MBG1 cluster - contain MBG, mbgctl (MBG control component), and iPerf3 client
2) MBG2 cluster - contain MBG, mbgctl (MBG control component), and iPerf3 serve


## <ins> Pre-requires installations <ins>
To run a Kind test, check all pre-requires are installed(Go, docker, Kubectl, Kind ):

    export PROJECT_FOLDER=`git rev-parse --show-toplevel`
    cd $PROJECT_FOLDER
    make prereqs

## <ins> iPerf3 test<ins>
Use a single script to run, build kind clusters, and run the iperf3 test. 

    make run-kind-iperf3

or use the following steps:

### <ins> Step 1: Build docker image <ins>
Build MBG docker image:
    
    make docker-build

### <ins> Step 2: Create kind clusters with MBG image <ins>
In this step, we build the kind cluster with an MBG image.
Build the first kind cluster with MBG, mbgctl, and iperf3-client:
1) Create a Kind cluster with MBG image:

        kind create cluster --config  $PROJECT_FOLDER/manifests/kind/mbg-config1.yaml --name=mbg-agent1  
        kind load docker-image mbg --name=mbg-agent1

2) Create a MBG deployment: 
    
        kubectl create -f $PROJECT_FOLDER/manifests/mbg/mbg.yaml  
        kubectl create -f $PROJECT_FOLDER/manifests/mbg/mbg-client-svc.yaml

3) Create a mbgctl deployment: 
   
        kubectl create -f $PROJECT_FOLDER/manifests/mbgctl/mbgctl.yaml
        kubectl create -f $PROJECT_FOLDER/manifests/mbgctl/mbgctl-svc.yaml
4) Create an iPerf3-client deployment: 
   
        kubectl create -f  $PROJECT_FOLDER/tests/iperf3/manifests/iperf3-client/iperf3-client.yaml

Build the second kind cluster with MBG, mbgctl, and iperf3-server:
1) Create a Kind cluster with MBG image:

        kind create cluster --config $PROJECT_FOLDER/manifests/kind/mbg-config2.yaml --name=mbg-agent2
        kind load docker-image mbg --name=mbg-agent2
2) Create a MBG deployment:
   
        kubectl create -f $PROJECT_FOLDER/manifests/mbg/mbg.yaml
        kubectl create -f $PROJECT_FOLDER/manifests/mbg/mbg-client-svc.yaml
3) Create a mbgctl deployment: 

        kubectl create -f $PROJECT_FOLDER/manifests/mbgctl/mbgctl.yaml
        kubectl create -f $PROJECT_FOLDER/manifests/mbgctl/mbgctl-svc.yaml
4) Create an iPerf3-server deployment:
   
        kubectl create -f $PROJECT_FOLDER/tests/iperf3/manifests/iperf3-server/iperf3.yaml
        kubectl create -f $PROJECT_FOLDER/tests/iperf3/manifests/iperf3-server/iperf3-svc.yaml

Check that container statuses are Running.

    kubectl get pods

### <ins> Step 3: Start running MBG and mbgctl  <ins>
In this step, start to run the MBG and mbgctl.  
First, Initialize parameters of Pods name and IPs:
    
    kubectl config use-context kind-mbg-agent1
    export MBG1=`kubectl get pods -l app=mbg -o custom-columns=:metadata.name`
    export MBG1IP=`kubectl get nodes  -o jsonpath={.items[0].status.addresses[0].address}`
    export MBG1PODIP=`kubectl get pod $MBG1 --template '{{.status.podIP}}'`
    export MBGCTL1=`kubectl get pods -l app=mbgctl -o custom-columns=:metadata.name`
    export MBGCTL1IP=`kubectl get pod $MBGCTL1 --template '{{.status.podIP}}'`
    export IPERF3CLIENT_IP=`kubectl get pods -l app=iperf3-client -o jsonpath={.items[*].status.podIP}`
    export IPERF3CLIENT=`kubectl get pods -l app=iperf3-client -o custom-columns=:metadata.name`

    kubectl config use-context kind-mbg-agent2
    export MBG2=`kubectl get pods -l app=mbgctl -o custom-columns=:metadata.name`
    export MBG2IP=`kubectl get nodes  -o jsonpath={.items[0].status.addresses[0].address}`
    export MBG2PODIP=`kubectl get pod $MBG2 --template '{{.status.podIP}}'`
    export MBGCTL2=`kubectl get pods -l app=mbgctl -o custom-columns=:metadata.name`
    export MBGCTL2IP=`kubectl get pod $MBGCTL2 --template '{{.status.podIP}}'`
    export IPERF3SERVER_IP=`kubectl get pods -l app=iperf3-server -o jsonpath={.items[*].status.podIP}`

Start MBG1:( the MBG creates an HTTP server, so it is better to run this command in a different terminal (using tmux) or run it in the background)

    kubectl config use-context kind-mbg-agent1
    kubectl exec -i $MBG1 -- ./mbg start --id "MBG1" --ip $MBG1IP --cport 30443 --cportLocal 8443 --dataplane mtls --rootCa ./mtls/ca.crt --certificate ./mtls/mbg1.crt --key ./mtls/mbg1.key
Initialize mbgctl (mbg control):  
    kubectl exec -i $MBGCTL1 -- ./mbgctl start --id "hostCluster"  --ip $MBGCTL1IP --mbgIP $MBG1PODIP:8443  --dataplane mtls --rootCa ./mtls/ca.crt --certificate ./mtls/mbg1.crt --key ./mtls/mbg1.key

Create K8s service nodeport to connect MBG cport to the MBG localcport.

    kubectl create service nodeport mbg --tcp=8443:8443 --node-port=30443    

Start MBG2:( the MBG creates an HTTP server, so it is better to run this command in a different terminal (using tmux) or run it in the background)

    kubectl config use-context kind-mbg-agent2
    kubectl exec -i $MBG2 -- ./mbg start --id "MBG2" --ip $MBG2IP --cport 30443 --cportLocal 8443 --dataplane mtls --rootCa ./mtls/ca.crt --certificate ./mtls/mbg2.crt --key ./mtls/mbg2.key 
Initialize mbgctl (mbg control):  
    kubectl exec -i $MBGCTL2 -- ./mbgctl start --id "destCluster"-ip $MBGCTL2IP  --mbgIP $MBG2PODIP:8443 --dataplane mtls --rootCa ./mtls/ca.crt --certificate ./mtls/mbg2.crt --key ./mtls/mbg2.key

Create K8s service nodeport to connect MBG cport to the MBG localcport.

    kubectl create service nodeport mbg --tcp=8443:8443 --node-port=30443

Note: The MBG certificate and key files are located in $PROJECT_FOLDER/tests/aux/mtls. The files are loaded to the MBG image (in step 1) and can be replaced.

### <ins> Step 4: MBG peers communication <ins>
In this step, we set the communication between the MBGs.  
First, send MBG2 details information to MBG1 using mbgctl:

    kubectl config use-context kind-mbg-agent1
    kubectl exec -i $MBGCTL1 -- ./mbgctl addPeer --id "MBG2" --ip $MBG2IP --cport 30443    
Send Hello message from MBG1 to MBG2:

    kubectl exec -i $MBGCTL1 -- ./mbgctl hello
### <ins> Step 5: Add services <ins>
In this step, we add the iperf3 services for each mbg.  
Add an iperf3-client service to MBG1:

    kubectl exec -i $MBGCTL1 -- ./mbgctl addService --serviceId iperf3-client --serviceIp $IPERF3CLIENT_IP
Add an iperf3-server service to MBG2:

    kubectl config use-context kind-mbg-agent2
    kubectl exec -i $MBGCTL2 -- ./mbgctl addService --serviceId iperf3-server --serviceIp $IPERF3SERVER_IP:5000

### <ins> Step 6: Expose service <ins>
In this step, we expose the iperf3-server service from MBG2 to MBG1.

    kubectl exec -i $MBGCTL2 -- ./mbgctl expose --serviceId iperf3-server

### <ins> Step 7: iPerf3 test <ins>
Add an iperf3-server service to MBG2.In this test, we can check the secure communication between the iPerf3 client and server by sending the traffic using the MBGs.
    
    kubectl config use-context kind-mbg-agent1
    export MBG1PORT_IPERF3SERVER=`python3  $PROJECT_FOLDER/tests/aux/getMbgLocalPort.py -m $MBG1 -s iperf3-server`
    kubectl exec -i  $IPERF3CLIENT --  iperf3 -c $MBG1PODIP -p $MBG1PORT_IPERF3SERVER


### <ins> Cleanup <ins>
Delete all Kind cluster.

    make clean-kind