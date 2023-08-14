#!/usr/bin/env python3
import os,time
import subprocess as sp
import sys
import argparse

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

from demos.utils.mbgAux import runcmd, runcmdb, printHeader, waitPod, getPodName
from demos.utils.kind.kindAux import useKindCluster


def getService(gwctlName, destSvc):
    printHeader(f"\n\Get imported serviced from {gwctlName}")
    runcmd(f'gwctl get import --myid {gwctlName}')
    runcmd(f'gwctl get binding --myid {gwctlName} --import {destSvc}')


############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-m','--mbg', help='Either mbg1/mbg2/mbg3', required=False, default="mbg1")
    
    args = vars(parser.parse_args())
    mbg       = args["mbg"]
    
    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)

    ### build Kind clusters environment 
    if mbg in ["mbg1", "mbg2","mbg3"]:
        gwctlName     = mbg[:-1]+"ctl"+ mbg[-1]
        getService(mbg, gwctlName)
    else:
        print("mbg value should be mbg1, mbg2 or mbg3")


    