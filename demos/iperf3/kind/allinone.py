#!/usr/bin/env python3
import os,time
import subprocess as sp
import sys
import argparse


proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

from demos.utils.mbgAux import runcmd, runcmdb, printHeader, waitPod, getPodNameIp
from demos.iperf3.kind.connect_mbgs import connectMbgs
from demos.iperf3.kind.iperf3_service_create import setIperf3client, setIperf3Server
from demos.iperf3.kind.iperf3_service_import import importService
from demos.iperf3.kind.iperf3_service_get import getService
from demos.iperf3.kind.iperf3_client_start import directTestIperf3,testIperf3Client
from demos.iperf3.kind.apply_policy import applyPolicy
from demos.iperf3.kind.iperf3_external_service import exportExternalService

from demos.utils.kind.kindAux import useKindCluster, startGwctl, getKindIp,startKindClusterMbg

############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-d','--dataplane', help='choose which dataplane to use mtls/tcp', required=False, default="mtls")
    parser.add_argument('-c','--cni', help='Which cni to use default(kindnet)/flannel/calico/diff (different cni for each cluster)', required=False, default="default")

    args = vars(parser.parse_args())
    allowAllPolicy =f"{proj_dir}/pkg/policyengine/policytypes/examples/allowAll.json"

    printHeader("\n\nStart Kind Test\n\n")
    printHeader("Start pre-setting")

    crtFol   = f"{proj_dir}/demos/utils/mtls"
    dataplane = args["dataplane"]
    cni = args["cni"]
    #MBG1 parameters 
    mbg1DataPort    = "30001"
    mbg1cPort       = "30443"
    mbg1cPortLocal  = "443"
    mbg1crtFlags    = f"--certca ./mtls/ca.crt --cert ./mtls/mbg1.crt --key ./mtls/mbg1.key"  if dataplane =="mtls" else ""
    gwctl1crt    = f"--certca {crtFol}/ca.crt --cert {crtFol}/mbg1.crt --key {crtFol}/mbg1.key"  if dataplane =="mtls" else ""
    mbg1Name        = "mbg1"
    gwctl1Name     = "gwctl1"
    mbg1cni         = cni 
    srcSvc          = "iperf3-client"
    
    #MBG2 parameters 
    mbg2DataPort    = "30001"
    mbg2cPort       = "30443"
    mbg2cPortLocal  = "443"
    mbg2crtFlags    = f"--certca ./mtls/ca.crt --cert ./mtls/mbg2.crt --key ./mtls/mbg2.key"  if dataplane =="mtls" else ""
    gwctl2crt    = f"--certca {crtFol}/ca.crt --cert {crtFol}/mbg2.crt --key {crtFol}/mbg2.key"  if dataplane =="mtls" else ""
    mbg2Name        = "mbg2"
    gwctl2Name     = "gwctl2"
    mbg2cni         = "flannel" if cni == "diff" else cni
    destSvc         = "iperf3-server"
    destPort        = "5000"
    kindDestPort    = "30001"
    

    
    #MBG3 parameters 
    mbg3DataPort    = "30001"
    mbg3cPort       = "30443"
    mbg3cPortLocal  = "443"
    mbg3crtFlags    = f"--certca ./mtls/ca.crt --cert ./mtls/mbg3.crt --key ./mtls/mbg3.key"  if dataplane =="mtls" else ""
    gwctl3crt    = f"--certca {crtFol}/ca.crt --cert {crtFol}/mbg3.crt --key {crtFol}/mbg3.key"  if dataplane =="mtls" else ""
    mbg3Name        = "mbg3"
    gwctl3Name     = "gwctl3"
    mbg3cni         = "calico" if cni == "diff" else cni
    srcSvc          = "iperf3-client"
    srcSvc2         = "iperf3-client2"

        
    #folders
    folCl=f"{proj_dir}/demos/iperf3/testdata/manifests/iperf3-client"
    folSv=f"{proj_dir}/demos/iperf3/testdata/manifests/iperf3-server"
    
    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)
    
    ### clean 
    print(f"Clean old kinds")
    os.system("make clean-kind-iperf3")
    
    ### build docker environment 
    os.system("make build")
    os.system("sudo make install")

    printHeader(f"Build docker image")
    os.system("make docker-build")
    
    
    ### Build MBG in Kind clusters environment 
    startKindClusterMbg(mbg1Name, gwctl1Name, mbg1cPortLocal, mbg1cPort, mbg1DataPort, dataplane ,mbg1crtFlags, cni=mbg1cni)        
    startKindClusterMbg(mbg2Name, gwctl2Name, mbg2cPortLocal, mbg2cPort, mbg2DataPort, dataplane ,mbg2crtFlags, cni=mbg2cni)        
    startKindClusterMbg(mbg3Name, gwctl3Name, mbg3cPortLocal, mbg3cPort, mbg3DataPort, dataplane ,mbg3crtFlags, cni=mbg3cni)        
      
    ###get mbg parameters
    useKindCluster(mbg1Name)
    mbg1Ip        = getKindIp("mbg1")
    useKindCluster(mbg2Name)
    mbg2Ip        = getKindIp(mbg2Name)
    useKindCluster(mbg3Name)
    mbg3Ip        = getKindIp("mbg3")

    # Start gwctl
    startGwctl(gwctl1Name, mbg1Ip, mbg1cPort, dataplane, gwctl1crt)
    startGwctl(gwctl2Name, mbg2Ip, mbg2cPort, dataplane, gwctl2crt)
    startGwctl(gwctl3Name, mbg3Ip, mbg3cPort, dataplane, gwctl3crt)

    # Add MBG Peer
    useKindCluster(mbg1Name)
    printHeader("Add MBG2 peer to MBG1")
    connectMbgs(gwctl1Name, mbg2Name, mbg2Ip, mbg2cPort)
    printHeader("Add MBG1, MBG3 peer to MBG2")
    connectMbgs(gwctl2Name, mbg1Name, mbg1Ip, mbg1cPort)
    connectMbgs(gwctl2Name, mbg3Name, mbg3Ip, mbg3cPort)
    printHeader("Add MBG2 peer to MBG3")
    connectMbgs(gwctl3Name, mbg2Name,mbg2Ip , mbg2cPort)
    
    # Set service iperf3-client in MBG1
    setIperf3client(mbg1Name, gwctl1Name, srcSvc)
    
    # Set service iperf3-server in MBG2
    setIperf3Server(mbg2Name, gwctl2Name, destSvc)

    # Set service iperf3-client in MBG3
    setIperf3client(mbg3Name, gwctl3Name, srcSvc)
    setIperf3client(mbg3Name, gwctl3Name, srcSvc2)
    
    #Import and bind a service
    importService(mbg1Name, gwctl1Name, destSvc,destPort, mbg2Name)
    importService(mbg3Name, gwctl3Name, destSvc,destPort, mbg2Name)

    # Set policies
    printHeader(f"\n\nApplying policy file {allowAllPolicy}")
    useKindCluster(mbg1Name)
    runcmd(f'gwctl --myid {gwctl1Name} create policy --type access --policyFile {allowAllPolicy}')
    runcmd(f'gwctl --myid {gwctl2Name} create policy --type access --policyFile {allowAllPolicy}')
    runcmd(f'gwctl --myid {gwctl3Name} create policy --type access --policyFile {allowAllPolicy}')
    #Get services
    getService(gwctl1Name,destSvc)
    
    #Testing
    printHeader("\n\nStart Iperf3 testing")
    useKindCluster(mbg2Name)
    waitPod("iperf3-server")
    
    #Test MBG1
    directTestIperf3(mbg1Name, srcSvc, mbg2Ip, kindDestPort)
    testIperf3Client(mbg1Name, srcSvc, destSvc, destPort)

    #Test MBG3
    directTestIperf3(mbg3Name, srcSvc, mbg2Ip, kindDestPort)
    testIperf3Client(mbg3Name, srcSvc, destSvc,    destPort)

    # Test external
    printHeader("\n\nTest external service")
    useKindCluster(mbg2Name)
    _ , destSvcPodIp = getPodNameIp(destSvc)
    externalName ="iperf3-external" 
    exportExternalService(gwctl2Name, externalName, externalName, destPort, destSvcPodIp, destPort)
    importService(mbg1Name, gwctl1Name, externalName, destPort, mbg2Name)
    testIperf3Client(mbg1Name, srcSvc, externalName, destPort)

    #Block Traffic in MBG3
    printHeader("Start Block Traffic in MBG3")
    applyPolicy(mbg3Name, gwctl3Name, type="deny")
    testIperf3Client(mbg3Name,srcSvc,destSvc, destPort, blockFlag=True)
    print("Allow Traffic in MBG3")
    applyPolicy(mbg3Name, gwctl3Name, type="allow")
    testIperf3Client(mbg3Name,srcSvc,destSvc, destPort)
    
    #Block Traffic in MBG2
    printHeader("Start Block Traffic in MBG2")
    print("Block Traffic in MBG2")
    applyPolicy(mbg2Name, gwctl2Name, type="deny")
    testIperf3Client(mbg3Name,srcSvc,destSvc, destPort, blockFlag=True)
    print("Allow Traffic in MBG3")
    applyPolicy(mbg2Name, gwctl2Name, type="allow")
    testIperf3Client(mbg3Name,srcSvc,destSvc, destPort)
    
