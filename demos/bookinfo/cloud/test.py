#!/usr/bin/env python3
# Copyright (c) The ClusterLink Authors.
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
from demos.utils.cloud import Cluster
from demos.bookinfo.test import bookInfoDemo


cl1gcp = Cluster(name="peer1", zone = "us-west1-b"   , platform = "gcp") # Oregon
cl1ibm = Cluster(name="peer1", zone = "sjc04"        , platform = "ibm") # San jose
cl2gcp = Cluster(name="peer2", zone = "us-central1-b", platform = "gcp") # Iowa
cl2ibm = Cluster(name="peer2", zone = "dal10"        , platform = "ibm") # Dallas
cl3gcp = Cluster(name="peer3", zone = "us-east4-b"   , platform = "gcp") # Virginia
cl3ibm = Cluster(name="peer3", zone = "wdc04"        , platform = "ibm") # Washington DC
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
    cl1 = cl1gcp if cloud in ["gcp","diff"] else cl1ibm
    cl2 = cl2gcp if cloud in ["gcp","diff"] else cl2ibm
    cl3 = cl3gcp if cloud in ["gcp"]        else cl3ibm
    print(f'Working directory {projDir}')
    os.chdir(projDir)

    if command =="delete":
        cl1.deleteCluster(runBg=True)
        cl2.deleteCluster(runBg=True)
        cl3.deleteCluster()
        exit()
    elif command =="clean":
        cl1.cleanCluster()
        cl2.cleanCluster()
        cl3.cleanCluster()
        exit()

    ### build docker environment
    os.system("make build")
    os.system("make install")

    cl1.machineType = machineType
    cl2.machineType = machineType
    cl3.machineType = machineType
    bookInfoDemo(cl1, cl2, cl3, testOutputFolder, args["logLevel"], args["dataplane"])
    print(f"Productpage1 url: http://{cl1.nodeIP}:30001/productpage")
    print(f"Productpage2 url: http://{cl1.nodeIP}:30002/productpage")
