#!/usr/bin/env python3
import os,time
import subprocess as sp
import sys
import argparse

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

from tests.utils.mbgAux import runcmd, runcmdb, printHeader, getPodName 
from tests.utils.kind.kindAux import useKindCluster, getKindIp

# Add MBG Peer
def connectMbgs(mbgName, gwctlName, gwctlPod, peerName, peerIp, peercPort):
    useKindCluster(mbgName)
    runcmd(f'kubectl exec -i {gwctlPod} -- ./gwctl add peer  --id {peerName} --target {peerIp} --port {peercPort}')
    

def sendHello(gwctlPod, gwctlName):
    # Send Hello
    printHeader("Send Hello commands")
    runcmd(f'kubectl exec -i {gwctlPod} -- ./gwctl hello')       


############################### MAIN ##########################
if __name__ == "__main__":
    
    #MBG1 parameters 
    mbg1cPort = "30443"
    mbg1Name  = "mbg1"
    
    #MBG2 parameters 
    mbg2cPort = "30443"
    mbg2Name  = "mbg2"
    gwctl2Name = "gwctl2"
        
    #MBG3 parameters 
    mbg3cPort = "30443"
    mbg3Name  = "mbg3"
        
    useKindCluster(mbg1Name)
    mbg1Ip=getKindIp(mbg1Name)
    useKindCluster(mbg3Name)
    mbg3Ip=getKindIp(mbg3Name)
    
    useKindCluster(mbg2Name)
    gwctl2Pod =getPodName("gwctl")
    printHeader("Add MBG2, MBG3 peer to MBG1")
    connectMbgs(mbg2Name, gwctl2Name, gwctl2Pod, mbg1Name, mbg1Ip, mbg1cPort)
    connectMbgs(mbg2Name, gwctl2Name, gwctl2Pod, mbg3Name, mbg3Ip, mbg3cPort)

    sendHello(gwctl2Pod, gwctl2Name)
    