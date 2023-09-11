#!/usr/bin/env python3
import os
import sys
import argparse

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')
from demos.utils.mbgAux import runcmd, printHeader

srcSvc   = "iperf3-client"
destSvc  = "iperf3-server"
denyIper3Policy=f"{proj_dir}/demos/iperf3/testdata/policy/denyToIperf3.json"
denyGw3Policy=f"{proj_dir}/demos/iperf3/testdata/policy/denyFromGw.json"

    
def applyAccessPolicy(mbgName, gwctlName, policyFile):
    printHeader(f"\n\nApplying policy file {policyFile} to {mbgName}")
    runcmd(f'gwctl --myid {gwctlName} create policy --type access --policyFile {policyFile}')

def deleteAccessPolicy(gwctlName, policyFile):
    runcmd(f'gwctl delete policy --myid {gwctlName} --type access --policyFile {policyFile}')
    
def applyPolicy(mbg, gwctlName, type, srcSvc=srcSvc,destSvc=destSvc ):
    if mbg in ["mbg1","mbg3"]:
        if type == "deny":
            printHeader(f"Block Traffic in {mbg}")          
            applyAccessPolicy(mbg, gwctlName, denyIper3Policy)
        elif type == "allow": # Remove the deny policy
            printHeader(f"Allow Traffic in {mbg}")
            deleteAccessPolicy(gwctlName, denyIper3Policy)
        elif type == "show":
            printHeader(f"Show Policies in {mbg}")
            runcmd(f'gwctl get policy --myid {gwctlName}')

        else:
            print("Unknown command")
    if mbg == "mbg2":
        if type == "deny":
            printHeader("Block Traffic in MBG2")
            applyAccessPolicy(mbg, gwctlName, denyGw3Policy)
        elif type == "allow": # Remove the deny policy
            printHeader("Allow Traffic in MBG2")
            deleteAccessPolicy(gwctlName, denyGw3Policy)
        else:
            print("Unknown command")


############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-m','--mbg', help='Either mbg1/mbg2/mbg3', required=True, default="mbg1")
    parser.add_argument('-t','--type', help='Either allow/deny/show', required=False, default="allow")

    args = vars(parser.parse_args())

    mbg = args["mbg"]
    type = args["type"]
    gwctlName     = mbg[:-1]+"ctl"+ mbg[-1]

    
    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)

    applyPolicy(mbg, gwctlName, type)
    