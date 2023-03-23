#!/usr/bin/env python3
import os,time
import subprocess as sp
import sys
import argparse


proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

from tests.utils.mbgAux import runcmd, runcmdb, printHeader, waitPod, getPodName, getMbgPorts,buildMbg,buildMbgctl,getPodIp,getPodNameIp
from tests.iperf3.kind.connect_mbgs import connectMbgs, sendHello
from tests.iperf3.kind.iperf3_service_create import setIperf3client, setIperf3Server
from tests.iperf3.kind.iperf3_service_expose import exposeService
from tests.iperf3.kind.iperf3_service_get import getService
from tests.iperf3.kind.iperf3_client_start import directTestIperf3,testIperf3Client
from tests.iperf3.kind.apply_policy import applyPolicy

from tests.utils.kind.kindAux import useKindCluster, getKindIp,startKindClusterMbg

############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-d','--dataplane', help='choose which dataplane to use mtls/tcp', required=False, default="tcp")
    parser.add_argument('-c','--cni', help='Which cni to use default(kindnet)/flannel/calico/diff (different cni for each cluster)', required=False, default="default")

    args = vars(parser.parse_args())

    printHeader("\n\nStart Kind Test\n\n")
    printHeader("Start pre-setting")

    dataplane = args["dataplane"]
    cni = args["cni"]
    #MBG1 parameters 
    mbg1DataPort    = "30001"
    mbg1cPort       = "30443"
    mbg1cPortLocal  = "8443"
    mbg1crtFlags    = f"--rootCa ./mtls/ca.crt --certificate ./mtls/mbg1.crt --key ./mtls/mbg1.key"  if dataplane =="mtls" else ""
    mbg1Name        = "mbg1"
    mbgctl1Name     = "mbgctl1"
    mbg1cni         = cni 
    srcSvc          = "iperf3-client"
    srck8sSvcPort   = "5000"
    
    #MBG2 parameters 
    mbg2DataPort    = "30001"
    mbg2cPort       = "30443"
    mbg2cPortLocal  = "8443"
    mbg2crtFlags    = f"--rootCa ./mtls/ca.crt --certificate ./mtls/mbg2.crt --key ./mtls/mbg2.key"  if dataplane =="mtls" else ""
    mbg2Name        = "mbg2"
    mbgctl2Name     = "mbgctl2"
    mbg2cni         = "flannel" if cni == "diff" else cni
    destSvc         = "iperf3-server"
    iperf3DestPort  = "30001"
    

    
    #MBG3 parameters 
    mbg3DataPort    = "30001"
    mbg3cPort       = "30443"
    mbg3cPortLocal  = "8443"
    mbg3crtFlags    = f"--rootCa ./mtls/ca.crt --certificate ./mtls/mbg3.crt --key ./mtls/mbg3.key"  if dataplane =="mtls" else ""
    mbg3Name        = "mbg3"
    mbgctl3Name     = "mbgctl3"
    mbg3cni         = "calico" if cni == "diff" else cni
    srcSvc          = "iperf3-client"
    srck8sSvcPort   = "5000"
    srcSvc2         = "iperf3-client2"

        
    #folders
    folCl=f"{proj_dir}/tests/iperf3/manifests/iperf3-client"
    folSv=f"{proj_dir}/tests/iperf3/manifests/iperf3-server"
    
    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)
    
    ### clean 
    print(f"Clean old kinds")
    os.system("make clean-kind-iperf3")
    
    ### build docker environment 
    printHeader(f"Build docker image")
    os.system("make docker-build")
    
    
    ### Build MBG in Kind clusters environment 
    startKindClusterMbg(mbg1Name, mbgctl1Name, mbg1cPortLocal, mbg1cPort, mbg1DataPort, dataplane ,mbg1crtFlags, cni=mbg1cni)        
    startKindClusterMbg(mbg2Name, mbgctl2Name, mbg2cPortLocal, mbg2cPort, mbg2DataPort, dataplane ,mbg2crtFlags, cni=mbg2cni)        
    startKindClusterMbg(mbg3Name, mbgctl3Name, mbg3cPortLocal, mbg3cPort, mbg3DataPort, dataplane ,mbg3crtFlags, cni=mbg3cni)        
      
    ###get mbg parameters
    useKindCluster(mbg1Name)
    mbg1Pod, _           = getPodNameIp("mbg")
    mbg1Ip               = getKindIp("mbg1")
    mbgctl1Pod, mbgctl1Ip= getPodNameIp("mbgctl")
    useKindCluster(mbg2Name)
    mbg2Pod, mbg2Ip       = getPodNameIp("mbg")
    mbgctl2Pod, mbgctl2Ip = getPodNameIp("mbgctl")
    destkindIp=getKindIp(mbg2Name)
    useKindCluster(mbg3Name)
    mbg3Pod, _            = getPodNameIp("mbg")
    mbg3Ip                = getKindIp("mbg3")
    mbgctl3Pod, mbgctl3Ip = getPodNameIp("mbgctl")

    
    # Add MBG Peer
    useKindCluster(mbg2Name)
    printHeader("Add MBG2, MBG3 peer to MBG1")
    connectMbgs(mbg2Name, mbgctl2Name, mbgctl2Pod, mbg1Name, mbg1Ip, mbg1cPort)
    connectMbgs(mbg2Name, mbgctl2Name, mbgctl2Pod, mbg3Name, mbg3Ip, mbg3cPort)

    # Send Hello
    sendHello(mbgctl2Pod, mbgctl2Name)        

    
    # Set service iperf3-client in MBG1
    setIperf3client(mbg1Name, mbgctl1Name, srcSvc)
    
    # Set service iperf3-server in MBG2
    setIperf3Server(mbg2Name, mbgctl2Name, destSvc)

    # Set service iperf3-client in MBG3
    setIperf3client(mbg3Name, mbgctl3Name, srcSvc)
    setIperf3client(mbg3Name, mbgctl3Name, srcSvc2)
    
    #Expose destination service
    exposeService(mbg2Name, mbgctl2Name, destSvc)

    #Get services
    getService(mbg1Name, mbgctl1Name)
    
    #Testing
    printHeader("\n\nStart Iperf3 testing")
    useKindCluster(mbg2Name)
    waitPod("iperf3-server")
    
    #Test MBG1
    (mbg1Name, srcSvc,destkindIp,iperf3DestPort)
    testIperf3Client(mbg1Name,srcSvc,destSvc)

    #Test MBG3
    directTestIperf3(mbg3Name, srcSvc, destkindIp, iperf3DestPort)
    testIperf3Client(mbg3Name,srcSvc,destSvc)


    #Block Traffic in MBG3
    printHeader("Start Block Traffic in MBG3")
    applyPolicy(mbg3Name, mbgctl3Name, type="deny")
    testIperf3Client(mbg3Name,srcSvc,destSvc, blockFlag=True)
    print("Allow Traffic in MBG3")
    applyPolicy(mbg3Name, mbgctl3Name, type="allow")
    testIperf3Client(mbg3Name,srcSvc,destSvc)
    
    #Block Traffic in MBG2
    printHeader("Start Block Traffic in MBG2")
    print("Block Traffic in MBG2")
    applyPolicy(mbg2Name, mbgctl2Name, type="deny")
    testIperf3Client(mbg3Name,srcSvc,destSvc, blockFlag=True)
    print("Allow Traffic in MBG3")
    applyPolicy(mbg2Name, mbgctl2Name, type="allow")
    testIperf3Client(mbg3Name,srcSvc,destSvc)
    
