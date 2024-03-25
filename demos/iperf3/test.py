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
# 1) GW and iPerf3 client
# 2) GW and iPerf3 server    
###############################################################
import os
import sys

projDir = os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__))))
sys.path.insert(0,f'{projDir}')

from demos.utils.common import runcmd, createFabric, printHeader, startGwctl
from demos.utils.kind import *

# Folders
folCl=f"{projDir}/demos/iperf3/testdata/manifests/iperf3-client"
folSv=f"{projDir}/demos/iperf3/testdata/manifests/iperf3-server"
# Policy
allowAllPolicy=f"{projDir}/pkg/policyengine/examples/allowAll.json"
#services
srcSvc           = "iperf3-client"
destSvc          = "iperf3-server"    
destPort         = 5000

# iperf3Test setup two cluster for creating iPerf3 test.
def iperf3Test(cl1:cluster, cl2:cluster, testOutputFolder,logLevel="info" ,dataplane="envoy"):    
    print(f'Working directory {projDir}')
    os.chdir(projDir)
    
    # build docker environment 
    printHeader("Build docker image")
    os.system("make docker-build")
    os.system("sudo make install")
    
    # Create Kind clusters environment 
    cl1.createCluster(runBg=True)        
    cl2.createCluster(runBg=False)  
    
    # Start Kind clusters environment 
    createFabric(testOutputFolder)
    cl1.startCluster(testOutputFolder,logLevel, dataplane)        
    cl2.startCluster(testOutputFolder,logLevel, dataplane)        
      
    # Start gwctl
    startGwctl(cl1.name, cl1.ip, cl1.port, testOutputFolder)
    startGwctl(cl2.name, cl2.ip, cl2.port, testOutputFolder)
    
    # Create iPerf3 micto-services
    cl1.loadService(srcSvc, "mlabbe/iperf3",f"{folCl}/iperf3-client.yaml" )
    cl2.loadService(destSvc, "mlabbe/iperf3",f"{folSv}/iperf3.yaml" )
    
    # Create peers
    printHeader("Create peers")
    runcmd(f'gwctl create peer --myid {cl1.name} --name {cl2.name} --host {cl2.ip} --port {cl1.port}')
    runcmd(f'gwctl create peer --myid {cl2.name} --name {cl1.name} --host {cl1.ip} --port {cl2.port}')
    
    # Create exports
    runcmd(f'gwctl create export --myid {cl1.name} --name {srcSvc} --host {srcSvc} --port {destPort}')
    runcmd(f'gwctl create export --myid {cl2.name} --name {destSvc} --host {destSvc} --port {destPort}')

    #Import destination service
    printHeader(f"\n\nStart Importing {destSvc} service to {cl1.name}")
    runcmd(f'gwctl --myid {cl1.name} create import --name {destSvc} --port {destPort} --peer {cl2.name}')

    #Add policy
    printHeader("Applying policies")
    runcmd(f'gwctl --myid {cl1.name} create policy --type access --policyFile {allowAllPolicy}')
    runcmd(f'gwctl --myid {cl2.name} create policy --type access --policyFile {allowAllPolicy}')
    
