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
    #parameters 
    mbg1ClusterName ="mbg-agent1"
    srcSvc          = "iperf3-client"
    mbg2ClusterName = "mbg-agent2"
    mbgctl2Name     = "mbgctl2"
    destSvc         = "iperf3-server"
    mbg3ClusterName = "mbg-agent3"
        
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
    
    #Expose destination service
    useKindCluster(mbg2ClusterName)
    printHeader("\n\nStart exposing {destSvc} service to MBG2")
    runcmdb(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl expose --serviceId {destSvc}')

    #Get services
    useKindCluster(mbg1ClusterName)
    printHeader("\n\Query service from MBG1")
    runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl getService')

    useKindCluster(mbg3ClusterName)
    printHeader("\n\Query service from MBG3")
    runcmd(f'kubectl exec -i {mbgctl3Pod} -- ./mbgctl getService')


    