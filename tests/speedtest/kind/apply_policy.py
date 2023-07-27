#!/usr/bin/env python3
import os,time
import subprocess as sp
import sys
import argparse

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

srcSvc   = "firefox"
destSvc  = "openspeedtest"
    
    
def applyPolicy(mbg, gwctlName, type,srcSvc=srcSvc,destSvc=destSvc ):
    if mbg in ["mbg1","mbg3"]:
        useKindCluster(mbg)
        gwctlPod=getPodName("gwctl")
        if type == "deny":
            printHeader(f"Block Traffic in {mbg}")          
            runcmd(f'gwctl create policy --myid {gwctlName} --type acl --serviceSrc {srcSvc} --serviceDst {destSvc} --gwDest mbg2 --priority 0 --action 1')
        elif type == "allow":
            printHeader(f"Allow Traffic in {mbg}")
            runcmd(f'gwctl delete policy --myid {gwctlName} --type acl --serviceSrc {srcSvc} --serviceDst {destSvc} --gwDest mbg2')
        elif type == "show":
            printHeader(f"Show Policies in {mbg}")
            runcmd(f'gwctl get policy --myid {gwctlName}')

        else:
            print("Unknown command")
    if mbg == "mbg2":
        useKindCluster(mbg)
        gwctl2Pod=getPodName("gwctl")
        if type == "deny":
            printHeader("Block Traffic in MBG2")
            runcmd(f'gwctl create policy --myid {gwctlName} --type acl --gwDest mbg3 --priority 0 --action 1')
        elif type == "allow":
            printHeader("Allow Traffic in MBG2")
            runcmd(f'gwctl delete policy --myid {gwctlName} --type acl --gwDest mbg3')
        else:
            print("Unknown command")


from tests.utils.mbgAux import runcmd, runcmdb, printHeader, getPodName
from tests.utils.kind.kindAux import useKindCluster

############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-m','--mbg', help='Either mbg1/mbg2/mbg3', required=True, default="mbg1")
    parser.add_argument('-t','--type', help='Either allow/deny/show', required=False, default="allow")

    args = vars(parser.parse_args())

    mbg = args["mbg"]
    type = args["type"]
    gwctlName = mbg[:-1]+"ctl"+ mbg[-1]


    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)

    applyPolicy(mbg, gwctlName, type)
    