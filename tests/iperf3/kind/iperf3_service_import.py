#!/usr/bin/env python3
import os,time
import subprocess as sp
import sys
import argparse

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

from tests.utils.mbgAux import runcmd, runcmdb, printHeader, getPodName
from tests.utils.kind.kindAux import useKindCluster

def importService(mbgName,destSvc,destPort, peer):
    useKindCluster(mbgName)
    gwctlPod = getPodName("gwctl")
    printHeader(f"\n\nStart Importing {destSvc} service to {mbgName}")
    runcmd(f'kubectl exec -i {gwctlPod} -- ./gwctl create import --name {destSvc} --host {destSvc} --port {destPort}')
    printHeader(f"\n\nStart binding {destSvc} service to {mbgName}")
    runcmd(f'kubectl exec -i {gwctlPod} -- ./gwctl create binding --import {destSvc} --peer {peer}')

############################### MAIN ##########################
if __name__ == "__main__":
    #parameters 
    mbg2Name     = "mbg2"
    gwctl2Name  = "gwctl2"
    destSvc      = "iperf3-server"
    
        
    
    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)

    importService(mbg2Name, gwctl2Name, destSvc)
    

    