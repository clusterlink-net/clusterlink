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
    parser.add_argument('-d','--dataplane', help='choose which dataplane to use mtls/tcp', required=False, default="mtls")
    parser.add_argument('-c','--cni', help='choose diff to use different cnis', required=False, default="same")

    args = vars(parser.parse_args())

    printHeader("\n\nStart Kind Test\n\n")
    printHeader("Start pre-setting")
    
    folman   = f"{proj_dir}/tests/speedtest/manifests/"
    dataplane = args["dataplane"]
    cni       = args["cni"]


    srcSvc1         = "firefox"
    srcSvc2         = "firefox2"
    destSvc         = "openspeedtest"
    
    #MBG1 parameters 
    mbg1DataPort    = "30001"
    mbg1cPort       = "30443"
    mbg1cPortLocal  = "8443"
    mbg1Name        = "mbg1"
    mbg1crtFlags    = f"--rootCa ./mtls/ca.crt --certificate ./mtls/mbg1.crt --key ./mtls/mbg1.key"  if dataplane =="mtls" else ""
    mbgctl1Name     = "mbgctl1"

    

    #MBG2 parameters 
    mbg2DataPort    = "30001"
    mbg2cPort       = "30443"
    mbg2cPortLocal  = "8443"
    mbg2crtFlags    = f"--rootCa ./mtls/ca.crt --certificate ./mtls/mbg2.crt --key ./mtls/mbg2.key"  if dataplane =="mtls" else ""
    mbg2Name        = "mbg2"
    mbgctl2Name     = "mbgctl2"

    #MBG3 parameters 
    mbg3DataPort    = "30001"
    mbg3cPort       = "30443"
    mbg3cPortLocal  = "8443"
    mbg3crtFlags    = f"--rootCa ./mtls/ca.crt --certificate ./mtls/mbg3.crt --key ./mtls/mbg3.key"  if dataplane =="mtls" else ""
    mbg3Name        = "mbg3"
    mbgctl3Name     = "mbgctl3"
    

    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)
    ### clean 
    print(f"Clean old kinds")
    os.system("make clean-kind")
    
    ### build docker environment 
    printHeader(f"Build docker image")
    os.system("make docker-build")
    
    ## build Kind clusters environment 

    if cni == "diff":
        printHeader(f"Cluster 1: Flannel, Cluster 2: KindNet, Cluster 3: Calico")
        startKindClusterMbg(mbg1Name, mbgctl1Name, mbg1cPortLocal, mbg1cPort, mbg1DataPort, dataplane ,mbg1crtFlags, False,  "flannel")        
        startKindClusterMbg(mbg2Name, mbgctl2Name, mbg2cPortLocal, mbg2cPort, mbg2DataPort,dataplane ,mbg2crtFlags)        
        startKindClusterMbg(mbg3Name, mbgctl3Name, mbg3cPortLocal, mbg3cPort, mbg3DataPort,dataplane ,mbg3crtFlags, False, "calico")        
    else:
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

    # Add MBG Peer
    useKindCluster(mbg2Name)
    printHeader("Add MBG1, MBG3 peer to MBG2")
    runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl addPeer --id {mbg1Name} --ip {mbg1Ip} --cport {mbg1cPort}')
    runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl addPeer --id {mbg3Name} --ip {mbg3Ip} --cport {mbg3cPort}')
    # Send Hello
    printHeader("Send Hello commands")
    runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl hello')
    
    ###Set mbg1 services
    useKindCluster(mbg1Name)
    runcmd(f"kind load docker-image jlesage/firefox --name={mbg1Name}")
    runcmd(f"kubectl create -f {folman}/firefox.yaml")    
    printHeader(f"Add {srcSvc1} services to host cluster")
    waitPod(srcSvc1)
    _ , srcSvcIp1 =getPodNameIp(srcSvc1)
    runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl addService --id {srcSvc1} --ip {srcSvcIp1} --description {srcSvc1}')
    runcmd(f"kubectl create service nodeport {srcSvc1} --tcp=5800:5800 --node-port=30000")
    
    ### Set mbg2 service
    useKindCluster(mbg2Name)
    runcmd(f"kind load docker-image openspeedtest/latest --name={mbg2Name}")
    runcmd(f"kubectl create -f {folman}/speedtest.yaml")
    printHeader(f"Add {destSvc} (server) service to destination cluster")
    waitPod(destSvc)
    destSvcIp = f"{getPodIp(destSvc)}:3000"
    runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl addService --id {destSvc} --ip {destSvcIp} --description v2')
    
    ### Set mbgctl3
    useKindCluster(mbg3Name)
    runcmd(f"kind load docker-image jlesage/firefox --name={mbg3Name}")
    runcmd(f"kubectl create -f {folman}/firefox.yaml")
    runcmd(f"kubectl create -f {folman}/firefox2.yaml")    
    printHeader(f"Add {srcSvc1} {srcSvc2} services to host cluster")
    waitPod(srcSvc1)
    waitPod(srcSvc2)
    _ , srcSvcIp1 =getPodNameIp(srcSvc1)
    _ , srcSvcIp2 =getPodNameIp(srcSvc2)
    runcmd(f'kubectl exec -i {mbgctl3Pod} -- ./mbgctl addService --id {srcSvc1} --ip {srcSvcIp1} --description {srcSvc1}')
    runcmd(f'kubectl exec -i {mbgctl3Pod} -- ./mbgctl addService --id {srcSvc2} --ip {srcSvcIp2} --description {srcSvc2}')
    runcmd(f"kubectl create service nodeport {srcSvc1} --tcp=5800:5800 --node-port=30000")
    runcmd(f"kubectl create service nodeport {srcSvc2} --tcp=5800:5800 --node-port=30001")
    
    print(f"Services created. Run service_expose.py")
