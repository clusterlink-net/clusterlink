#!/usr/bin/env python3
import os,time
import subprocess as sp
import sys
import argparse

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

from tests.utils.mbgAux import runcmd, runcmdb, printHeader, waitPod, getPodName
from tests.utils.kind.kindAux import useKindCluster

def getService(mbgName):
    useKindCluster(mbgName)
    mbgctlPod = getPodName("mbgctl")
    printHeader(f"\n\Query service from {mbgName}")
    runcmd(f'kubectl exec -i {mbgctlPod} -- ./mbgctl getService')
    


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
        getService(mbg)
    else:
        print("mbg value should be mbg1, mbg2 or mbg3")


    