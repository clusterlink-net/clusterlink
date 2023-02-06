##############################################################################################
# Name: Bookinfo
# Info: support bookinfo application with mbgctl inside the clusters 
#       In this we create three kind clusters
#       1) MBG1- contain mbg, mbgctl,product and details microservices (bookinfo services)
#       2) MBG2- contain mbg, mbgctl, review-v2 and rating microservices (bookinfo services)
#       3) MBG3- contain mbg, mbgctl, review-v3 and rating microservices (bookinfo services)
##############################################################################################

import os,time
import subprocess as sp
import sys
import argparse
proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

from tests.utils.mbgAux import runcmd, runcmdb, printHeader, getPodName, waitPod,getMbgPorts,buildMbg,buildMbgctl,getPodIp
from tests.utils.kind.kindAux import useKindCluster


def connectSvc(srcSvc,destSvc, k8sSvcName):
    printHeader(f"\n\nStart connect {srcSvc} to {destSvc}")
    useKindCluster(mbg1ClusterName)    
    podMbg1= getPodName("mbg-deployment")        
    mbg1LocalPort, mbg1ExternalPort = getMbgPorts(podMbg1, destSvc)
    runcmd(f"kubectl delete service {destSvc}")
    runcmd(f"kubectl create service clusterip {k8sSvcName} --tcp={srcK8sSvcPort}:{mbg1LocalPort}")
    runcmd(f"kubectl patch service {k8sSvcName} -p "+  "\'{\"spec\":{\"selector\":{\"app\": \"mbg\"}}}\'") #replacing app name

############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-c','--command', help='Script command: test/connect/disconnect', required=False, default="test")
    parser.add_argument('-d','--dataplane', help='choose which dataplane to use mtls/tcp', required=False, default="tcp")

    parser.add_argument('-src','--src', help='Source service name', required=False)
    parser.add_argument('-dst','--dest', help='Destination service name', required=False)
    args = vars(parser.parse_args())

    printHeader("\n\nStart Kind Test\n\n")
    printHeader("Start pre-setting")
    
    folpdct   = f"{proj_dir}/tests/bookinfo/manifests/product/"
    folReview = f"{proj_dir}/tests/bookinfo/manifests/review"
    dataplane = args["dataplane"]
 

    destSvc         = "reviews"
    #MBG1 parameters 
    mbg1DataPort    = "30001"
    mbg1cPort       = "30443"
    mbg1cPortLocal  = "8443"
    mbg1ClusterName = "mbg-agent1"
    mbg1crtFlags    = f"--rootCa ./mtls/ca.crt --certificate ./mtls/mbg1.crt --key ./mtls/mbg1.key"  if dataplane =="mtls" else ""
    mbgctl1Name     = "mbgctl1"
    srcSvc          = "productpage"
    srcK8sSvcPort   = "9080"
    srcK8sSvcIp     = ":"+srcK8sSvcPort
    srcDefaultGW    = "10.244.0.1"
    

    #MBG2 parameters 
    mbg2DataPort    = "30001"
    mbg2cPort       = "30443"
    mbg2cPortLocal  = "8443"
    mbg2crtFlags    = f"--rootCa ./mtls/ca.crt --certificate ./mtls/mbg2.crt --key ./mtls/mbg2.key"  if dataplane =="mtls" else ""
    mbg2ClusterName = "mbg-agent2"
    mbgctl2Name     = "mbgctl2"
    review2DestPort = "30001"
    review2pod      = "reviews-v2"

    #MBG3 parameters 
    mbg3DataPort    = "30001"
    mbg3cPort       = "30443"
    mbg3cPortLocal  = "8443"
    mbg3crtFlags    = f"--rootCa ./mtls/ca.crt --certificate ./mtls/mbg3.crt --key ./mtls/mbg3.key"  if dataplane =="mtls" else ""
    mbg3ClusterName = "mbg-agent3"
    mbgctl3Name     = "mbgctl3"
    review3DestPort = "30001"
    review3pod      = "reviews-v3"
    

    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)
    if args["command"] == "disconnect":
        useKindCluster(mbg1ClusterName)
        mbgctlPod= getPodName("mbgctl")
        printHeader("\n\nClose Iperf3 connection")
        runcmd(f'kubectl exec -i {mbgctlPod} -- ./mbgctl disconnect --serviceId {args["src"]} --serviceIdDest {args["dest"]}')
    elif args["command"] == "connect":
        connectSvc(args["src"],args["dest"],destSvc)
    else:
        ### clean 
        print(f"Clean old kinds")
        os.system("make clean-kind-bookinfo")
        
        ### build docker environment 
        printHeader(f"Build docker image")
        os.system("make docker-build")
        ## build Kind clusters environment 
        ###first Mbg
        printHeader("\n\nStart building MBG1")
        podMbg1, mbg1Ip= buildMbg(mbg1ClusterName)
        ###Second Mbg
        printHeader("\n\nStart building MBG2")
        podMbg2, mbg2Ip= buildMbg(mbg2ClusterName)
        
        ###Third Mbg
        printHeader("\n\nStart building MBG3")
        podMbg3, mbg3Ip= buildMbg(mbg3ClusterName)

        #Set First MBG
        useKindCluster(mbg1ClusterName)
        runcmdb(f'kubectl exec -i {podMbg1} -- ./mbg start --id "MBG1" --ip {mbg1Ip} --cport {mbg1cPort} --cportLocal {mbg1cPortLocal} --externalDataPortRange {mbg1DataPort} \
        --dataplane {args["dataplane"]}  {mbg1crtFlags} --startPolicyEngine {True}')
        runcmd(f"kubectl create service nodeport mbg --tcp={mbg1cPortLocal}:{mbg1cPortLocal} --node-port={mbg1cPort}")
        
        #Set Second MBG
        useKindCluster(mbg2ClusterName)
        runcmdb(f'kubectl exec -i {podMbg2} --  ./mbg start --id "MBG2" --ip {mbg2Ip} --cport {mbg2cPort} --cportLocal {mbg2cPortLocal} --externalDataPortRange {mbg2DataPort}\
        --dataplane {args["dataplane"]}  {mbg2crtFlags}  --startPolicyEngine {True}')
        runcmd(f"kubectl create service nodeport mbg --tcp={mbg2cPortLocal}:{mbg2cPortLocal} --node-port={mbg2cPort}")
        
        #Set Third MBG
        useKindCluster(mbg3ClusterName)
        runcmdb(f'kubectl exec -i {podMbg3} --  ./mbg start --id "MBG3" --ip {mbg3Ip} --cport {mbg3cPort} --cportLocal {mbg3cPortLocal} --externalDataPortRange {mbg3DataPort}\
        --dataplane {args["dataplane"]}  {mbg3crtFlags}  --startPolicyEngine {True}')
        runcmd(f"kubectl create service nodeport mbg --tcp={mbg3cPortLocal}:{mbg3cPortLocal} --node-port={mbg3cPort}")
        
        
        ###Set mbgctl1
        useKindCluster(mbg1ClusterName)
        runcmd(f"kind load docker-image maistra/examples-bookinfo-productpage-v1 --name={mbg1ClusterName}")
        runcmd(f"kind load docker-image maistra/examples-bookinfo-details-v1:0.12.0 --name={mbg1ClusterName}")
        runcmd(f"kubectl create -f {folpdct}/product.yaml")
        runcmd(f"kubectl create -f {folpdct}/details.yaml")
        mbgctl1Pod, mbgctl1Ip= buildMbgctl(mbgctl1Name, mbgMode="inside")
        destMbg1Ip = f"{getPodIp(podMbg1)}:{mbg1cPortLocal}" 
        runcmdb(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl start --id {mbgctl1Name}   --ip {mbgctl1Ip} --mbgIP  {destMbg1Ip} --dataplane {args["dataplane"]} {mbg1crtFlags}')
        printHeader(f"Add {srcSvc} (client) service to host cluster")
        srcSvcIp =getPodIp(srcSvc)  
        runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl addService --id {srcSvc} --ip {srcSvcIp} --description product')
        runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl addPolicyEngine --target {getPodIp(podMbg1)}:9990')

        # Add MBG Peer
        printHeader("Add MBG2, MBG3 peer to MBG1")
        runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl addPeer --id "MBG2" --ip {mbg2Ip} --cport {mbg2cPort}')
        runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl addPeer --id "MBG3" --ip {mbg3Ip} --cport {mbg3cPort}')
        # Send Hello
        printHeader("Send Hello commands")
        runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl hello')
        
        ###Set mbgctl2
        useKindCluster(mbg2ClusterName)
        runcmd(f"kind load docker-image maistra/examples-bookinfo-reviews-v2 --name={mbg2ClusterName}")
        runcmd(f"kind load docker-image maistra/examples-bookinfo-ratings-v1:0.12.0 --name={mbg2ClusterName}")
        runcmd(f"kubectl create -f {folReview}/review-v2.yaml")
        runcmd(f"kubectl create -f {folReview}/rating.yaml")
        mbgctl2pod, mbgctl2Ip= buildMbgctl(mbgctl2Name, mbgMode="inside")   
        destMbg2Ip = f"{getPodIp(podMbg2)}:{mbg2cPortLocal}" 
        runcmdb(f'kubectl exec -i {mbgctl2pod} -- ./mbgctl start --id {mbgctl2Name}  --ip {mbgctl2Ip} --mbgIP {destMbg2Ip} --dataplane {args["dataplane"]} {mbg2crtFlags}')
        printHeader(f"Add {destSvc} (server) service to destination cluster")
        waitPod(review2pod)
        destSvcReview2Ip = f"{getPodIp(review2pod)}:{srcK8sSvcPort}"
        runcmd(f'kubectl exec -i {mbgctl2pod} -- ./mbgctl addService --id {destSvc} --ip {destSvcReview2Ip} --description v2')
        runcmd(f'kubectl exec -i {mbgctl2pod} -- ./mbgctl addPolicyEngine --target {getPodIp(podMbg2)}:9990')


        ###Set mbgctl3
        useKindCluster(mbg3ClusterName)
        runcmd(f"kind load docker-image maistra/examples-bookinfo-reviews-v3 --name={mbg3ClusterName}")
        runcmd(f"kind load docker-image maistra/examples-bookinfo-ratings-v1:0.12.0 --name={mbg3ClusterName}")
        runcmd(f"kubectl create -f {folReview}/review-v3.yaml")
        runcmd(f"kubectl create -f {folReview}/rating.yaml")
        mbgctl3pod, mbgctl3Ip= buildMbgctl(mbgctl3Name , mbgMode="inside")   
        destMbg3Ip = f"{getPodIp(podMbg3)}:{mbg3cPortLocal}" 
        runcmdb(f'kubectl exec -i {mbgctl3pod} -- ./mbgctl start --id {mbgctl3Name}  --ip {mbgctl3Ip} --mbgIP {destMbg3Ip} --dataplane {args["dataplane"]} {mbg3crtFlags}')
        printHeader(f"Add {destSvc} (server) service to destination cluster")
        waitPod(review3pod)
        destSvcReview3Ip = f"{getPodIp(review3pod)}:{srcK8sSvcPort}"
        runcmd(f'kubectl exec -i {mbgctl3pod} -- ./mbgctl addService --id {destSvc} --ip {destSvcReview3Ip} --description v3')
        runcmd(f'kubectl exec -i {mbgctl3pod} -- ./mbgctl addPolicyEngine --target {getPodIp(podMbg3)}:9990')

        #Add host cluster to MBG1
        useKindCluster(mbg1ClusterName)
        printHeader("Add mbgctl to MBG1")
        runcmd(f'kubectl exec -i {podMbg1} -- ./mbg addMbgctl --id {mbgctl1Name} --ip {mbgctl1Ip}')

        #Add dest cluster to MBG2
        useKindCluster(mbg2ClusterName)
        printHeader("Add mbgctl2 to MBG2")
        runcmd(f'kubectl exec -i {podMbg2} -- ./mbg addMbgctl --id {mbgctl2Name} --ip {mbgctl2Ip}')
        
        #Add dest cluster to MBG3
        useKindCluster(mbg3ClusterName)
        printHeader("Add mbgctl3 to MBG3")
        runcmd(f'kubectl exec -i {podMbg3} -- ./mbg addMbgctl --id {mbgctl3Name} --ip {mbgctl3Ip}')
        
        #Expose service
        useKindCluster(mbg2ClusterName)
        printHeader(f"\n\nStart exposing svc {destSvc}")
        runcmd(f'kubectl exec -i {mbgctl2pod} -- ./mbgctl expose --serviceId {destSvc}')
        useKindCluster(mbg3ClusterName)
        printHeader(f"\n\nStart exposing svc {destSvc}")
        runcmd(f'kubectl exec -i {mbgctl3pod} -- ./mbgctl expose --serviceId {destSvc}')

        #Get services
        useKindCluster(mbg1ClusterName)
        printHeader("\n\nStart get service")
        runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl getService')
    
        #set services
        runcmdb(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl policy --command lb_set --policy ecmp')
        #connect
        connectSvc(srcSvc, destSvc+"-MBG2",destSvc)
