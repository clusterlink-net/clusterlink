#!/usr/bin/env python3
import os,time
import subprocess as sp
import sys
import argparse

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

from tests.utils.mbgAux import runcmd, runcmdb, printHeader, waitPod, getPodName, getMbgPorts,buildMbg,buildMbgctl,getPodIp
from tests.utils.kind.kindAux import useKindCluster,getKindIp

iperf3DestPort  = "30001"
#folders
folCl=f"{proj_dir}/tests/iperf3/manifests/iperf3-client"
folSv=f"{proj_dir}/tests/iperf3/manifests/iperf3-server"
    
def setIperf3client(mbgName, mbgctlName,srcSvc):
    printHeader(f"Create {srcSvc} (client) service in {mbgName}")
    useKindCluster(mbgName)
    runcmd(f"kind load docker-image mlabbe/iperf3 --name={mbgName}")
    runcmd(f"kubectl create -f {folCl}/{srcSvc}.yaml")
    waitPod(srcSvc)
    mbgctlPod =getPodName("mbgctl")
    runcmd(f'kubectl exec -i {mbgctlPod} -- ./mbgctl add service --id {srcSvc}')

def setIperf3Server(mbgName, mbgctlName, destSvc):
    printHeader(f"Add {destSvc} (server) service in {mbgName}")
    useKindCluster(mbgName)
    runcmd(f"kind load docker-image mlabbe/iperf3 --name={mbgName}")
    runcmd(f"kubectl create -f {folSv}/iperf3.yaml")
    waitPod(destSvc)
    runcmd(f"kubectl create service nodeport iperf3-server --tcp=5000:5000 --node-port={iperf3DestPort}")
    destSvcPort = f"5000"
    mbgctlPod =getPodName("mbgctl")
    destSvcIp  = "iperf3-server"
    runcmd(f'kubectl exec -i {mbgctlPod} -- ./mbgctl add service --id {destSvc} --target {destSvcIp} --port {destSvcPort} --description iperf3-server')

############################### MAIN ##########################
if __name__ == "__main__":
    #parameters 
    mbg1Name    = "mbg1"
    srcSvc      = "iperf3-client"
    srcSvc2     = "iperf3-client2"
    mbg2Name    = "mbg2"
    mbgctl2Name = "mbgctl2"
    destSvc     = "iperf3-server"
    mbg3Name    = "mbg3"
        
    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)
    
    # Set iperf3-client in MBG1
    setIperf3client(mbg1Name, srcSvc)
    
    #Set iperf3-service in MBG2
    setIperf3Server(mbg2Name,destSvc)

    # Set iperf3-client in MBG3
    setIperf3client(mbg3Name, srcSvc)
    setIperf3client(mbg3Name, srcSvc2)
    