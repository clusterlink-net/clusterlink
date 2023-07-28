################################################################
#Name: speedtest application test
#Desc: create 3 MBG  with speed test server and firefox clients
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

mbg1gcp = cluster(name="mbg1", zone = "us-west1-b", platform = "gcp", type = "host") 
mbg1ibm = cluster(name="mbg2", zone = "dal10",      platform = "ibm", type = "host")
mbg2gcp = cluster(name="mbg2", zone = "us-west1-b", platform = "gcp", type = "target")
mbg2ibm = cluster(name="mbg2", zone = "dal10",      platform = "ibm", type = "target")
mbg3gcp = cluster(name="mbg3", zone = "us-east4-b"   , platform = "gcp", type = "target") #Virginia
mbg3ibm = cluster(name="mbg3", zone = "syd04"        , platform = "ibm", type = "target") #Sydney

srcSvc1         = "firefox"
srcSvc2         = "firefox2"
destSvc         = "openspeedtest"
mbgcPort="443"
folman   = f"{proj_dir}/tests/speedtest/manifests/"


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-d','--dataplane', help='choose which dataplane to use mtls/tcp', required=False, default="mtls")
    parser.add_argument('-c','--command', help='Script command: test/delete', required=False, default="test")
    parser.add_argument('-m','--machineType', help='Type of machine to create small/large', required=False, default="small")
    parser.add_argument('-cloud','--cloud', help='Cloud setup using gcp/ibm/diff (different clouds)', required=False, default="gcp")
    parser.add_argument('-delete','--deleteCluster', help='Delete clusters in the end of the test', required=False, default="true")

    args = vars(parser.parse_args())

    dataplane = args["dataplane"]
    command = args["command"]
    cloud = args["cloud"]
    dltCluster = args["deleteCluster"]
    machineType = args["machineType"]
    mbg1crtFlags    = f"--certca ./mtls/ca.crt --cert ./mtls/mbg1.crt --key ./mtls/mbg1.key"  if dataplane =="mtls" else ""
    mbg2crtFlags    = f"--certca ./mtls/ca.crt --cert ./mtls/mbg2.crt --key ./mtls/mbg2.key"  if dataplane =="mtls" else ""
    mbg3crtFlags    = f"--certca ./mtls/ca.crt --cert ./mtls/mbg3.crt --key ./mtls/mbg3.key"  if dataplane =="mtls" else ""
    mbg1 = mbg1gcp if cloud in ["gcp","diff"] else mbg1ibm
    mbg2 = mbg2gcp if cloud in ["gcp","diff"] else mbg2ibm
    mbg3 = mbg3gcp if cloud in ["gcp"]        else mbg3ibm
    
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
    mbgSetup(mbg1,dataplane,mbg1crtFlags,gwctlName="gwctl1",mbgIp=mbg1Ip, mbgcPort=mbgcPort)

    #Build MBG2
    checkClusterIsReady(mbg2)
    mbg2Ip=mbgBuild(mbgcPort=mbgcPort)
    mbgSetup(mbg2,dataplane,mbg2crtFlags,gwctlName="gwctl2",mbgIp=mbg2Ip,mbgcPort=mbgcPort)
    
    #Build MBG3
    checkClusterIsReady(mbg3)
    mbg3Ip=mbgBuild(mbgcPort=mbgcPort)
    mbgSetup(mbg3,dataplane,mbg3crtFlags,gwctlName="gwctl3",mbgIp=mbg3Ip,mbgcPort=mbgcPort)


    #Add MBG Peer
    connectToCluster(mbg2)
    gwctl2Pod =getPodName("gwctl")
    printHeader("Add MBG1, MBG3 to MBG2")
    runcmd(f'kubectl exec -i {gwctl2Pod} -- ./gwctl create peer --name "MBG1" --host {mbg1Ip} --port {mbgcPort}')
    runcmd(f'kubectl exec -i {gwctl2Pod} -- ./gwctl create peer --name "MBG3" --host {mbg3Ip} --port {mbgcPort}')
            
    # Send Hello
    printHeader("Send Hello commands")
    runcmd(f'kubectl exec -i {gwctl2Pod} -- ./gwctl hello')
        
    #Add services 
    connectToCluster(mbg1)
    gwctl1Pod =getPodName("gwctl")
    runcmd(f"kubectl create -f {folman}/firefox.yaml")    
    printHeader(f"Add {srcSvc1} services to host cluster")
    waitPod(srcSvc1)
    _ , srcSvcIp1 =getPodNameIp(srcSvc1)
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl create export --name {srcSvc1} --host {srcSvcIp1} --description {srcSvc1}')
    runcmd(f"kubectl create service nodeport {srcSvc1} --tcp=5800:5800 --node-port=30000")
    mbg1.setClusterIP()

    connectToCluster(mbg2)
    runcmd(f"kubectl create -f {folman}/speedtest.yaml")
    printHeader(f"Add {destSvc} (server) service to destination cluster")
    waitPod(destSvc)
    destSvcIp = f"{getPodIp(destSvc)}:3000"
    runcmd(f'kubectl exec -i {gwctl2Pod} -- ./gwctl create export --name {destSvc} --host {destSvcIp} --description v2')
    
    connectToCluster(mbg3)
    gwctl3Pod =getPodName("gwctl")
    runcmd(f"kubectl create -f {folman}/firefox.yaml")
    runcmd(f"kubectl create -f {folman}/firefox2.yaml")    
    printHeader(f"Add {srcSvc1} {srcSvc2} services to host cluster")
    waitPod(srcSvc1)
    waitPod(srcSvc2)
    _ , srcSvcIp1 =getPodNameIp(srcSvc1)
    _ , srcSvcIp2 =getPodNameIp(srcSvc2)
    runcmd(f'kubectl exec -i {gwctl3Pod} -- ./gwctl create export --name {srcSvc1} --host {srcSvcIp1} --description {srcSvc1}')
    runcmd(f'kubectl exec -i {gwctl3Pod} -- ./gwctl create export --name {srcSvc2} --host {srcSvcIp2} --description {srcSvc2}')
    runcmd(f"kubectl create service nodeport {srcSvc1} --tcp=5800:5800 --node-port=30000")
    runcmd(f"kubectl create service nodeport {srcSvc2} --tcp=5800:5800 --node-port=30001")
    mbg3.setClusterIP()

    #Expose destination service
    connectToCluster(mbg2)
    printHeader("\n\nStart exposing connection")
    runcmdb(f'kubectl exec -i {gwctl2Pod} -- ./gwctl expose --service {destSvc}')

  #Set K8s network services
    connectToCluster(mbg1)
    mbg1Pod, _ = getPodNameIp("mbg")
    printHeader("\n\nStart get service")
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl get service')
    mbg1LocalPort, mbg1ExternalPort = getMbgPorts(mbg1Pod, destSvc)
    runcmd(f"kubectl create service clusterip {destSvc} --tcp=3000:{mbg1LocalPort}")
    runcmd(f"kubectl patch service {destSvc} -p "+  "\'{\"spec\":{\"selector\":{\"app\": \"mbg\"}}}\'") #replacing app name

    connectToCluster(mbg3)
    printHeader("\n\nStart get service")
    mbg3Pod, _ = getPodNameIp("mbg")
    runcmd(f'kubectl exec -i {gwctl3Pod} -- ./gwctl get service')
    mbg3LocalPort, mbg3ExternalPort = getMbgPorts(mbg3Pod, destSvc)
    runcmd(f"kubectl create service clusterip {destSvc} --tcp=3000:{mbg3LocalPort}")
    runcmd(f"kubectl patch service {destSvc} -p "+  "\'{\"spec\":{\"selector\":{\"app\": \"mbg\"}}}\'") #replacing app name
    
    #Firefox communications
    printHeader(f"Firefox urls")
    print(f"To use the mbg1 firefox client, run the command:\n    firefox http://{mbg1.ip}:30000/")
    print(f"To use the first mbg3 firefox client, run the command:\n    firefox http://{mbg3.ip}:30000/")
    print(f"To use the second mbg3 firefox client, run the command:\n   firefox http://{mbg3.ip}:30001/")
    
    print(f"The OpenSpeedTest url: http://{destSvc}:3000/ ")

