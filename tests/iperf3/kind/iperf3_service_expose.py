#!/usr/bin/env python3
import os,time
import subprocess as sp
import sys
import argparse

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

from tests.utils.mbgAux import runcmd, runcmdb, printHeader, getPodName
from tests.utils.kind.kindAux import useKindCluster

def exposeService(mbgName, gwctlName, destSvc):
    useKindCluster(mbgName)
    gwctlPod = getPodName("gwctl")
    printHeader(f"\n\nStart exposing {destSvc} service to {mbgName}")
    runcmd(f'kubectl exec -i {gwctlPod} -- ./gwctl expose --service {destSvc}')


def bindService(mbgName, destSvc, port):
    useKindCluster(mbgName)
    gwctlPod = getPodName("gwctl")
    printHeader(f"\n\nStart binding {destSvc} service to {mbgName}")
    runcmd(f'kubectl exec -i {gwctlPod} -- ./gwctl add binding --service {destSvc} --port {port}')
############################### MAIN ##########################
if __name__ == "__main__":
    #parameters 
    mbg2Name     = "mbg2"
    gwctl2Name  = "gwctl2"
    destSvc      = "iperf3-server"
    
        
    
    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)

    exposeService(mbg2Name, gwctl2Name, destSvc)
    

    