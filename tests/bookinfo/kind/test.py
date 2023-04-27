#!/usr/bin/env python3
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

from tests.utils.mbgAux import runcmd, runcmdb, printHeader, getPodName, waitPod,getMbgPorts,buildMbg,buildMbgctl,getPodIp,getPodNameIp
from tests.utils.kind.kindAux import useKindCluster,startKindClusterMbg,getKindIp


############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
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
    mbg1Name        = "mbg1"
    mbg1crtFlags    = f"--rootCa ./mtls/ca.crt --certificate ./mtls/mbg1.crt --key ./mtls/mbg1.key"  if dataplane =="mtls" else ""
    mbgctl1Name     = "mbgctl1"
    srcSvc1         = "productpage"
    srcSvc2         = "productpage2"
    srcK8sSvcPort   = "9080"
    srcK8sSvcIp     = ":"+srcK8sSvcPort
    srcDefaultGW    = "10.244.0.1"
    

    #MBG2 parameters 
    mbg2DataPort    = "30001"
    mbg2cPort       = "30443"
    mbg2cPortLocal  = "8443"
    mbg2crtFlags    = f"--rootCa ./mtls/ca.crt --certificate ./mtls/mbg2.crt --key ./mtls/mbg2.key"  if dataplane =="mtls" else ""
    mbg2Name        = "mbg2"
    mbgctl2Name     = "mbgctl2"
    review2DestPort = "30001"
    review2pod      = "reviews-v2"
    
    #MBG3 parameters 
    mbg3DataPort    = "30001"
    mbg3cPort       = "30443"
    mbg3cPortLocal  = "8443"
    mbg3crtFlags    = f"--rootCa ./mtls/ca.crt --certificate ./mtls/mbg3.crt --key ./mtls/mbg3.key"  if dataplane =="mtls" else ""
    mbg3Name        = "mbg3"
    mbgctl3Name     = "mbgctl3"
    review3DestPort = "30001"
    review3pod      = "reviews-v3"

    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)

    ### clean 
    print(f"Clean old kinds")
    os.system("make clean-kind-bookinfo")
    
    ### build docker environment 
    printHeader(f"Build docker image")
    os.system("make docker-build")
    
    ## build Kind clusters environment 
    startKindClusterMbg(mbg1Name, mbgctl1Name, mbg1cPortLocal, mbg1cPort, mbg1DataPort, dataplane ,mbg1crtFlags)        
    startKindClusterMbg(mbg2Name, mbgctl2Name, mbg2cPortLocal, mbg2cPort, mbg2DataPort,dataplane ,mbg2crtFlags)        
    startKindClusterMbg(mbg3Name, mbgctl3Name, mbg3cPortLocal, mbg3cPort, mbg3DataPort,dataplane ,mbg3crtFlags)        
    
    ###get mbg parameters
    useKindCluster(mbg1Name)
    mbg1Pod, _           = getPodNameIp("mbg")
    mbg1Ip               = getKindIp("mbg1")
    mbgctl1Pod, mbgctl1Ip= getPodNameIp("mbgctl")
    useKindCluster(mbg2Name)
    mbg2Pod, _            = getPodNameIp("mbg")
    mbgctl2Pod, mbgctl2Ip = getPodNameIp("mbgctl")
    mbg2Ip                =getKindIp(mbg2Name)
    useKindCluster(mbg3Name)
    mbg3Pod, _            = getPodNameIp("mbg")
    mbg3Ip                = getKindIp("mbg3")
    mbgctl3Pod, mbgctl3Ip = getPodNameIp("mbgctl")

    ###Set mbg1 services
    useKindCluster(mbg1Name)
    runcmd(f"kind load docker-image maistra/examples-bookinfo-productpage-v1 --name={mbg1Name}")
    runcmd(f"kind load docker-image maistra/examples-bookinfo-details-v1:0.12.0 --name={mbg1Name}")
    runcmd(f"kubectl create -f {folpdct}/product.yaml")
    runcmd(f"kubectl create -f {folpdct}/product2.yaml")
    runcmd(f"kubectl create -f {folpdct}/details.yaml")
    printHeader(f"Add {srcSvc1} {srcSvc2}  services to host cluster")
    waitPod(srcSvc1)
    waitPod(srcSvc2)
    _ , srcSvcIp1 =getPodNameIp(srcSvc1)
    _ , srcSvcIp2 =getPodNameIp(srcSvc2)
    runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl add service --id {srcSvc1}  --description {srcSvc1}')
    runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl add service --id {srcSvc2} --description {srcSvc2}')

    

    # Add MBG Peer
    printHeader("Add MBG2, MBG3 peer to MBG1")
    runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl add peer --id {mbg2Name} --target {mbg2Ip} --port {mbg2cPort}')
    runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl add peer --id {mbg3Name} --target {mbg3Ip} --port {mbg3cPort}')
    # Send Hello
    printHeader("Send Hello commands")
    runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl hello --myid {mbgctl1Name}')
    
    ###Set mbg2 service
    useKindCluster(mbg2Name)
    runcmd(f"kind load docker-image maistra/examples-bookinfo-reviews-v2 --name={mbg2Name}")
    runcmd(f"kind load docker-image maistra/examples-bookinfo-ratings-v1:0.12.0 --name={mbg2Name}")
    runcmd(f"kubectl create -f {folReview}/review-v2.yaml")
    runcmd(f"kubectl create -f {folReview}/rating.yaml")
    printHeader(f"Add {destSvc} (server) service to destination cluster")
    waitPod(destSvc)
    destSvcReview2Ip = f"{getPodIp(destSvc)}"
    destSvcReview2Port = f"{srcK8sSvcPort}"
    runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl add service --id {destSvc} --target {destSvcReview2Ip} --port {destSvcReview2Port} --description v2')
    

    ###Set mbgctl3
    useKindCluster(mbg3Name)
    runcmd(f"kind load docker-image maistra/examples-bookinfo-reviews-v3 --name={mbg3Name}")
    runcmd(f"kind load docker-image maistra/examples-bookinfo-ratings-v1:0.12.0 --name={mbg3Name}")
    runcmd(f"kubectl create -f {folReview}/review-v3.yaml")
    runcmd(f"kubectl create -f {folReview}/rating.yaml")
    printHeader(f"Add {destSvc} (server) service to destination cluster")
    waitPod(destSvc)
    destSvcReview3Ip = f"{getPodIp(destSvc)}"
    destSvcReview3Port = f"{srcK8sSvcPort}"
    runcmd(f'kubectl exec -i {mbgctl3Pod} -- ./mbgctl add service --id {destSvc} --target {destSvcReview3Ip} --port {destSvcReview3Port} --description v3')

    #Expose service
    useKindCluster(mbg2Name)
    printHeader(f"\n\nStart exposing svc {destSvc}")
    runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl expose --service {destSvc}')
    useKindCluster(mbg3Name)
    printHeader(f"\n\nStart exposing svc {destSvc}")
    runcmd(f'kubectl exec -i {mbgctl3Pod} -- ./mbgctl expose --service {destSvc}')

    #Get services
    useKindCluster(mbg1Name)
    printHeader("\n\nStart get service")
    runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl get service --myid {mbgctl1Name}')
    runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl get policy --myid {mbgctl1Name}')

    #Create k8s service for destSvc
    useKindCluster(mbg1Name)    
    podMbg1= getPodName("mbg-deployment")        
    mbg1LocalPort, mbg1ExternalPort = getMbgPorts(podMbg1, destSvc)
    runcmd(f"kubectl delete service {destSvc}")
    runcmd(f"kubectl create service clusterip {destSvc} --tcp={srcK8sSvcPort}:{mbg1LocalPort}")
    runcmd(f"kubectl patch service {destSvc} -p "+  "\'{\"spec\":{\"selector\":{\"app\": \"mbg\"}}}\'") #replacing app name

    print(f"Proctpage1 url: http://{mbg1Ip}:30001/productpage")
    print(f"Proctpage2 url: http://{mbg1Ip}:30002/productpage")


