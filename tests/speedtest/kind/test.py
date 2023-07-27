#!/usr/bin/env python3
##############################################################################################
# Name: Bookinfo
# Info: support bookinfo application with gwctl inside the clusters 
#       In this we create three kind clusters
#       1) MBG1- contain mbg, gwctl,product and details microservices (bookinfo services)
#       2) MBG2- contain mbg, gwctl, review-v2 and rating microservices (bookinfo services)
#       3) MBG3- contain mbg, gwctl, review-v3 and rating microservices (bookinfo services)
##############################################################################################

import os,time
import subprocess as sp
import sys
import argparse
proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

from tests.utils.mbgAux import runcmd, runcmdb, printHeader, getPodName, waitPod,getMbgPorts,buildMbg,buildMbgctl,getPodIp,getPodNameIp
from tests.utils.kind.kindAux import startKindClusterMbg, startMbgctl, useKindCluster,getKindIp

############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-d','--dataplane', help='choose which dataplane to use mtls/tcp', required=False, default="mtls")
    parser.add_argument('-c','--cni', help='choose diff to use different cnis', required=False, default="same")

    args = vars(parser.parse_args())

    printHeader("\n\nStart Kind Test\n\n")
    printHeader("Start pre-setting")
    
    folman   = f"{proj_dir}/tests/speedtest/manifests/"
    crtFol   = f"{proj_dir}/tests/utils/mtls"
    dataplane = args["dataplane"]
    cni       = args["cni"]


    srcSvc1         = "firefox"
    srcSvc2         = "firefox2"
    destSvc         = "openspeedtest"
    
    #MBG1 parameters 
    mbg1DataPort    = "30001"
    mbg1cPort       = "30443"
    mbg1cPortLocal  = "443"
    mbg1Name        = "mbg1"
    mbg1crtFlags    = f"--certca ./mtls/ca.crt --cert ./mtls/mbg1.crt --key ./mtls/mbg1.key"  if dataplane =="mtls" else ""
    gwctl1crt    = f"--certca {crtFol}/ca.crt --cert {crtFol}/mbg1.crt --key {crtFol}/mbg1.key"  if dataplane =="mtls" else ""
    gwctl1Name     = "gwctl1"

    

    #MBG2 parameters 
    mbg2DataPort    = "30001"
    mbg2cPort       = "30443"
    mbg2cPortLocal  = "443"
    mbg2crtFlags    = f"--certca ./mtls/ca.crt --cert ./mtls/mbg2.crt --key ./mtls/mbg2.key"  if dataplane =="mtls" else ""
    gwctl2crt    = f"--certca {crtFol}/ca.crt --cert {crtFol}/mbg2.crt --key {crtFol}/mbg2.key"  if dataplane =="mtls" else ""
    mbg2Name        = "mbg2"
    gwctl2Name     = "gwctl2"

    #MBG3 parameters 
    mbg3DataPort    = "30001"
    mbg3cPort       = "30443"
    mbg3cPortLocal  = "443"
    mbg3crtFlags    = f"--certca ./mtls/ca.crt --cert ./mtls/mbg3.crt --key ./mtls/mbg3.key"  if dataplane =="mtls" else ""
    gwctl3crt    = f"--certca {crtFol}/ca.crt --cert {crtFol}/mbg3.crt --key {crtFol}/mbg3.key"  if dataplane =="mtls" else ""
    mbg3Name        = "mbg3"
    gwctl3Name     = "gwctl3"
    

    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)
    ### clean 
    print(f"Clean old kinds")
    os.system("make clean-kind")
    
    ### Build mbg/gwctl

    os.system("make build")
    os.system("sudo make install")

    ### build docker environment 
    printHeader(f"Build docker image")
    os.system("make docker-build")
    
    ## build Kind clusters environment 

    if cni == "diff":
        printHeader(f"Cluster 1: Flannel, Cluster 2: KindNet, Cluster 3: Calico")
        startKindClusterMbg(mbg1Name, gwctl1Name, mbg1cPortLocal, mbg1cPort, mbg1DataPort, dataplane ,mbg1crtFlags,False, False,  "flannel")
        startKindClusterMbg(mbg2Name, gwctl2Name, mbg2cPortLocal, mbg2cPort, mbg2DataPort, dataplane ,mbg2crtFlags, False)
        startKindClusterMbg(mbg3Name, gwctl3Name, mbg3cPortLocal, mbg3cPort, mbg3DataPort, dataplane ,mbg3crtFlags, False, False, "calico")
    else:
        startKindClusterMbg(mbg1Name, gwctl1Name, mbg1cPortLocal, mbg1cPort, mbg1DataPort, dataplane ,mbg1crtFlags, False)
        startKindClusterMbg(mbg2Name, gwctl2Name, mbg2cPortLocal, mbg2cPort, mbg2DataPort, dataplane ,mbg2crtFlags, False)
        startKindClusterMbg(mbg3Name, gwctl3Name, mbg3cPortLocal, mbg3cPort, mbg3DataPort, dataplane ,mbg3crtFlags, False)
    ###get mbg parameters
    useKindCluster(mbg1Name)
    mbg1Pod, _            = getPodNameIp("mbg")
    mbg1Ip                = getKindIp("mbg1")
    gwctl1Pod, gwctl1Ip = getPodNameIp("gwctl")

    useKindCluster(mbg2Name)
    mbg2Pod, _            = getPodNameIp("mbg")
    gwctl2Pod, gwctl2Ip = getPodNameIp("gwctl")
    mbg2Ip                = getKindIp(mbg2Name)

    useKindCluster(mbg3Name)
    mbg3Pod, _            = getPodNameIp("mbg")
    mbg3Ip                = getKindIp("mbg3")
    gwctl3Pod, gwctl3Ip = getPodNameIp("gwctl")


    # Start gwctl
    startMbgctl(gwctl1Name, mbg1Ip, mbg1cPort, dataplane, gwctl1crt)
    startMbgctl(gwctl2Name, mbg2Ip, mbg2cPort, dataplane, gwctl2crt)
    startMbgctl(gwctl3Name, mbg3Ip, mbg3cPort, dataplane, gwctl3crt)


    # Add MBG Peer
    useKindCluster(mbg1Name)
    printHeader("Add MBG2 peer to MBG1")
    runcmd(f'gwctl create peer --myid {gwctl1Name} --name {mbg2Name} --host {mbg2Ip} --port {mbg2cPort}')
    useKindCluster(mbg2Name)
    printHeader("Add MBG1, MBG3 peer to MBG2")
    runcmd(f'gwctl create peer --myid {gwctl2Name} --name {mbg1Name} --host {mbg1Ip} --port {mbg1cPort}')
    runcmd(f'gwctl create peer --myid {gwctl2Name} --name {mbg3Name} --host {mbg3Ip} --port {mbg3cPort}')
    useKindCluster(mbg3Name)
    printHeader("Add MBG2 peer to MBG3")
    runcmd(f'gwctl create peer --myid {gwctl3Name} --name {mbg2Name} --host {mbg2Ip} --port {mbg2cPort}')
    
    ###Set mbg1 services
    useKindCluster(mbg1Name)
    runcmd(f"kind load docker-image jlesage/firefox --name={mbg1Name}")
    runcmd(f"kubectl create -f {folman}/firefox.yaml")    
    printHeader(f"Add {srcSvc1} services to host cluster")
    waitPod(srcSvc1)
    runcmd(f'gwctl create export --myid {gwctl1Name} --name {srcSvc1} --host {srcSvc1} --port {5800}')
    runcmd(f"kubectl create service nodeport {srcSvc1} --tcp=5800:5800 --node-port=30000")
    
    ### Set mbg2 service
    useKindCluster(mbg2Name)
    runcmd(f"kind load docker-image openspeedtest/latest --name={mbg2Name}")
    runcmd(f"kubectl create -f {folman}/speedtest.yaml")
    printHeader(f"Add {destSvc} (server) service to destination cluster")
    waitPod(destSvc)
    destSvcPort = "3000"
    _ , destSvcIp =getPodNameIp(destSvc)
    runcmd(f'gwctl create export --myid {gwctl2Name} --name {destSvc} --host {destSvcIp} --port {destSvcPort}')
    
    ### Set gwctl3
    useKindCluster(mbg3Name)
    runcmd(f"kind load docker-image jlesage/firefox --name={mbg3Name}")
    runcmd(f"kubectl create -f {folman}/firefox.yaml")
    runcmd(f"kubectl create -f {folman}/firefox2.yaml")    
    printHeader(f"Add {srcSvc1} {srcSvc2} services to host cluster")
    waitPod(srcSvc1)
    waitPod(srcSvc2)
    runcmd(f'gwctl create export --myid {gwctl3Name} --name {srcSvc1}  --host {srcSvc1} --port {5800}')
    runcmd(f'gwctl create export --myid {gwctl3Name} --name {srcSvc2}  --host {srcSvc2} --port {5800}')
    runcmd(f"kubectl create service nodeport {srcSvc1} --tcp=5800:5800 --node-port=30000")
    runcmd(f"kubectl create service nodeport {srcSvc2} --tcp=5800:5800 --node-port=30001")
    
    print(f"Services created. Run service_import.py")
