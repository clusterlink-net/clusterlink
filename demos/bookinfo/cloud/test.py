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
#Name: Service node test
#Desc: create 1 proxy that send data to target ip
###############################################################
import os
import sys
import argparse

projDir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{projDir}')
from demos.utils.cloud import cluster
from demos.bookinfo.test import bookInfoDemo


gw1gcp = cluster(name="peer1", zone = "us-west1-b"   , platform = "gcp") # Oregon
gw1ibm = cluster(name="peer1", zone = "sjc04"        , platform = "ibm") # San jose
gw2gcp = cluster(name="peer2", zone = "us-central1-b", platform = "gcp") # Iowa
gw2ibm = cluster(name="peer2", zone = "dal10"        , platform = "ibm") # Dallas
gw3gcp = cluster(name="peer3", zone = "us-east4-b"   , platform = "gcp") # Virginia
gw3ibm = cluster(name="peer3", zone = "wdc04"        , platform = "ibm") # Washington DC
testOutputFolder = f"{projDir}/bin/tests/bookinfo" 

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
    gw2 = gw2gcp if cloud in ["gcp","diff"] else gw2ibm
    gw3 = gw3gcp if cloud in ["gcp"]        else gw3ibm
    print(f'Working directory {projDir}')
    os.chdir(projDir)
    
    if command =="delete":
        gw1.deleteCluster(runBg=True)
        gw2.deleteCluster(runBg=True)
        gw3.deleteCluster()
        exit()
    elif command =="clean":
        gw1.cleanCluster()
        gw2.cleanCluster()
        gw3.cleanCluster()
        exit()

    ### build docker environment 
    os.system("make build")
    os.system("sudo make install")
    
    gw1.machineType = machineType
    gw2.machineType = machineType
    gw3.machineType = machineType
    bookInfoDemo(gw1, gw2, gw3, testOutputFolder, args["logLevel"], args["dataplane"])
    print(f"Proctpage1 url: http://{gw1.nodeIP}:30001/productpage")
    print(f"Proctpage2 url: http://{gw1.nodeIP}:30002/productpage")
