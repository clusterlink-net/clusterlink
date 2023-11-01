# <ins>OpenSpeedTest application<ins>
In this demo we use OpenSpeedTest application for checking connectivity between different kind clusters using the Clusterlink components.  
This demo shows different access policies defined by various attributes such as source service, destination service, and destination gateway.
This setup uses three Kind clusters- 
1. Cluster 1- contains GW, gwctl (GW CLI component), and firefox client.
2. Cluster 2- contains GW, gwctl (GW CLI component), and OpenSpeedTest server.
3. cluster 3- contains GW, gwctl (GW CLI component), and two firefox clients.
     
System illustration:


![alt text](../../..//docs/openspeedtest.png)
## <ins> Pre-requires installations <ins>
To run a Kind demo, check that all pre-requires are installed (Go, docker, Kubectl, Kind):

    export PROJECT_FOLDER=`git rev-parse --show-toplevel`
    cd $PROJECT_FOLDER
    make prereqs

## <ins> OpenSpeedTest test<ins>
Use a single script to build the kind clusters. 

    python3 ./test.py

To run the OpenSpeedTest from GW1, connect with the web browser to the firefox client.
1. Connect to firefox client :
   
        kubectl config use-context kind-peer1  
        export GW1IP=`kubectl get nodes -o jsonpath={.items[0].status.addresses[0].address}`  
        firefox http://$GW1IP:30000/
2. In the firefox client connect to OpenSpeedTest server:  
   
        http://openspeedtest:3000/ 
3. Run the SpeedTest using the gui

To run the OpenSpeedTest from GW3, connect with the web browser to the firefox client.
1. Connect to one of the firefox clients :  

        kubectl config use-context kind-peer3  
        export GW3IP=`kubectl get nodes -o jsonpath={.items[0].status.addresses[0].address}`  
        firefox http://$GW3IP:30000/               #First Client
        firefox http://$GW3IP:30001/               #Second client
2. In the firefox client connect to OpenSpeedTest server:  
   
        http://openspeedtest:3000/ 
3. Run the SpeedTest using the gui

### Apply Policy to Block connection at GW3
    python3 ./apply_policy.py -m peer3 -t deny
    python3 ./apply_policy.py -m peer3 -t show
    
### Apply Policy to Allow connection at GW3
    python3 ./apply_policy.py -m peer3 -t allow
    python3 ./apply_policy.py -m peer3 -t show
    

### Apply Policy to Block connection at GW2
    python3 ./apply_policy.py -m peer2 -t deny
    python3 ./apply_policy.py -m peer2 -t show
    
### Apply Policy to Allow connection at GW3
    python3 ./apply_policy.py -m peer2 -t allow
    python3 ./apply_policy.py -m peer2 -t show
    
### <ins> Cleanup <ins>
Delete all Kind clusters.

    make clean-kind
