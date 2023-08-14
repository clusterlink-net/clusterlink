#!/usr/bin/env python3

################################################################
#Name: Service node test
#Desc: create 1 proxy that send data to target ip
###############################################################
import os,sys
file_dir = os.path.dirname(__file__)
proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')
sys.path.insert(1,f'{proj_dir}/demos/utils/cloud/')


from demos.utils.mbgAux import runcmd, runcmdb, printHeader, waitPod, getPodName, getPodNameApp, getMbgPorts,getPodIp,clean_cluster,getPodNameIp

from demos.utils.cloud.check_k8s_cluster_ready import checkClusterIsReady,connectToCluster
from demos.utils.cloud.mbg_setup import mbgSetup,pushImage,mbgBuild
from demos.utils.cloud.create_k8s_cluster import createCluster
from demos.utils.cloud.clusterClass import cluster
from demos.utils.cloud.delete_k8s_cluster import deleteClustersList, cleanClustersList
from demos.utils.cloud.PROJECT_PARAMS import PROJECT_PATH
import argparse

mbg1gcp = cluster(name="mbg1", zone = "us-west1-b"   , platform = "gcp", type = "host")   #Oregon
mbg1ibm = cluster(name="mbg1", zone = "sjc04"        , platform = "ibm", type = "host")   #San jose
mbg2gcp = cluster(name="mbg2", zone = "us-central1-b", platform = "gcp", type = "target") #Iowa
mbg2ibm = cluster(name="mbg2", zone = "dal10"        , platform = "ibm", type = "target") #Dallas
mbg3gcp = cluster(name="mbg3", zone = "us-east4-b"   , platform = "gcp", type = "target") #Virginia
mbg3ibm = cluster(name="mbg3", zone = "wdc04"        , platform = "ibm", type = "target") #Washington DC

destSvc    = "iperf3-server"
srcSvc1    = "productpage"
srcSvc2    = "productpage2"
destSvc    = "reviews"


mbgcPort="443"
folMn=f"{PROJECT_PATH}/demos/bookinfo/manifests/"
folpdct   = f"{proj_dir}/demos/bookinfo/manifests/product/"
folReview = f"{proj_dir}/demos/bookinfo/manifests/review"


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-d','--dataplane', help='choose which dataplane to use mtls/tcp', required=False, default="mtls")
    parser.add_argument('-c','--command', help='Script command: test/delete/clean', required=False, default="test")
    parser.add_argument('-cloud','--cloud', help='Cloud setup using gcp/ibm/diff (different clouds)', required=False, default="gcp")
    parser.add_argument('-delete','--deleteCluster', help='Delete clusters in the end of the test', required=False, default="true")

    args = vars(parser.parse_args())

    dataplane = args["dataplane"]
    command = args["command"]
    dltCluster = args["deleteCluster"]
    cloud = args["cloud"]

    mbg1crtFlags    = f"--certca ./mtls/ca.crt --cert ./mtls/mbg1.crt --key ./mtls/mbg1.key"  if dataplane =="mtls" else ""
    mbg2crtFlags    = f"--certca ./mtls/ca.crt --cert ./mtls/mbg2.crt --key ./mtls/mbg2.key"  if dataplane =="mtls" else ""
    mbg3crtFlags    = f"--certca ./mtls/ca.crt --cert ./mtls/mbg3.crt --key ./mtls/mbg3.key"  if dataplane =="mtls" else ""
    mbg1 = mbg1gcp if cloud in ["gcp","diff"] else mbg1ibm
    mbg2 = mbg2gcp if cloud in ["gcp","diff"] else mbg2ibm
    mbg3 = mbg3gcp if cloud in ["gcp"]        else mbg3ibm
    gwctl1 ="gwctl1"
    gwctl2 ="gwctl2"
    gwctl3 ="gwctl3"
    if command =="delete":
        deleteClustersList([mbg1, mbg2, mbg3])
        exit()
    elif command =="clean":
        cleanClustersList([mbg1, mbg2, mbg3])
        exit()
    
    #Create k8s cluster
    createCluster(cluster=mbg1,run_in_bg=True)
    createCluster(cluster=mbg2,run_in_bg=True)
    createCluster(cluster=mbg3,run_in_bg=False)
    
    #Build MBG1
    checkClusterIsReady(mbg1)
    mbg1Ip=mbgBuild(mbgcPort=mbgcPort)
    mbgSetup(mbg1,dataplane,mbg1crtFlags,gwctlName=gwctl1,mbgIp=mbg1Ip, mbgcPort=mbgcPort)

    #Build MBG2
    checkClusterIsReady(mbg2)
    mbg2Ip=mbgBuild(mbgcPort=mbgcPort)
    mbgSetup(mbg2,dataplane,mbg2crtFlags,gwctlName=gwctl2,mbgIp=mbg2Ip,mbgcPort=mbgcPort)
    
    #Build MBG3
    checkClusterIsReady(mbg3)
    mbg3Ip=mbgBuild(mbgcPort=mbgcPort)
    mbgSetup(mbg3,dataplane,mbg3crtFlags,gwctlName=gwctl3,mbgIp=mbg3Ip,mbgcPort=mbgcPort)

    #Add MBG Peer
    connectToCluster(mbg1)
    printHeader("Add MBG2 MBG3 to MBG1")
    gwctl1Pod =getPodName("gwctl")
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl create peer --name "{mbg2.name}" --host {mbg2Ip} --port {mbgcPort}')
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl create peer --name "{mbg3.name}" --host {mbg3Ip} --port {mbgcPort}')

            
    # Send Hello
    printHeader("Send Hello commands")
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl hello')
        
    #Add services 
    connectToCluster(mbg1)
    runcmd(f"kubectl create -f {folpdct}/product.yaml")
    runcmd(f"kubectl create -f {folpdct}/product2.yaml")
    runcmd(f"kubectl create -f {folpdct}/details.yaml")
    printHeader(f"Add {srcSvc1} {srcSvc2}  services to host cluster")
    waitPod(srcSvc1)
    waitPod(srcSvc2)
    _ , srcSvcIp1 =getPodNameIp(srcSvc1)
    _ , srcSvcIp2 =getPodNameIp(srcSvc2)
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl create export --name {srcSvc1} --host {srcSvcIp1} --description {srcSvc1}')
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl create export --name {srcSvc2} --host {srcSvcIp2} --description {srcSvc2}')
   
    connectToCluster(mbg2)
    gwctl2Pod =getPodName("gwctl")
    runcmd(f"kubectl create -f {folReview}/review-v2.yaml")
    runcmd(f"kubectl create -f {folReview}/rating.yaml")
    printHeader(f"Add {destSvc} (server) service to destination cluster")
    waitPod(destSvc)
    destSvcReview2Ip   = f"{getPodIp(destSvc)}"
    destSvcReview2Port = "9080"
    runcmd(f'kubectl exec -i {gwctl2Pod} -- ./gwctl create export --name {destSvc} --host {destSvcReview2Ip} --port {destSvcReview2Port} --description v2')
    
    connectToCluster(mbg3)
    gwctl3Pod =getPodName("gwctl")
    runcmd(f"kubectl create -f {folReview}/review-v3.yaml")
    runcmd(f"kubectl create -f {folReview}/rating.yaml")
    printHeader(f"Add {destSvc} (server) service to destination cluster")
    waitPod(destSvc)
    destSvcReview3Ip = f"{getPodIp(destSvc)}"
    destSvcReview3Port = "9080"
    runcmd(f'kubectl exec -i {gwctl3Pod} -- ./gwctl create export --name {destSvc} --host {destSvcReview3Ip}  --port {destSvcReview3Port} --description v3')

     #Expose service
    connectToCluster(mbg2)
    printHeader(f"\n\nStart exposing svc {destSvc}")
    runcmd(f'kubectl exec -i {gwctl2Pod} -- ./gwctl expose --service {destSvc}')
    connectToCluster(mbg3)
    printHeader(f"\n\nStart exposing svc {destSvc}")
    runcmd(f'kubectl exec -i {gwctl3Pod} -- ./gwctl expose --service {destSvc}')

    #Get services
    connectToCluster(mbg1)
    printHeader("\n\nStart get service")
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl get service')

    #connect
    podMbg1= getPodName("mbg-deployment")        
    mbg1LocalPort, mbg1ExternalPort = getMbgPorts(podMbg1, destSvc)
    runcmd(f"kubectl delete service {destSvc}")
    runcmd(f"kubectl create service clusterip {destSvc} --tcp=9080:{mbg1LocalPort}")
    runcmd(f"kubectl patch service {destSvc} -p "+  "\'{\"spec\":{\"selector\":{\"app\": \"mbg\"}}}\'") #replacing app name

    #Type 
    mbg1.setClusterIP()
    print(f"Proctpage1 url: http://{mbg1.ip}:30001/productpage")
    print(f"Proctpage2 url: http://{mbg1.ip}:30002/productpage")