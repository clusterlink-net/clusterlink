#!/usr/bin/env python3
import os,time
import subprocess as sp
import sys
import argparse

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))

sys.path.insert(0,f'{proj_dir}/tests/')
print(f"{proj_dir}/tests/")
from aux.kindAux import runcmd, runcmdb, printHeader, waitPod, getPodName, getKindIp, getMbgPorts,buildMbg,buildMbgctl,useKindCluster,getPodIp
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
        print(f'Test Pass')
    else:
        print(f'Test Fail')
    print("***************************************")


############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-m','--mbg', help='Either mbg1/mbg2/mbg3', required=True, default="mbg1")
    parser.add_argument('-t','--type', help='Either allow/deny/show', required=False, default="allow")

    args = vars(parser.parse_args())

    mbg = args["mbg"]
    type = args["type"]
    #MBG1 parameters 
    mbg1DataPort    = "30001"
    mbg1cPort       = "30443"
    mbg1cPortLocal  = "8443"
    mbg1ClusterName ="mbg-agent1"
    mbgctl1Name     = "mbgctl1"
    srcSvc          = "iperf3-client"
    srcDefaultGW    = "10.244.0.1"
    srck8sSvcPort   = "5000"
    
    #MBG2 parameters 
    mbg2DataPort    = "30001"
    mbg2cPort       = "30443"
    mbg2cPortLocal  = "8443"
    mbg2ClusterName = "mbg-agent2"
    mbgctl2Name     = "mbgctl2"
    destSvc         = "iperf3-server"
    iperf3DestPort  = "30001"
    
    #MBG3 parameters 
    mbg3DataPort    = "30001"
    mbg3cPort       = "30443"
    mbg3cPortLocal  = "8443"
    mbg3ClusterName = "mbg-agent3"
    mbgctl3Name     = "mbgctl3"
    srcSvc          = "iperf3-client"
    srcDefaultGW    = "10.244.0.1"
    srck8sSvcPort   = "5000"
        
    #folders
    folCl=f"{proj_dir}/tests/iperf3/manifests/iperf3-client"
    folSv=f"{proj_dir}/tests/iperf3/manifests/iperf3-server"
    
    print(f'Working directory {proj_dir}')
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
    
    

    #Block Traffic in MBG3
    if mbg == "mbg3":
        if type == "deny":
            printHeader("Block Traffic in MBG3")
            useKindCluster(mbg3ClusterName)
            runcmd(f'kubectl exec -i {mbgctl3Pod} -- ./mbgctl policy --command acl_add --serviceSrc {srcSvc} --serviceDst {destSvc} --mbgDest MBG2 --priority 0 --action 1')
        elif type == "allow":
            printHeader("Allow Traffic in MBG3")
            useKindCluster(mbg3ClusterName)
            runcmd(f'kubectl exec -i {mbgctl3Pod} -- ./mbgctl policy --command acl_del --serviceSrc {srcSvc} --serviceDst {destSvc} --mbgDest MBG2 --priority 0 --action 1')
        elif type == "show":
            printHeader("Show Policies in MBG3")
            useKindCluster(mbg3ClusterName)
            runcmd(f'kubectl exec -i {mbgctl3Pod} -- ./mbgctl policy --command show')

        else:
            print("Unknown command")
    if mbg == "mbg2":
        if type == "deny":
            printHeader("Block Traffic in MBG2")
            useKindCluster(mbg2ClusterName)
            runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl policy --command acl_add --mbgDest MBG3 --priority 0 --action 1')
        elif type == "allow":
            printHeader("Allow Traffic in MBG2")
            useKindCluster(mbg2ClusterName)
            runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl policy --command acl_del --mbgDest MBG3 --priority 0 --action 1')
        else:
            print("Unknown command")
