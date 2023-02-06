################################################################
#Name: Service node test
#Desc: create 1 proxy that send data to target ip
###############################################################
import os,sys
file_dir = os.path.dirname(__file__)
proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}/tests/utils')
print(f"{proj_dir}")
from kindAux import runcmd, runcmdb, printHeader, waitPod, getPodName, getPodNameApp, getMbgPorts,getPodIp,clean_cluster,getPodNameIp

sys.path.append(file_dir+"/utils")

from utils.cloud.check_k8s_cluster_ready import checkClusterIsReady,connectToCluster
from utils.cloud.mbg_setup import mbgSetup,pushImage,mbgBuild
from utils.cloud.create_k8s_cluster import createCluster
from utils.cloud.clusterClass import cluster
from utils.cloud.delete_k8s_cluster import deleteCluster
from utils.cloud.PROJECT_PARAMS import PROJECT_PATH
import argparse

mbg1 = cluster(name="mbg1",   zone = "us-west1-b",    platform = "gcp", type = "host")
mbg2 = cluster(name="mbg2", zone = "us-west1-b",    platform = "gcp", type = "target")

destSvc  = "iperf3-server"
srcSvc   = "iperf3-client"
mbgcPort="8443"
folMn=f"{PROJECT_PATH}/tests/iperf3/manifests/"

def deleteClusters():
    print("Start delete all cluster")
    deleteCluster(mbg1,run_in_bg=True)
    deleteCluster(mbg2,run_in_bg=False)

def cleanClusters():
    print("Start delete all cluster")
    connectToCluster(mbg1)
    clean_cluster()
    connectToCluster(mbg2)
    clean_cluster()


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

    if command =="delete":
        deleteClusters()
        exit()
    elif command =="clean":
        cleanClusters()
        exit()
    
    #Create k8s cluster
    createCluster(cluster=mbg1,run_in_bg=True)
    createCluster(cluster=mbg2,run_in_bg=False)
        
    #Push mbg image
    pushImage(mbg1.platform)
    
    #Build MBG1
    checkClusterIsReady(mbg1)
    mbg1Ip=mbgBuild(mbgcPort=mbgcPort)
    
    #Build MBG2
    checkClusterIsReady(mbg2)
    mbg2Ip=mbgBuild(mbgcPort=mbgcPort )

    # #Setup mbg1
    connectToCluster(mbg1)
    mbgSetup(mbg1,dataplane,mbg1crtFlags,mbgctlName="mbgctl1",mbgIp=mbg1Ip, mbgcPort=mbgcPort)
    mbgctl1Pod =getPodName("mbgctl")

    
    # #Setup mbg2
    connectToCluster(mbg2)
    mbgSetup(mbg2,dataplane,mbg2crtFlags,mbgctlName="mbgctl2",mbgIp=mbg2Ip,mbgcPort=mbgcPort)
    mbgctl2Pod =getPodName("mbgctl")

    #Add MBG Peer
    connectToCluster(mbg2)
    printHeader("Add MBG1 MBG2")
    runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl addPeer --id "MBG1" --ip {mbg1Ip} --cport {mbgcPort}')

            
    # Send Hello
    printHeader("Send Hello commands")
    runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl hello')
        
    #Add services 
    connectToCluster(mbg1)
    runcmd(f"kubectl create -f {folMn}/iperf3-client/iperf3-client.yaml")
    waitPod(srcSvc)
    podIperf3 =getPodIp(srcSvc)
    runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl addService --id {srcSvc} --ip {podIperf3} --description {srcSvc}')
    
    connectToCluster(mbg2)
    runcmd(f"kubectl create -f {folMn}/iperf3-server/iperf3.yaml")
    waitPod(destSvc)
    destSvcIp =getPodIp(destSvc)
    runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl addService --id {destSvc} --ip {destSvcIp}:5000 --description {destSvc}')
    
    #Expose destination service
    printHeader("\n\nStart exposing connection")
    runcmdb(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl expose --serviceId {destSvc}')

    #Test MBG1
    for i in range(10):
        connectToCluster(mbg1)
        podIperf3= getPodName(srcSvc)
        mbgPod,mbgPodIP=getPodNameIp("mbg")
        mbg1LocalPort, mbg1ExternalPort = getMbgPorts(mbgPod,destSvc+"-"+mbg2.name)

        printHeader("The iPerf3 test connects directly to the destination")
        cmd = f'kubectl exec -i {podIperf3} --  iperf3 -c {mbgPodIP} -p {mbg1LocalPort} -t 40'
        #iperf3Test(cmd)
        runcmd(cmd)

    #clean target and source clusters
    #delete_all_clusters()