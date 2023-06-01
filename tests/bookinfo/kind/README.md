
# BookInfo application Test
This test set [Istio BookInfo application](https://istio.io/latest/docs/examples/bookinfo/) in different clusters.   
This test create three kind clusters:  
* The Product and details microservices run on the first cluster.  
* The Reviews(V2) and Rating microservices run on second cluster.   
* The Reviews(V3) and Rating microservices run on third cluster.   
   
## <ins> Pre-requires installations <ins>
To run a Kind test, check all pre-requires are installed (Go, docker, Kubectl, Kind):

    export PROJECT_FOLDER=`git rev-parse --show-toplevel`
    cd $PROJECT_FOLDER
    make prereqs
## <ins> BookInfo test<ins>
* To Build the kind clusters and run the bookInfo application:    
Use BookInfo script:
  * Run bookinfo script:
        
        make run-kind-bookinfo
  * The BookInfo application can be viewed by connecting to the Product microservice:
   
        export MBG1IP=`kubectl get nodes -o jsonpath={.items[0].status.addresses[0].address}`
        firefox http://$MBG1IP:30001/productpage

or use the following steps:

### <ins> Step 1: Build docker image <ins>
Build MBG docker image:
    
    make docker-build

### <ins> Step 2: Create kind clusters with MBG image <ins>
In this step, we build the kind cluster with an MBG image.  
Build the first kind cluster with MBG, gwctl, Product and details micro-services:
1) Create a Kind cluster with MBG image:

        kind create cluster  --name=mbg-agent1
        kind load docker-image mbg --name=mbg-agent1

2) Create a MBG deployment: 
    
        kubectl create -f $PROJECT_FOLDER/config/manifests/mbg/mbg.yaml

3) Create a gwctl deployment: 
   
        kubectl create -f $PROJECT_FOLDER/config/manifests/gwctl/gwctl.yaml
4) Create product and details microservices: 
   
        docker pull maistra/examples-bookinfo-productpage-v1
        kind load docker-image maistra/examples-bookinfo-productpage-v1 --name=mbg-agent1
        docker pull maistra/examples-bookinfo-details-v1:0.12.0
        kind load docker-image maistra/examples-bookinfo-details-v1:0.12.0 --name=mbg-agent1
        kubectl create -f $PROJECT_FOLDER/tests/bookinfo/manifests/product/product.yaml
        kubectl create -f $PROJECT_FOLDER/tests/bookinfo/manifests/product/details.yaml

Build the second kind cluster with MBG, gwctl, reviews(v2) and rating microservices:
1) Create a Kind cluster with MBG image:

        kind create cluster --name=mbg-agent2
        kind load docker-image mbg --name=mbg-agent2
2) Create a MBG deployment:
   
        kubectl create -f $PROJECT_FOLDER/config/manifests/mbg/mbg.yaml
3) Create a gwctl deployment: 

        kubectl create -f $PROJECT_FOLDER/config/manifests/gwctl/gwctl.yaml
4) Create reviews and ratings microservices:
   
        docker pull maistra/examples-bookinfo-reviews-v2
        kind load docker-image maistra/examples-bookinfo-reviews-v2 --name=mbg-agent2
        kubectl create -f $PROJECT_FOLDER/tests/bookinfo/manifests/review/review-v2.yaml 
        docker pull maistra/examples-bookinfo-ratings-v1:0.12.0
        kind load docker-image maistra/examples-bookinfo-ratings-v1:0.12.0 --name=mbg-agent2
        kubectl create -f $PROJECT_FOLDER/tests/bookinfo/manifests/review//rating.yaml

Build the third kind cluster with MBG, gwctl, reviews(v3) and rating microservices:
1) Create a Kind cluster with MBG image:

        kind create cluster --name=mbg-agent3
        kind load docker-image mbg --name=mbg-agent3
2) Create a MBG deployment:
   
        kubectl create -f $PROJECT_FOLDER/config/manifests/mbg/mbg.yaml
3) Create a gwctl deployment: 

        kubectl create -f $PROJECT_FOLDER/config/manifests/gwctl/gwctl.yaml
4) Create review and ratings microservices:
   
        docker pull maistra/examples-bookinfo-reviews-v3
        kind load docker-image maistra/examples-bookinfo-reviews-v3 --name=mbg-agent3
        kubectl create -f $PROJECT_FOLDER/tests/bookinfo/manifests/review/review-v3.yaml
        docker pull maistra/examples-bookinfo-ratings-v1:0.12.0
        kind load docker-image maistra/examples-bookinfo-ratings-v1:0.12.0 --name=mbg-agent3
        kubectl create -f $PROJECT_FOLDER/tests/bookinfo/manifests/review/rating.yaml

Check that container statuses are Running.

    kubectl get pods
### <ins> Step 3: Start running MBG and gwctl  <ins>
In this step, start to run the MBG and gwctl.  
First, Initialize the parameters of the test (pods' names and IPs):
    
    kubectl config use-context kind-mbg-agent1
    export MBG1=`kubectl get pods -l app=mbg -o custom-columns=:metadata.name`
    export MBG1IP=`kubectl get nodes -o jsonpath={.items[0].status.addresses[0].address}`
    export MBG1PODIP=`kubectl get pod $MBG1 --template '{{.status.podIP}}'`
    export MBGCTL1=`kubectl get pods -l app=gwctl -o custom-columns=:metadata.name`
    export MBGCTL1IP=`kubectl get pod $MBGCTL1 --template '{{.status.podIP}}'`
    export PRODUCTPAGEPOD_IP=`kubectl get pods -l app=productpage -o jsonpath={.items[*].status.podIP}`

    kubectl config use-context kind-mbg-agent2
    export MBG2=`kubectl get pods -l app=mbg -o custom-columns=:metadata.name`
    export MBG2IP=`kubectl get nodes -o jsonpath={.items[0].status.addresses[0].address}`
    export MBG2PODIP=`kubectl get pod $MBG2 --template '{{.status.podIP}}'`
    export MBGCTL2=`kubectl get pods -l app=gwctl -o custom-columns=:metadata.name`
    export MBGCTL2IP=`kubectl get pod $MBGCTL2 --template '{{.status.podIP}}'`
    export REVIEWS2POD_IP=`kubectl get pods -l app=reviews-v2 -o jsonpath={.items[*].status.podIP}`    
    
    kubectl config use-context kind-mbg-agent3
    export MBG3=`kubectl get pods -l app=mbg -o custom-columns=:metadata.name`
    export MBG3IP=`kubectl get nodes -o jsonpath={.items[0].status.addresses[0].address}`
    export MBG3PODIP=`kubectl get pod $MBG3 --template '{{.status.podIP}}'`
    export MBGCTL3=`kubectl get pods -l app=gwctl -o custom-columns=:metadata.name`
    export MBGCTL3IP=`kubectl get pod $MBGCTL3 --template '{{.status.podIP}}'`
    export REVIEWS3POD_IP=`kubectl get pods -l app=reviews-v3 -o jsonpath={.items[*].status.podIP}`
    
Start MBG1: (the MBG creates an HTTP server, so it is better to run this command in a different terminal (using tmux) or run it in the background)

    kubectl config use-context kind-mbg-agent1
    kubectl exec -i $MBG1 -- ./controlplane start --id "MBG1" --ip $MBG1IP --cport 30443 --cportLocal 8443 --dataplane mtls --rootCa ./mtls/ca.crt --certificate ./mtls/mbg1.crt --key ./mtls/mbg1.key

Initialize gwctl (mbg control):

    kubectl exec -i $MBGCTL1 -- ./gwctl start --id "gwctl1" --ip $MBGCTL1IP --mbgIP $MBG1PODIP:8443 --dataplane mtls --rootCa ./mtls/ca.crt --certificate ./mtls/mbg1.crt --key ./mtls/mbg1.key

Create K8s service nodeport to connect MBG cport to the MBG localcport.

    kubectl create service nodeport mbg --tcp=8443:8443 --node-port=30443

Start MBG2: (the MBG creates an HTTP server, so it is better to run this command in a different terminal (using tmux) or run it in the background)

    kubectl config use-context kind-mbg-agent2
    kubectl exec -i $MBG2 -- ./controlplane start --id "MBG2" --ip $MBG2IP --cport 30443 --cportLocal 8443 --dataplane mtls --rootCa ./mtls/ca.crt --certificate ./mtls/mbg2.crt --key ./mtls/mbg2.key 

Initialize gwctl (mbg control):

    kubectl exec -i $MBGCTL2 -- ./gwctl start --id "gwctl2" --ip $MBGCTL2IP --mbgIP $MBG2PODIP:8443 --dataplane mtls --rootCa ./mtls/ca.crt --certificate ./mtls/mbg2.crt --key ./mtls/mbg2.key

Create K8s service nodeport to connect MBG cport to the MBG localcport.

    kubectl create service nodeport mbg --tcp=8443:8443 --node-port=30443

Start MBG3: (the MBG creates an HTTP server, so it is better to run this command in a different terminal (using tmux) or run it in the background)

    kubectl config use-context kind-mbg-agent3
    kubectl exec -i $MBG3 -- ./controlplane start --id "MBG3" --ip $MBG3IP --cport 30443 --cportLocal 8443 --dataplane mtls --rootCa ./mtls/ca.crt --certificate ./mtls/mbg3.crt --key ./mtls/mbg3.key 

Initialize gwctl (mbg control):

    kubectl exec -i $MBGCTL3 -- ./gwctl start --id "gwctl3" --ip $MBGCTL3IP --mbgIP $MBG3PODIP:8443 --dataplane mtls --rootCa ./mtls/ca.crt --certificate ./mtls/mbg3.crt --key ./mtls/mbg3.key

Create K8s service nodeport to connect MBG cport to the MBG localcport.

    kubectl create service nodeport mbg --tcp=8443:8443 --node-port=30443

Note: The MBG certificate and key files are located in $PROJECT_FOLDER/tests/aux/mtls. The files are loaded to the MBG image (in step 1) and can be replaced.

### <ins> Step 4: MBG peers communication <ins>
In this step, we set the communication between the MBGs.  
First, send MBG2, MBG3 details information to MBG1 using gwctl:

    kubectl config use-context kind-mbg-agent1
    kubectl exec -i $MBGCTL1 -- ./gwctl addPeer --id "MBG2" --ip $MBG2IP --cport 30443
    kubectl exec -i $MBGCTL1 -- ./gwctl addPeer --id "MBG3" --ip $MBG3IP --cport 30443

Send Hello message from MBG1 to MBG2, MBG3:

    kubectl exec -i $MBGCTL1 -- ./gwctl hello
### <ins> Step 5: Add services <ins>
In this step, we add servicers to the MBGs.  
Add service product to MBG1.

    kubectl config use-context kind-mbg-agent2
    kubectl exec -i $MBGCTL1 -- ./gwctl addService --id "productpage" --id $PRODUCTPAGEPOD_IP
        
Add services reviews-v2 and reviews-v3 services to MBG2 and MBG3 respectively.

    kubectl config use-context kind-mbg-agent2
    kubectl exec -i $MBGCTL2 -- ./gwctl addService --id reviews-v2 --ip $REVIEWS2POD_IP:9080
    kubectl config use-context kind-mbg-agent3
    kubectl exec -i $MBGCTL3 -- ./gwctl addService --id reviews-v3 --ip $REVIEWS3POD_IP:9080

### <ins> Step 6: Expose service <ins>
In this step, we expose the reviews services from MBG2 and MBG3 to MBG1.

    kubectl config use-context kind-mbg-agent2
    kubectl exec -i $MBGCTL2 -- ./gwctl expose --serviceId reviews-v2
    kubectl config use-context kind-mbg-agent3
    kubectl exec -i $MBGCTL3 -- ./gwctl expose --serviceId reviews-v3

### <ins> Step 7: Create K8s service for application <ins>
In this step, we create K8s service to connect product microservice to the reviews-v2 listen port that created by the MBG.
   
    kubectl config use-context kind-mbg-agent1
    export MBG1PORT_REVIEWSV2=`python3 $PROJECT_FOLDER/tests/aux/getMbgLocalPort.py -m $MBG1 -s reviews-v2`
    kubectl create service clusterip reviews --tcp=9080:$MBG1PORT_REVIESWV2
    kubectl patch service reviews -p '{"spec":{"selector":{"app": "mbg"}}}'

To connect to reviews-v3 microservices
    
    kubectl config use-context kind-mbg-agent1
    kubectl delete service reviews
    export MBG1PORT_REVIEWSV3=`python3 $PROJECT_FOLDER/tests/aux/getMbgLocalPort.py -m $MBG1 -s reviews-v3`
    kubectl create service clusterip reviews --tcp=9080:$MBG1PORT_REVIEWSV3
    kubectl patch service reviews -p '{"spec":{"selector":{"app": "mbg"}}}'

### <ins> Step 8: Run BookInfo application <ins>
In this step, we run the bookinfo application using firefox web browser.  
The application is running on two different kind clusters and the secure communication done by the MBGs.
    
Run the BookInfo application:  

        firefox http://$MBG1IP:30001/productpage

### <ins> Cleanup <ins>
Delete all Kind cluster.

    make clean-kind
