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

mbg1gcp = cluster(name="mbg1", zone = "us-west1-b", platform = "gcp", type = "host") 
mbg1ibm = cluster(name="mbg1", zone = "dal10",      platform = "ibm", type = "host")
mbg2gcp = cluster(name="mbg2", zone = "us-west1-b", platform = "gcp", type = "target")
mbg2ibm = cluster(name="mbg2", zone = "dal10",      platform = "ibm", type = "target")

destSvc  = "iperf3-server"
srcSvc   = "iperf3-client"
mbgcPort="443"
folMn=f"{PROJECT_PATH}/demos/iperf3/manifests/"


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
    mbg1 = mbg1gcp if cloud in ["gcp","diff"] else mbg1ibm
    mbg2 = mbg2gcp if cloud in ["gcp"]        else mbg2ibm
    gwctl1 ="gwctl1"
    gwctl2 ="gwctl2"
    
    if command =="delete":
        deleteClustersList([mbg1, mbg2])
        exit()
    elif command =="clean":
        cleanClustersList([mbg1, mbg2])
        exit()

    #Create k8s cluster
    createCluster(cluster=mbg1, run_in_bg=True , machineType = machineType)
    createCluster(cluster=mbg2, run_in_bg=False, machineType = machineType)
        
    # #Setup MBG1
    checkClusterIsReady(mbg1)
    mbg1Ip=mbgBuild(mbgcPort=mbgcPort)
    mbgSetup(mbg1, dataplane, mbg1crtFlags, gwctlName=gwctl1, mbgIp=mbg1Ip, mbgcPort=mbgcPort)
    
    #Build MBG2
    checkClusterIsReady(mbg2)
    mbg2Ip=mbgBuild(mbgcPort=mbgcPort)
    mbgSetup(mbg2, dataplane, mbg2crtFlags, gwctlName=gwctl2, mbgIp=mbg2Ip, mbgcPort=mbgcPort)
    

    #Add MBG Peer
    connectToCluster(mbg2)
    gwctl2Pod =getPodName("gwctl")
    printHeader("Add MBG1 MBG2")
    runcmd(f'kubectl exec -i {gwctl2Pod} -- ./gwctl create peer --name {mbg1.name} --host {mbg1Ip} --port {mbgcPort}')

            
    # Send Hello
    printHeader("Send Hello commands")
    runcmd(f'kubectl exec -i {gwctl2Pod} -- ./gwctl hello')
        
    #Add services 
    connectToCluster(mbg1)
    gwctl1Pod =getPodName("gwctl")
    runcmd(f"kubectl create -f {folMn}/iperf3-client/iperf3-client.yaml")
    waitPod(srcSvc)
    podIperf3 =getPodIp(srcSvc)
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl create export --name {srcSvc} --host {podIperf3} --description {srcSvc}')
    
    connectToCluster(mbg2)
    runcmd(f"kubectl create -f {folMn}/iperf3-server/iperf3.yaml")
    runcmd(f"kubectl create service nodeport iperf3-server --tcp=5000:5000 --node-port=30001")
    waitPod(destSvc)
    destSvcIp =getPodIp(destSvc)
    runcmd(f'kubectl exec -i {gwctl2Pod} -- ./gwctl create export --name {destSvc} --host {destSvcIp}:5000 --description {destSvc}')
    
    #Expose destination service
    printHeader("\n\nStart exposing connection")
    runcmdb(f'kubectl exec -i {gwctl2Pod} -- ./gwctl expose --service {destSvc}')

    #Test MBG1
    connectToCluster(mbg1)
    podIperf3= getPodName(srcSvc)
    mbgPod,mbgPodIP=getPodNameIp("mbg")
    mbg1LocalPort, mbg1ExternalPort = getMbgPorts(mbgPod,destSvc)
    for i in range(2):
        printHeader(f"iPerf3 test {i}")
        cmd = f'kubectl exec -i {podIperf3} --  iperf3 -c {mbgPodIP} -p {mbg1LocalPort} -t 40'
        runcmd(cmd)

    #clean target and source clusters
    #delete_all_clusters()