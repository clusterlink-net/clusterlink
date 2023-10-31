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
# Name: Simple iperf3  test
# Desc: create 2 kind clusters :
# 1) GW and iperf3 client
# 2) GW and iperf3 server    
###############################################################
import os
import sys

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

from demos.utils.common import runcmd, createFabric, printHeader, startGwctl
from demos.utils.kind import startKindCluster,useKindCluster, getKindIp,loadService

from demos.iperf3.kind.iperf3_client_start import directTestIperf3,testIperf3Client


############################### MAIN ##########################
if __name__ == "__main__":
    printHeader("\n\nStart Kind Test\n\n")
    printHeader("Start pre-setting")

    # GW parameters 
    gwPort           = "30443"
    gw1Name          = "peer1"
    gw2Name          = "peer2"
    srcSvc           = "iperf3-client"
    destSvc          = "iperf3-server"
    destPort         = 5000
    iperf3DirectPort = "30001"
    
    # Folders
    folCl=f"{proj_dir}/demos/iperf3/testdata/manifests/iperf3-client"
    folSv=f"{proj_dir}/demos/iperf3/testdata/manifests/iperf3-server"
    testOutputFolder = f"{proj_dir}/bin/tests/iperf3"

    # Policy
    allowAllPolicy=f"{proj_dir}/pkg/policyengine/policytypes/examples/allowAll.json"
    
    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)
    
    ### build docker environment 
    printHeader(f"Build docker image")
    os.system("make docker-build")
    os.system("sudo make install")
    
    ### Build Kind clusters environment 
    createFabric(testOutputFolder)
    startKindCluster(gw1Name, testOutputFolder)        
    startKindCluster(gw2Name, testOutputFolder)        
      
    ###get Gateways parameters
    gw1Ip = getKindIp(gw1Name)
    gw2Ip = getKindIp(gw2Name)
   
    # Start gwctl
    startGwctl(gw1Name, gw1Ip, gwPort, testOutputFolder)
    startGwctl(gw2Name, gw2Ip, gwPort, testOutputFolder)
    
    # Create peers
    printHeader("Create peers")
    runcmd(f'gwctl create peer --myid {gw1Name} --name {gw2Name} --host {gw2Ip} --port {gwPort}')
    runcmd(f'gwctl create peer --myid {gw2Name} --name {gw1Name} --host {gw1Ip} --port {gwPort}')
    
    # Set service iperf3-client in gw1
    loadService(srcSvc,gw1Name, "mlabbe/iperf3",f"{folCl}/iperf3-client.yaml" )
    runcmd(f'gwctl create export --myid {gw1Name} --name {srcSvc} --host {srcSvc} --port {destPort}')

    # Set service iperf3-server in gw2
    loadService(destSvc,gw2Name, "mlabbe/iperf3",f"{folSv}/iperf3.yaml" )
    runcmd(f"kubectl create service nodeport {destSvc} --tcp={destPort}:{destPort} --node-port={iperf3DirectPort}")
    runcmd(f'gwctl create export --myid {gw2Name} --name {destSvc} --host {destSvc} --port {destPort}')

    #Import destination service
    printHeader(f"\n\nStart Importing {destSvc} service to {gw1Name}")
    runcmd(f'gwctl --myid {gw1Name} create import --name {destSvc} --host {destSvc} --port {destPort}')
    printHeader(f"\n\nStart binding {destSvc} service to {gw1Name}")
    runcmd(f'gwctl --myid {gw1Name} create binding --import {destSvc} --peer {gw2Name}')

    #Add policy
    printHeader("Applying policies")
    runcmd(f'gwctl --myid {gw1Name} create policy --type access --policyFile {allowAllPolicy}')
    runcmd(f'gwctl --myid {gw2Name} create policy --type access --policyFile {allowAllPolicy}')
    
    #Testing
    printHeader("\n\nStart Iperf3 testing")
    directTestIperf3(gw1Name, srcSvc, gw2Ip, iperf3DirectPort)
    testIperf3Client(gw1Name, srcSvc, destSvc, destPort)


