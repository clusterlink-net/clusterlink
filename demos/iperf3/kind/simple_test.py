#!/usr/bin/env python3
# Copyright 2023 The ClusterLink Authors.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

################################################################
#Name: Simple iperf3  test
#Desc: create 2 kind clusters :
# 1) MBG and iperf3 client
# 2) MBG and iperf3 server    
###############################################################
import os,time
import subprocess as sp
import sys
import argparse


proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

from demos.utils.mbgAux import runcmd, runcmdb, printHeader, waitPod, getPodName, getMbgPorts,buildMbg,buildMbgctl,getPodIp,getPodNameIp
from demos.iperf3.kind.connect_mbgs import connectMbgs
from demos.iperf3.kind.iperf3_service_create import setIperf3client, setIperf3Server
from demos.iperf3.kind.iperf3_service_import import importService
from demos.iperf3.kind.iperf3_service_get import getService
from demos.iperf3.kind.iperf3_client_start import directTestIperf3,testIperf3Client
from demos.iperf3.kind.apply_policy import applyAccessPolicy

from demos.utils.kind.kindAux import useKindCluster, getKindIp,startKindClusterMbg,startGwctl

############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-d','--dataplane', help='choose which dataplane to use mtls/tcp', required=False, default="mtls")
    parser.add_argument('-c','--cni', help='Which cni to use default(kindnet)/flannel/calico/diff (different cni for each cluster)', required=False, default="default")

    args = vars(parser.parse_args())

    printHeader("\n\nStart Kind Test\n\n")
    printHeader("Start pre-setting")

    dataplane = args["dataplane"]
    cni = args["cni"]
    crtFol   = f"{proj_dir}/demos/utils/mtls"
    #MBG1 parameters 
    mbg1DataPort    = "30001"
    mbg1cPort       = "30443"
    mbg1cPortLocal  = 443
    mbg1crtFlags    = f"--certca ./mtls/ca.crt --cert ./mtls/mbg1.crt --key ./mtls/mbg1.key"  if dataplane =="mtls" else ""
    gwctl1crt    = f"--certca {crtFol}/ca.crt --cert {crtFol}/mbg1.crt --key {crtFol}/mbg1.key"  if dataplane =="mtls" else ""
    mbg1Name        = "mbg1"
    gwctl1Name     = "gwctl1"
    mbg1cni         = cni 
    srcSvc          = "iperf3-client"

    
    #MBG2 parameters 
    mbg2DataPort    = "30001"
    mbg2cPort       = "30443"
    mbg2cPortLocal  = 443
    mbg2crtFlags    = f"--certca ./mtls/ca.crt --cert ./mtls/mbg2.crt --key ./mtls/mbg2.key"  if dataplane =="mtls" else ""
    gwctl2crt    = f"--certca {crtFol}/ca.crt --cert {crtFol}/mbg2.crt --key {crtFol}/mbg2.key"  if dataplane =="mtls" else ""
    mbg2Name        = "mbg2"
    gwctl2Name     = "gwctl2"
    mbg2cni         = "flannel" if cni == "diff" else cni
    destSvc         = "iperf3-server"
    destPort        = 5000
    kindDestPort    = "30001"
    
        
    #folders
    folCl=f"{proj_dir}/demos/iperf3/testdata/manifests/iperf3-client"
    folSv=f"{proj_dir}/demos/iperf3/testdata/manifests/iperf3-server"

    #files
    allowAllPolicy=f"{proj_dir}/pkg/policyengine/policytypes/examples/allowAll.json"
    
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
      
    ###get mbg parameters
    useKindCluster(mbg1Name)
    mbg1Ip               = getKindIp(mbg1Name)
    useKindCluster(mbg2Name)
    mbg2Ip               = getKindIp(mbg2Name)
   
    # Start gwctl
    startGwctl(gwctl1Name, mbg1Ip, mbg1cPort, dataplane, gwctl1crt)
    startGwctl(gwctl2Name, mbg2Ip, mbg2cPort, dataplane, gwctl2crt)
    
    # Add MBG Peer
    printHeader("Add MBG1 peer to MBG2")
    connectMbgs(gwctl1Name, mbg2Name, mbg2Ip, mbg2cPort)
    printHeader("Add MBG2 peer to MBG1")
    connectMbgs(gwctl2Name, mbg1Name, mbg1Ip, mbg1cPort)
    
    # Set service iperf3-client in MBG1
    setIperf3client(mbg1Name, gwctl1Name, srcSvc)
    
    # Set service iperf3-server in MBG2
    setIperf3Server(mbg2Name, gwctl2Name,destSvc)

    #Import destination service
    importService(mbg1Name, gwctl1Name, destSvc,destPort, mbg2Name)

    #Get services
    getService(gwctl1Name, destSvc)
    #Add policy
    applyAccessPolicy(mbg1Name, gwctl1Name, allowAllPolicy)
    applyAccessPolicy(mbg2Name, gwctl2Name, allowAllPolicy)
    #Testing
    printHeader("\n\nStart Iperf3 testing")
    useKindCluster(mbg2Name)
    waitPod("iperf3-server")
    # Test MBG1
    directTestIperf3(mbg1Name, srcSvc, mbg2Ip, kindDestPort)
    testIperf3Client(mbg1Name, srcSvc, destSvc,    destPort)


