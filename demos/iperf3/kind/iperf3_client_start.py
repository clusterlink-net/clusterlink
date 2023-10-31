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
import subprocess as sp
import sys

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

from demos.utils.common import printHeader
from demos.utils.k8s import waitPod, getPodName
from demos.utils.kind import useKindCluster

def iperf3Test(cmd ,blockFlag=False):
    print(cmd)
    testPass=False
    try:
        direct_output = sp.check_output(cmd,shell=True)
        printHeader("iPerf3 Test Results:\n") 
        print(f"{direct_output.decode()}")
        if "iperf Done" in direct_output.decode():
            testPass=True
    
    except sp.CalledProcessError as e:
        print(f"Test Code:{e.returncode}")
        if blockFlag and e.returncode == 1:
            testPass =True
            printHeader("Test block succeed") 

    print("***************************************")
    if testPass:
        print('iPerf3 Connection Succeeded')
    else:
        print('iPerf3 Connection Failed')
    print("***************************************")

def testIperf3Client(gwName,srcSvc, destSvc,destPort,blockFlag=False):
    useKindCluster(gwName)
    waitPod("iperf3-client")
    podIperf3= getPodName(srcSvc)
    printHeader("Starting Client Service(iperf3 client1)->peer1->peer2->Dest Service(iperf3 server)")
    cmd = f'kubectl exec -i {podIperf3} --  iperf3 -c {destSvc} -p {destPort}'
    iperf3Test(cmd,blockFlag)

def directTestIperf3(gwName,srcSvc,destkindIp,iperf3DestPort):
    useKindCluster(gwName)
    waitPod("iperf3-client")
    podIperf3= getPodName(srcSvc)
    printHeader("The Iperf3 test connects directly to the destination")
    cmd = f'kubectl exec -i {podIperf3} --  iperf3 -c {destkindIp} -p {iperf3DestPort}'
    iperf3Test(cmd)
