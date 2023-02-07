#!/usr/bin/env python3
import os,time
import subprocess as sp
import sys
import argparse

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))

sys.path.insert(0,f'{proj_dir}/tests/')
#print(f"{proj_dir}/tests/")
from tests.utils.mbgAux import runcmd, runcmdb, printHeader, waitPod, getPodName, getKindIp, getMbgPorts,buildMbg,buildMbgctl,getPodIp
from tests.utils.kind.kindAux import useKindCluster

from dotenv import load_dotenv

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


############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-m','--mbg', help='Either mbg1/mbg3', required=False, default="mbg1")
    args = vars(parser.parse_args())

    mbg = args["mbg"]
    #MBG1 parameters 
    mbg1ClusterName = "mbg-agent1"
    srcSvc          = "iperf3-client"
    srcSvc2          = "iperf3-2-client"

    #MBG2 parameters 
    destSvc         = "iperf3-server"    
    #MBG3 parameters 
    mbg3ClusterName = "mbg-agent3"
        
    #print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)

    load_dotenv()
    podMbg1 = os.getenv("podMbg1")
    mbg1Ip = os.getenv("mbg1Ip")
    mbgctl1Pod = os.getenv("mbgctl1Pod")
    podMbg2 = os.getenv("podMbg2")
    mbg2Ip = os.getenv("mbg2Ip")
    mbgctl2Pod = os.getenv("mbgctl2Pod")
    podMbg3 = os.getenv("podMbg3")
    mbg3Ip = os.getenv("mbg3Ip")
    mbgctl3Pod = os.getenv("mbgctl3Pod")

    if mbg == "mbg1":
        #Test MBG1
        useKindCluster(mbg1ClusterName)
        waitPod("iperf3-client")
        podIperf3= getPodName("iperf3-clients")
        mbg1LocalPort, mbg1ExternalPort = getMbgPorts(podMbg1,destSvc+"-MBG2")
        printHeader("Starting Client Service(iperf3 client1)->MBG1->MBG2->Dest Service(iperf3 server)")
        cmd = f'kubectl exec -i {podIperf3} --  iperf3 -c {getPodIp(podMbg1) } -p {mbg1LocalPort}'
        iperf3Test(cmd)

    elif mbg == "mbg3":
        #Test MBG3
        useKindCluster(mbg3ClusterName)
        waitPod("iperf3-client")
        podIperf3= getPodName("iperf3-clients")
        pod2Iperf3= getPodName("iperf3-2-clients")
        mbg3LocalPort, mbg3ExternalPort = getMbgPorts(podMbg3,destSvc+"-MBG2")
        printHeader("Starting Client Service(iperf3 client)->MBG3->MBG2->Dest Service(iperf3 server)")
        cmd = f'kubectl exec -i {podIperf3} --  iperf3 -c {getPodIp(podMbg3)} -p {mbg3LocalPort}'
        iperf3Test(cmd)
        printHeader("Starting Client Service(iperf3 client2)->MBG3->MBG2->Dest Service(iperf3 server)")
        cmd = f'kubectl exec -i {pod2Iperf3} --  iperf3 -c {getPodIp(podMbg3) } -p {mbg3LocalPort}'
        iperf3Test(cmd)
    else:
        print("Please choose either mbg1/mbg3 for running iperf3 client")
