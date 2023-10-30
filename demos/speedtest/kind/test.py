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

import os
import sys
import argparse
proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

from demos.utils.common import runcmd, createFabric, printHeader, startGwctl
from demos.utils.kind import startKindCluster,useKindCluster, getKindIp,loadService
from demos.utils.k8s import getPodNameIp

############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-c','--cni', help='choose diff to use different cnis', required=False, default="same")

    args = vars(parser.parse_args())

    printHeader("\n\nStart Kind Test\n\n")
    printHeader("Start pre-setting")
    
    folman   = f"{proj_dir}/demos/speedtest/manifests/"
    crtFol   = f"{proj_dir}/demos/utils/mtls"
    testOutputFolder = f"{proj_dir}/bin/tests/speedtest" 
    cni       = args["cni"]

    #GW parameters 
    gwPort      = "30443"    
    gw1Name     = "peer1"
    gw2Name     = "peer2"
    gw3Name     = "peer3"
    srcSvc1     = "firefox"
    srcSvc2     = "firefox2"
    srcSvcPort  = 5800
    destSvc     = "openspeedtest"
    destSvcPort = 3000
    
    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)
    
    ### build environment 
    printHeader("Build docker image")
    os.system("make docker-build")
    os.system("sudo make install")

    ## build Kind clusters environment 
    createFabric(testOutputFolder) 
    if cni == "diff":
        printHeader("Cluster 1: Flannel, Cluster 2: KindNet, Cluster 3: Calico")
        startKindCluster(gw1Name, testOutputFolder,cni="flannel")        
        startKindCluster(gw2Name, testOutputFolder)
        startKindCluster(gw3Name, testOutputFolder,cni="calico") 
    else:
        startKindCluster(gw1Name, testOutputFolder)        
        startKindCluster(gw2Name, testOutputFolder)
        startKindCluster(gw3Name, testOutputFolder) 

    ###get gw parameters
    gw1Ip                = getKindIp(gw1Name)
    gwctl1Pod, gwctl1Ip = getPodNameIp("gwctl")
    gw2Ip                = getKindIp(gw2Name)
    gwctl2Pod, gwctl2Ip = getPodNameIp("gwctl")
    gw3Ip                = getKindIp(gw3Name)
    gwctl3Pod, gwctl3Ip = getPodNameIp("gwctl")

    # Start gwctl
    startGwctl(gw1Name, gw1Ip, gwPort, testOutputFolder)
    startGwctl(gw2Name, gw2Ip, gwPort, testOutputFolder)
    startGwctl(gw3Name, gw3Ip, gwPort, testOutputFolder)

    # Add gw Peer
    useKindCluster(gw1Name)
    printHeader("Add gw2 peer to gw1")
    runcmd(f'gwctl create peer --myid {gw1Name} --name {gw2Name} --host {gw2Ip} --port {gwPort}')
    useKindCluster(gw2Name)
    printHeader("Add gw1, gw3 peer to gw2")
    runcmd(f'gwctl create peer --myid {gw2Name} --name {gw1Name} --host {gw1Ip} --port {gwPort}')
    runcmd(f'gwctl create peer --myid {gw2Name} --name {gw3Name} --host {gw3Ip} --port {gwPort}')
    useKindCluster(gw3Name)
    printHeader("Add gw2 peer to gw3")
    runcmd(f'gwctl create peer --myid {gw3Name} --name {gw2Name} --host {gw2Ip} --port {gwPort}')
    
    ###Set gw1 services
    useKindCluster(gw1Name)
    loadService(srcSvc1,gw1Name, "jlesage/firefox",f"{folman}/firefox.yaml" )
    runcmd(f'gwctl create export --myid {gw1Name} --name {srcSvc1} --host {srcSvc1} --port {srcSvcPort}')
    runcmd(f"kubectl create service nodeport {srcSvc1} --tcp={srcSvcPort}:{srcSvcPort} --node-port=30000")
    
    ### Set gw2 service
    useKindCluster(gw2Name)
    loadService(destSvc,gw2Name, " openspeedtest/latest",f"{folman}/speedtest.yaml" )
    runcmd(f'gwctl create export --myid {gw2Name} --name {destSvc} --host {destSvc} --port {destSvcPort}')
    
    ### Set gwctl3
    useKindCluster(gw3Name)
    loadService(srcSvc1,gw3Name, "jlesage/firefox",f"{folman}/firefox.yaml" )
    loadService(srcSvc2,gw3Name, "jlesage/firefox",f"{folman}/firefox2.yaml" )
    runcmd(f'gwctl create export --myid {gw3Name} --name {srcSvc1}  --host {srcSvc1} --port {srcSvcPort}')
    runcmd(f'gwctl create export --myid {gw3Name} --name {srcSvc2}  --host {srcSvc2} --port {srcSvcPort}')
    runcmd(f"kubectl create service nodeport {srcSvc1} --tcp={srcSvcPort}:{srcSvcPort} --node-port=30000")
    runcmd(f"kubectl create service nodeport {srcSvc2} --tcp={srcSvcPort}:{srcSvcPort} --node-port=30001")
    print("Services created. Run service_import.py")
