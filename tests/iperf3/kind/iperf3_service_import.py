#!/usr/bin/env python3
import os,time
import subprocess as sp
import sys
import argparse

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

from tests.utils.mbgAux import runcmd, runcmdb, printHeader, getPodName
from tests.utils.kind.kindAux import useKindCluster


def importService(mbgName,gwctlName,destSvc,destPort, peer):
    printHeader(f"\n\nStart Importing {destSvc} service to {mbgName}")
    runcmd(f'gwctl --myid {gwctlName} create import --name {destSvc} --host {destSvc} --port {destPort}')
    printHeader(f"\n\nStart binding {destSvc} service to {mbgName}")
    runcmd(f'gwctl --myid {gwctlName} create binding --import {destSvc} --peer {peer}')

############################### MAIN ##########################
if __name__ == "__main__":
    #parameters 
    mbg1Name    = "mbg1"
    gwctl1Name  = "gwctl1"
    mbg2Name    = "mbg2"
    gwctl2Name  = "gwctl2"
    destSvc     = "iperf3-server"
    destPort    = "5000"

        
    
    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)

    importService(mbg2Name, gwctl2Name, destSvc,destPort,mbg2Name)
    