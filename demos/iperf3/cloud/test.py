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

from demos.utils.common import runcmd, printHeader
from demos.utils.k8s import  getPodNameIp
from demos.utils.cloud import cluster
from demos.iperf3.test import iperf3Test

gw1gcp = cluster(name="peer1", zone = "us-west1-b", platform = "gcp") 
gw1ibm = cluster(name="peer1", zone = "dal10",      platform = "ibm")
gw2gcp = cluster(name="peer2", zone = "us-west1-b", platform = "gcp")
gw2ibm = cluster(name="peer2", zone = "dal10",      platform = "ibm")

srcSvc           = "iperf3-client"
destSvc          = "iperf3-server"
destPort         = 5000
iperf3DirectPort = "30001"

# Folders
folCl=f"{projDir}/demos/iperf3/testdata/manifests/iperf3-client"
folSv=f"{projDir}/demos/iperf3/testdata/manifests/iperf3-server"
testOutputFolder = f"{projDir}/bin/tests/iperf3"

# Policy
allowAllPolicy=f"{projDir}/pkg/policyengine/policytypes/examples/allowAll.json"

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-c','--command', help='Script command: test/delete', required=False, default="test")
    parser.add_argument('-m','--machineType', help='Type of machine to create small/large', required=False, default="small")
    parser.add_argument('-cloud','--cloud', help='Cloud setup using gcp/ibm/diff (different clouds)', required=False, default="gcp")
    parser.add_argument('-delete','--deleteCluster', help='Delete clusters in the end of the test', required=False, default="true")
    parser.add_argument('-l','--logLevel', help='The log level. One of fatal, error, warn, info, debug.', required=False, default="info")
    parser.add_argument('-d','--dataplane', help='Which dataplane to use envoy/go', required=False, default="envoy")

    args = vars(parser.parse_args())

    command = args["command"]
    cloud = args["cloud"]
    dltCluster = args["deleteCluster"]
    machineType = args["machineType"]
    gw1 = gw1gcp if cloud in ["gcp","diff"] else gw1ibm
    gw2 = gw2gcp if cloud in ["gcp"]        else gw2ibm
    print(f'Working directory {projDir}')
    os.chdir(projDir)
    
    if command =="delete":
        gw1.deleteCluster(runBg=True)
        gw2.deleteCluster()

        exit()
    elif command =="clean":
        gw1.cleanCluster()
        gw2.cleanCluster()
        exit()

    ### build docker environment 
    printHeader("Build docker image")
    os.system("make build")
    os.system("sudo make install")
    
    gw1.machineType=machineType
    gw2.machineType=machineType
    
    iperf3Test(gw1, gw2, testOutputFolder, args["logLevel"], args["dataplane"])

    # iPerf3 test
    gw1.useCluster()
    podIperf3,_= getPodNameIp(srcSvc)

    for i in range(2):
        printHeader(f"iPerf3 test {i}")
        cmd = f'kubectl exec -i {podIperf3} --  iperf3 -c iperf3-server -p {5000} -t 40'
        runcmd(cmd)
