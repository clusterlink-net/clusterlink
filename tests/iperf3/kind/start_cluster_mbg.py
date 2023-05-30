#!/usr/bin/env python3
import os,time
import subprocess as sp
import sys
import argparse
proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

from tests.utils.mbgAux import printHeader
from tests.utils.kind.kindAux import useKindCluster, getKindIp, startKindClusterMbg



############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-d','--dataplane', help='choose which dataplane to use mtls/tcp', required=False, default="mtls")
    parser.add_argument('-m','--mbg', help='Either mbg1/mbg2/mbg3', required=False, default="mbg1")
    parser.add_argument('-b','--build', help='Build Image', required=False, default=False)
    parser.add_argument('-c','--cni', help='Which cni to use default(kindnet)/flannel/calico', required=False, default="default")
    parser.add_argument('-fg','--fg', help='Run MBg command in fg', action="store_true", default=False)
    parser.add_argument('-noLogFile','--noLogFile', help='Print output to the screen', action="store_false", default=True)

    args = vars(parser.parse_args())

    dataplane = args["dataplane"]
    mbg       = args["mbg"]
    build     = args["build"]
    runInfg   = args["fg"]
    cni       = args["cni"]
    logFile   = args["noLogFile"]

    #MBG parameters 
    mbgDataPort    = "30001"
    mbgcPort       = "30443"
    mbgcPortLocal  = "8443"
    mbgcrtFlags    = f"--rootCa ./mtls/ca.crt --certificate ./mtls/{mbg}.crt --key ./mtls/{mbg}.key"  if dataplane =="mtls" else ""
    gwctlName     = mbg[:-1]+"ctl"+ mbg[-1]
    
    print("Starting mbg ("+mbg+") with dataplane "+ dataplane)
        
    #print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)

    
    ### build docker environment 
    if build:
        printHeader(f"Building docker image")
        os.system("make docker-build")
    
    
    ### build Kind clusters environment 
    if mbg in ["mbg1", "mbg2","mbg3"]:
        startKindClusterMbg(mbg, gwctlName, mbgcPortLocal, mbgcPort, mbgDataPort,dataplane ,mbgcrtFlags, runInfg,cni=cni,logFile=logFile)        
    else:
        print("mbg value should be mbg1, mbg2 or mbg3")





