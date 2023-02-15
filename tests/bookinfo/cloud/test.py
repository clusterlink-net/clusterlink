################################################################
#Name: Service node test
#Desc: create 1 proxy that send data to target ip
###############################################################
import os,sys
file_dir = os.path.dirname(__file__)
proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')
sys.path.insert(1,f'{proj_dir}/tests/utils/cloud/')


from tests.utils.mbgAux import runcmd, runcmdb, printHeader, waitPod, getPodName, getPodNameApp, getMbgPorts,getPodIp,clean_cluster,getPodNameIp

from tests.utils.cloud.check_k8s_cluster_ready import checkClusterIsReady,connectToCluster
from tests.utils.cloud.mbg_setup import mbgSetup,pushImage,mbgBuild
from tests.utils.cloud.create_k8s_cluster import createCluster
from tests.utils.cloud.clusterClass import cluster
from tests.utils.cloud.delete_k8s_cluster import deleteClustersList, cleanClustersList
from tests.utils.cloud.PROJECT_PARAMS import PROJECT_PATH
import argparse

mbg1 = cluster(name="mbg1", zone = "us-west1-b"   , platform = "gcp", type = "host")   #Oregon
mbg2 = cluster(name="mbg2", zone = "us-central1-b", platform = "gcp", type = "target") #Iowa
mbg3 = cluster(name="mbg3", zone = "us-east4-b"   , platform = "gcp", type = "target") #Virginia

destSvc    = "iperf3-server"
srcSvc1    = "productpage"
srcSvc2    = "productpage2"
destSvc    = "reviews"
review2pod = "reviews-v2"
review3pod = "reviews-v3"

mbgcPort="8443"
folMn=f"{PROJECT_PATH}/tests/bookinfo/manifests/"
folpdct   = f"{proj_dir}/tests/bookinfo/manifests/product/"
folReview = f"{proj_dir}/tests/bookinfo/manifests/review"


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-d','--dataplane', help='choose which dataplane to use mtls/tcp', required=False, default="tcp")
    parser.add_argument('-c','--command', help='Script command: test/delete', required=False, default="test")
    parser.add_argument('-delete','--deleteCluster', help='Delete clusters in the end of the test', required=False, default="true")

    args = vars(parser.parse_args())

    dataplane = args["dataplane"]
    command = args["command"]
    dltCluster = args["deleteCluster"]
    mbg1crtFlags    = f"--rootCa ./mtls/ca.crt --certificate ./mtls/mbg1.crt --key ./mtls/mbg1.key"  if dataplane =="mtls" else ""
    mbg2crtFlags    = f"--rootCa ./mtls/ca.crt --certificate ./mtls/mbg2.crt --key ./mtls/mbg2.key"  if dataplane =="mtls" else ""
    mbg3crtFlags    = f"--rootCa ./mtls/ca.crt --certificate ./mtls/mbg3.crt --key ./mtls/mbg3.key"  if dataplane =="mtls" else ""

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
        
    #Push mbg image
    pushImage(mbg1.platform)
    
    #Build MBG1
    checkClusterIsReady(mbg1)
    mbg1Ip=mbgBuild(mbgcPort=mbgcPort)
    mbgSetup(mbg1,dataplane,mbg1crtFlags,mbgctlName="mbgctl1",mbgIp=mbg1Ip, mbgcPort=mbgcPort)

    #Build MBG2
    checkClusterIsReady(mbg2)
    mbg2Ip=mbgBuild(mbgcPort=mbgcPort)
    mbgSetup(mbg2,dataplane,mbg2crtFlags,mbgctlName="mbgctl2",mbgIp=mbg2Ip,mbgcPort=mbgcPort)
    
    #Build MBG2
    checkClusterIsReady(mbg3)
    mbg3Ip=mbgBuild(mbgcPort=mbgcPort)
    mbgSetup(mbg3,dataplane,mbg3crtFlags,mbgctlName="mbgctl3",mbgIp=mbg3Ip,mbgcPort=mbgcPort)

    #Add MBG Peer
    connectToCluster(mbg1)
    printHeader("Add MBG2 MBG3 to MBG1")
    mbgctl1Pod =getPodName("mbgctl")
    runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl addPeer --id "{mbg2.name}" --ip {mbg2Ip} --cport {mbgcPort}')
    runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl addPeer --id "{mbg3.name}" --ip {mbg3Ip} --cport {mbgcPort}')

            
    # Send Hello
    printHeader("Send Hello commands")
    runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl hello')
        
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
    runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl addService --id {srcSvc1} --ip {srcSvcIp1} --description {srcSvc1}')
    runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl addService --id {srcSvc2} --ip {srcSvcIp2} --description {srcSvc2}')
   
    connectToCluster(mbg2)
    mbgctl2Pod =getPodName("mbgctl")
    runcmd(f"kubectl create -f {folReview}/review-v2.yaml")
    runcmd(f"kubectl create -f {folReview}/rating.yaml")
    printHeader(f"Add {destSvc} (server) service to destination cluster")
    waitPod(review2pod)
    destSvcReview2Ip = f"{getPodIp(review2pod)}:9080"
    runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl addService --id {destSvc} --ip {destSvcReview2Ip} --description v2')
    
    connectToCluster(mbg3)
    mbgctl3Pod =getPodName("mbgctl")
    runcmd(f"kubectl create -f {folReview}/review-v3.yaml")
    runcmd(f"kubectl create -f {folReview}/rating.yaml")
    printHeader(f"Add {destSvc} (server) service to destination cluster")
    waitPod(review3pod)
    destSvcReview3Ip = f"{getPodIp(review3pod)}:9080"
    runcmd(f'kubectl exec -i {mbgctl3Pod} -- ./mbgctl addService --id {destSvc} --ip {destSvcReview3Ip} --description v3')

     #Expose service
    connectToCluster(mbg2)
    printHeader(f"\n\nStart exposing svc {destSvc}")
    runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl expose --serviceId {destSvc}')
    connectToCluster(mbg3)
    printHeader(f"\n\nStart exposing svc {destSvc}")
    runcmd(f'kubectl exec -i {mbgctl3Pod} -- ./mbgctl expose --serviceId {destSvc}')

    #Get services
    connectToCluster(mbg1)
    printHeader("\n\nStart get service")
    runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl getService')

    #set Policy
    connectToCluster(mbg1)
    mbgctl1Pod =getPodName("mbgctl")
    runcmdb(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl policy --command lb_add --policy ecmp')
    
    #connect
    podMbg1= getPodName("mbg-deployment")        
    mbg1LocalPort, mbg1ExternalPort = getMbgPorts(podMbg1, destSvc+"-mbg2")
    runcmd(f"kubectl delete service {destSvc}")
    runcmd(f"kubectl create service clusterip {destSvc} --tcp=9080:{mbg1LocalPort}")
    runcmd(f"kubectl patch service {destSvc} -p "+  "\'{\"spec\":{\"selector\":{\"app\": \"mbg\"}}}\'") #replacing app name

    #Type 
    mbg1.setClusterIP()
    print(f"Proctpage1 url: http://{mbg1.ip}:30001/productpage")
    print(f"Proctpage2 url: http://{mbg1.ip}:30002/productpage")