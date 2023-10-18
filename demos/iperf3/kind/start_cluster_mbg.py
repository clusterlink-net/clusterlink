#!/usr/bin/env python3
# Copyright 2023 The ClusterLink Authors.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import os,time
import subprocess as sp
import sys
import argparse
proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

from demos.utils.mbgAux import printHeader
from demos.utils.kind.kindAux import useKindCluster, startGwctl, getKindIp, startKindClusterMbg



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

    crtFol   = f"{proj_dir}/demos/utils/mtls"

    dataplane = args["dataplane"]
    mbg       = args["mbg"]
    build     = args["build"]
    runInfg   = args["fg"]
    cni       = args["cni"]
    logFile   = args["noLogFile"]

    #MBG parameters 
    mbgDataPort    = "30001"
    mbgcPort       = "30443"
    mbgcPortLocal  = "443"
    mbgcrtFlags    = f"--certca ./mtls/ca.crt --cert ./mtls/{mbg}.crt --key ./mtls/{mbg}.key"  if dataplane =="mtls" else ""
    gwctlcrt      = f"--certca {crtFol}/ca.crt --cert {crtFol}/{mbg}.crt --key {crtFol}/{mbg}.key"  if dataplane =="mtls" else ""
    gwctlName     = "gwctl"+ mbg[-1]
    
    print("Starting mbg ("+mbg+") with dataplane "+ dataplane)
        
    #print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)

    
    ### build docker environment 
    if build:
        printHeader(f"Building docker image")
        os.system("make docker-build")
    
    
    ### build Kind clusters environment 
    if mbg in ["mbg1", "mbg2","mbg3"]:
        startKindClusterMbg(mbg, gwctlName, mbgcPortLocal, mbgcPort, mbgDataPort,dataplane ,mbgcrtFlags, runInfg, cni=cni,logFile=logFile) 
        mbgIP = getKindIp(mbg)
        startGwctl(gwctlName, mbgIP, mbgcPort, dataplane, gwctlcrt)
    else:
        print("mbg value should be mbg1, mbg2 or mbg3")





