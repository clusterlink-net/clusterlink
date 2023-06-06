#!/usr/bin/env python3
import os,time
import subprocess as sp
import sys
import argparse

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

from tests.utils.mbgAux import runcmd, runcmdb, printHeader, getPodName
from tests.utils.kind.kindAux import useKindCluster

def removePeer(mbgName, gwctlName, peerName):
    useKindCluster(mbgName)
    gwctlPod = getPodName("gwctl")
    printHeader(f"\n\Remove {peerName} peer to {mbgName}")
    runcmd(f'kubectl exec -i {gwctlPod} -- ./gwctl remove peer --id {peerName}')

############################### MAIN ##########################
if __name__ == "__main__":
    #parameters 
    mbgName     = "mbg1"
    mbgCtlName  = "gwctl1"
    peerName    = "mbg3"
    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)

    removePeer(mbgName,mbgCtlName,peerName)
    

    