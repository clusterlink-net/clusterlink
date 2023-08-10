#!/usr/bin/env python3
import os,time
import subprocess as sp
import sys
import argparse

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

from demos.utils.mbgAux import runcmd, runcmdb, printHeader, getPodName
from demos.utils.kind.kindAux import getKindIp


def exportExternalService(gwctlName,svcName, destSvc,destPort, externalHost, externalPort):
    runcmd(f'gwctl --myid {gwctlName} create export --name {svcName} --host {destSvc} --port {destPort} --external {externalHost}:{externalPort}')

############################### MAIN ##########################
if __name__ == "__main__":
    #parameters 
    mbg1Name    = "mbg1"
    gwctl1Name  = "gwctl1"
    mbg2Name    = "mbg2"
    gwctl2Name  = "gwctl2"
    destSvc     = "iperf3-server-external"
    destPort    = "5000"
    
    externalPort ="30001"
        
    mbg2Ip = getKindIp(mbg2Name)

    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)

    exportExternalService(gwctl2Name, destSvc, destPort, mbg2Ip, externalPort )
    