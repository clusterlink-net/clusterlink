#!/usr/bin/env python3
import os,time
import subprocess as sp
import sys
import argparse

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

from tests.utils.mbgAux import runcmd, runcmdb, printHeader, waitPod, getPodName, getKindIp, getMbgPorts,buildMbg,buildMbgctl,getPodIp
from tests.utils.kind.kindAux import useKindCluster

from dotenv import load_dotenv


############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-d','--dataplane', help='choose which dataplane to use mtls/tcp', required=False, default="tcp")
    parser.add_argument('-m','--mbg', help='Either mbg1/mbg2/mbg3', required=False, default="mbg1")

    args = vars(parser.parse_args())

    dataplane = args["dataplane"]
    mbg       = args["mbg"]

    #MBG1 parameters 
    mbg1cPort       = "30443"
    mbg1ClusterName ="mbg-agent1"
    
    #MBG2 parameters 
    mbg2cPort       = "30443"
    mbg2ClusterName = "mbg-agent2"
    iperf3DestPort  = "30001"
    
    #MBG3 parameters 
    mbg3cPort       = "30443"
    mbg3ClusterName = "mbg-agent3"
        

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

    # Add MBG Peer
    useKindCluster(mbg2ClusterName)
    printHeader("Add MBG2, MBG3 peer to MBG1")
    runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl addPeer --id "MBG1" --ip {mbg1Ip} --cport {mbg1cPort}')
    runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl addPeer --id "MBG3" --ip {mbg3Ip} --cport {mbg3cPort}')
        
    # Send Hello
    printHeader("Send Hello commands")
    runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl hello')       