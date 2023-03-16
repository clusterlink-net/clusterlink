#!/usr/bin/env python3
import os,time
import subprocess as sp
import sys
import argparse

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

from tests.utils.mbgAux import runcmd, runcmdb, printHeader, getPodName
from tests.utils.kind.kindAux import useKindCluster

def exposeService(mbgName, mbgctlName,destSvc):
    useKindCluster(mbgName)
    mbgctlPod = getPodName("mbgctl")
    printHeader(f"\n\nStart exposing {destSvc} service to {mbgName}")
    runcmd(f'kubectl exec -i {mbgctlPod} -- ./mbgctl expose --myid {mbgctlName} --service {destSvc}')

############################### MAIN ##########################
if __name__ == "__main__":
    #parameters 
    mbgName     = "mbg3"
    mbgCtlName  = "mbgctl3"
    destSvc     = "reviews"
    
    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)

    exposeService(mbgName,mbgCtlName,destSvc)
    

    