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
def connectMbgs( gwctlName, peerName, peerIp, peercPort):
    runcmd(f'gwctl create peer --myid {gwctlName} --name {peerName} --host {peerIp} --port {peercPort}')
    
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
    printHeader("Add MBG2, MBG3 peer to MBG1")
    connectMbgs(gwctl2Name, mbg1Name, mbg1Ip, mbg1cPort)
    connectMbgs(gwctl2Name, mbg3Name, mbg3Ip, mbg3cPort)
