#!/usr/bin/env python3
import os,time
import subprocess as sp
import sys
import argparse

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

from tests.utils.mbgAux import runcmd, runcmdb, printHeader, waitPod, getPodName, getPodNameIp, getMbgPorts,getPodNameApp
from tests.utils.kind.kindAux import useKindCluster

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

def testIperf3Client(mbgName,srcSvc, destSvc, blockFlag=False):
    useKindCluster(mbgName)
    waitPod("iperf3-client")
    podIperf3= getPodNameApp(srcSvc)
    mbgPod,mbgIp  = getPodNameIp("mbg")
    mbgLocalPort, mbgExternalPort = getMbgPorts(mbgPod,destSvc)
    printHeader("Starting Client Service(iperf3 client1)->MBG1->MBG2->Dest Service(iperf3 server)")
    cmd = f'kubectl exec -i {podIperf3} --  iperf3 -c {mbgIp } -p {mbgLocalPort}'
    iperf3Test(cmd,blockFlag)

def directTestIperf3(mbgName,srcSvc,destSvc,destkindIp,iperf3DestPort):
    useKindCluster(mbgName)
    waitPod("iperf3-client")
    podIperf3= getPodNameApp(srcSvc)
    mbgPod,mbgIp  = getPodNameIp("mbg")
    mbg1LocalPort, mbg1ExternalPort = getMbgPorts(mbgPod,destSvc)
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
        
    os.chdir(proj_dir)

    if mbg == "mbg1":
        #Test MBG1
        testIperf3Client(mbg,srcSvc ,destSvc)
        
    elif mbg == "mbg3":
        #Test MBG3
        testIperf3Client(mbg,srcSvc ,destSvc)
        testIperf3Client(mbg,srcSvc2 ,destSvc)
    else:
        print("Please choose either mbg1/mbg3 for running iperf3 client")
