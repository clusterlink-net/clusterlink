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
import argparse
projDir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{projDir}')

from demos.utils.common import printHeader
from demos.utils.kind import cluster
from demos.iperf3.kind.iperf3_client_start import directTestIperf3,testIperf3Client
from demos.iperf3.test import iperf3Test

testOutputFolder = f"{projDir}/bin/tests/iperf3"

############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-l','--logLevel', help='The log level. One of fatal, error, warn, info, debug.', required=False, default="info")
    parser.add_argument('-d','--dataplane', help='Which dataplane to use envoy/go', required=False, default="envoy")

    args = vars(parser.parse_args())
    printHeader("\n\nStart Kind Test\n\n")
    printHeader("Start pre-setting")

    # GW parameters 
    gw1= cluster("peer1")
    gw2= cluster("peer2")
    srcSvc           = "iperf3-client"
    destSvc          = "iperf3-server"
    destPort         = 5000
    iperf3DirectPort = "30001"
    
    # Setup
    iperf3Test(gw1, gw2, testOutputFolder, args["logLevel"], args["dataplane"])
    #Testing
    printHeader("\n\nStart Iperf3 testing")
    directTestIperf3(gw1, srcSvc, gw2.ip, iperf3DirectPort)
    testIperf3Client(gw1, srcSvc, destSvc, destPort)


