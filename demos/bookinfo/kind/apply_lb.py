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

srcSvc1  = "productpage"
srcSvc2  = "productpage2"
destSvc  = "reviews"
    
    
def applyPolicy(mbg, gwctlName, type):
    useKindCluster(mbg)
    gwctlPod=getPodName("gwctl")
    if type == "ecmp":
        printHeader(f"Set Ecmp poilicy")          
        runcmd(f'kubectl exec -i {gwctlPod} -- ./gwctl create policy --type lb --serviceDst {destSvc} --gwDest mbg2 --policy ecmp')
    elif type == "same":
        printHeader(f"Set same policy to all services")          
        runcmd(f'kubectl exec -i {gwctlPod} -- ./gwctl create policy  --type lb --serviceDst {destSvc} --gwDest mbg2 --policy static')
    elif type == "diff":
        runcmd(f'kubectl exec -i {gwctlPod} -- ./gwctl create policy --type lb --serviceSrc {srcSvc1} --serviceDst {destSvc} --gwDest mbg2 --policy static')
        runcmd(f'kubectl exec -i {gwctlPod} -- ./gwctl create policy --type lb --serviceSrc {srcSvc2} --serviceDst {destSvc} --gwDest mbg3 --policy static')
    elif type == "show":
        runcmd(f'kubectl exec -i {gwctlPod} -- ./gwctl get policy --myid {gwctlName}')
    elif type == "clean":
        runcmd(f'kubectl exec -i {gwctlPod} -- ./gwctl delete policy --type lb --serviceSrc {srcSvc2} --serviceDst {destSvc} ')
        runcmd(f'kubectl exec -i {gwctlPod} -- ./gwctl delete policy --type lb --serviceSrc {srcSvc1} --serviceDst {destSvc} ')
        runcmd(f'kubectl exec -i {gwctlPod} -- ./gwctl delete policy --type lb --serviceDst {destSvc}')



from demos.utils.mbgAux import runcmd, runcmdb, printHeader, getPodName
from demos.utils.kind.kindAux import useKindCluster

############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-m','--mbg', help='Either mbg1/mbg2/mbg3', required=False, default="mbg1")
    parser.add_argument('-t','--type', help='Either ecmp/same/diff/clean/show', required=False, default="ecmp")

    args = vars(parser.parse_args())

    mbg = args["mbg"]
    type = args["type"]
    gwctlName     = "gwctl"+ mbg[-1]
    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)

    applyPolicy(mbg, gwctlName, type)
    