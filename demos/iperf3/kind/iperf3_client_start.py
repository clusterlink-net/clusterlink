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

import os,time
import subprocess as sp
import sys
import argparse

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

from demos.utils.mbgAux import runcmd, runcmdb, printHeader, waitPod, getPodName, getPodNameIp, getMbgPorts,getPodNameApp
from demos.utils.kind.kindAux import useKindCluster

def iperf3Test(cmd ,blockFlag=False):
    print(cmd)
    testPass=False
    try:
        direct_output = sp.check_output(cmd,shell=True) #could be anything here.  
        printHeader(f"Iperf3 Test Results:\n") 
        print(f"{direct_output.decode()}")
        if "iperf Done" in direct_output.decode():
            testPass=True
    
    except sp.CalledProcessError as e:
        print(f"Test Code:{e.returncode}")
        if blockFlag and e.returncode == 1:
            testPass =True
            printHeader(f"Test block succeed") 

    print("***************************************")
    if testPass:
        print(f'iperf3 Connection Succeeded')
    else:
        print(f'iperf3 Connection Failed')
    print("***************************************")

def testIperf3Client(mbgName,srcSvc, destSvc,destPort,blockFlag=False):
    useKindCluster(mbgName)
    waitPod("iperf3-client")
    podIperf3= getPodNameApp(srcSvc)
    printHeader("Starting Client Service(iperf3 client1)->MBG1->MBG2->Dest Service(iperf3 server)")
    cmd = f'kubectl exec -i {podIperf3} --  iperf3 -c {destSvc} -p {destPort}'
    iperf3Test(cmd,blockFlag)

def directTestIperf3(mbgName,srcSvc,destkindIp,iperf3DestPort):
    useKindCluster(mbgName)
    waitPod("iperf3-client")
    podIperf3= getPodNameApp(srcSvc)
    printHeader("The Iperf3 test connects directly to the destination")
    cmd = f'kubectl exec -i {podIperf3} --  iperf3 -c {destkindIp} -p {iperf3DestPort}'
    iperf3Test(cmd)


############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-m','--mbg', help='Either mbg1/mbg3', required=False, default="mbg1")
    args = vars(parser.parse_args())

    mbg = args["mbg"]
    #MBG1 parameters 
    mbg1ClusterName = "mbg1"
    srcSvc          = "iperf3-client"
    srcSvc2         = "iperf3-client2"

    #MBG2 parameters 
    destSvc         = "iperf3-server"    
    #MBG3 parameters 
    mbg3ClusterName = "mbg3"
    destPort        = "5000"
    
    os.chdir(proj_dir)

    if mbg == "mbg1":
        #Test MBG1
        testIperf3Client(mbg, srcSvc ,destSvc, destPort)
        
    elif mbg == "mbg3":
        #Test MBG3
        testIperf3Client(mbg, srcSvc ,destSvc, destPort)
        testIperf3Client(mbg, srcSvc2 ,destSvc, destPort)
    else:
        print("Please choose either mbg1/mbg3 for running iperf3 client")
