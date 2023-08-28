#!/usr/bin/env python3
import os
import sys
import argparse

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

from demos.utils.mbgAux import runcmd, printHeader


def applyAccessPolicy(mbgName, gwctlName, policyFile):
    printHeader(f"\n\nApplying policy file {policyFile} to {mbgName}")
    runcmd(f'gwctl --myid {gwctlName} create policy --type access --policyFile {policyFile}')

############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Apply access policies')
    parser.add_argument('-m','--mbg', help='Either mbg1/mbg2/mbg3', required=True)
    parser.add_argument('-f','--file', help='Policy file to apply', required=True)

    args = vars(parser.parse_args())

    mbg = args["mbg"]
    policyFile = args["file"]
    gwctlName = "gwctl"+ mbg[-1]        
    
    applyAccessPolicy(mbg, gwctlName, policyFile)
