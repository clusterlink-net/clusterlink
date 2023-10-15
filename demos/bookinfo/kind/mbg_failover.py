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

from demos.utils.mbgAux import runcmd, runcmdb, printHeader, getPodName, waitPod,getMbgPorts,buildMbg,buildMbgctl,getPodIp,getPodNameIp
from demos.utils.kind.kindAux import useKindCluster,startKindClusterMbg,getKindIp

srcSvc1  = "productpage"
srcSvc2  = "productpage2"
destSvc  = "reviews"
    

#MBG3 parameters 
mbg3DataPort    = "30001"
mbg3cPort       = "30443"
mbg3cPortLocal  = "443"
mbg3crtFlags    = "--certca ./mtls/ca.crt --cert ./mtls/mbg3.crt --key ./mtls/mbg3.key"
mbg3Name        = "mbg3"
gwctl3Name     = "gwctl3"

destSvc      = "reviews"
    

def exposeService(mbgName, mbgCtlName, destSvc):
    gwctlPod = getPodName("gwctl")
    printHeader(f"\n\nStart exposing {destSvc} service to {mbgName}")
    runcmd(f'kubectl exec -i {gwctlPod} -- ./gwctl expose --service {destSvc}')


def applyFail(mbg, gwctlName, type):
    useKindCluster(mbg)
    mPod=getPodName("mbg-")
    print(mPod)
    mbgKindIp=getKindIp(mbg)
    print(mbgKindIp)

    if type == "fail":
        printHeader(f"Failing MBG")
        runcmd(f'kubectl exec -i {mPod} -- killall mbg')
    elif type == "start":
        printHeader(f"Starting up and Restoring MBG")
        runcmdb(f'kubectl exec -i {mPod} -- ./controlplane start --name "{mbg3Name}" --ip {mbgKindIp} --cport {mbg3cPort} --cportLocal {mbg3cPortLocal}  --externalDataPortRange {mbg3DataPort}\
    --dataplane mtls {mbg3crtFlags} --startPolicyEngine {True} --restore {True}')
        time.sleep(2)
        exposeService(mbg, gwctlName, destSvc)


from demos.utils.mbgAux import runcmd, runcmdb, printHeader, getPodName
from demos.utils.kind.kindAux import useKindCluster

############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-t','--type', help='Either fail/start', required=False, default="fail")

    args = vars(parser.parse_args())

    type = args["type"]
    
    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)

    applyFail(mbg3Name, gwctl3Name, type)
    