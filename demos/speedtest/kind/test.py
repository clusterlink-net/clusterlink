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
projDir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{projDir}')

from demos.utils.common import runcmd, createFabric, printHeader, startGwctl
from demos.utils.kind import cluster
from demos.utils.k8s import getPodNameIp

############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-c','--cni', help='choose diff to use different cnis', required=False, default="same")

    args = vars(parser.parse_args())

    printHeader("\n\nStart Kind Test\n\n")
    printHeader("Start pre-setting")
    
    folman   = f"{projDir}/demos/speedtest/testdata/manifests/"
    crtFol   = f"{projDir}/demos/utils/mtls"
    testOutputFolder = f"{projDir}/bin/tests/speedtest" 
    cni       = args["cni"]

    #GW parameters 
    gw1         = cluster(name="peer1")
    gw2         = cluster(name="peer2")
    gw3         = cluster(name="peer3")
    srcSvc1     = "firefox"
    srcSvc2     = "firefox2"
    srcSvcPort  = 5800
    destSvc     = "openspeedtest"
    destSvcPort = 3000
    
    print(f'Working directory {projDir}')
    os.chdir(projDir)
    
    ### build environment 
    printHeader("Build docker image")
    os.system("make docker-build")
    os.system("sudo make install")
    if cni == "diff":
        printHeader("Cluster 1: Flannel, Cluster 2: KindNet, Cluster 3: Calico")
        gw1.cni="flannel"
        gw3.cni="calico"
    
    # Create Kind clusters environment 
    gw1.createCluster(runBg=True)        
    gw2.createCluster(runBg=True)
    gw3.createCluster(runBg=False)  

    # Start Kind clusters environment 
    createFabric(testOutputFolder) 
    gw1.startCluster(testOutputFolder)        
    gw2.startCluster(testOutputFolder)        
    gw3.startCluster(testOutputFolder)        
     
    # Start gwctl
    startGwctl(gw1.name, gw1.ip, gw1.port, testOutputFolder)
    startGwctl(gw2.name, gw2.ip, gw2.port, testOutputFolder)
    startGwctl(gw3.name, gw3.ip, gw3.port, testOutputFolder)

    # Load services 
    gw1.useCluster()
    gw1.loadService(srcSvc1, "jlesage/firefox",f"{folman}/firefox.yaml" )
    runcmd(f"kubectl create service nodeport {srcSvc1} --tcp={srcSvcPort}:{srcSvcPort} --node-port=30000")
    gw2.useCluster()
    gw2.loadService(destSvc, " openspeedtest/latest",f"{folman}/speedtest.yaml")
    gw3.useCluster()
    gw3.loadService(srcSvc1, "jlesage/firefox",f"{folman}/firefox.yaml" )
    gw3.loadService(srcSvc2, "jlesage/firefox",f"{folman}/firefox2.yaml" )
    runcmd(f"kubectl create service nodeport {srcSvc1} --tcp={srcSvcPort}:{srcSvcPort} --node-port=30000")
    runcmd(f"kubectl create service nodeport {srcSvc2} --tcp={srcSvcPort}:{srcSvcPort} --node-port=30001")
    
    # Add gw Peer
    printHeader("Add gw2 peer to gw1")
    runcmd(f'gwctl create peer --myid {gw1.name} --name {gw2.name} --host {gw2.ip} --port {gw2.port}')
    printHeader("Add gw1, gw3 peer to gw2")
    runcmd(f'gwctl create peer --myid {gw2.name} --name {gw1.name} --host {gw1.ip} --port {gw1.port}')
    runcmd(f'gwctl create peer --myid {gw2.name} --name {gw3.name} --host {gw3.ip} --port {gw3.port}')
    printHeader("Add gw2 peer to gw3")
    runcmd(f'gwctl create peer --myid {gw3.name} --name {gw2.name} --host {gw2.ip} --port {gw2.port}')
    
    # Set exports
    runcmd(f'gwctl create export --myid {gw1.name} --name {srcSvc1} --host {srcSvc1} --port {srcSvcPort}')    
    runcmd(f'gwctl create export --myid {gw2.name} --name {destSvc} --host {destSvc} --port {destSvcPort}')    
    runcmd(f'gwctl create export --myid {gw3.name} --name {srcSvc1}  --host {srcSvc1} --port {srcSvcPort}')
    runcmd(f'gwctl create export --myid {gw3.name} --name {srcSvc2}  --host {srcSvc2} --port {srcSvcPort}')
    print("Services created. Run service_import.py")
