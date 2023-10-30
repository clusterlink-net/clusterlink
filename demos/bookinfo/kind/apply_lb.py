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

import os
import sys
import argparse

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')
from demos.utils.common import runcmd, printHeader
from demos.utils.k8s import getPodName
from demos.utils.kind import useKindCluster

srcSvc1  = "productpage"
srcSvc2  = "productpage2"
destSvc  = "reviews"
    
def applyPolicy(name, type):
    useKindCluster(name)
    gwctlPod=getPodName("gwctl")
    if type == "ecmp":
        printHeader(f"Set Ecmp poilicy")          
        runcmd(f'kubectl exec -i {gwctlPod} -- gwctl create policy --type lb --serviceDst {destSvc} --gwDest peer2 --policy ecmp')
    elif type == "same":
        printHeader("Set same policy to all services")          
        runcmd(f'kubectl exec -i {gwctlPod} -- gwctl create policy  --type lb --serviceDst {destSvc} --gwDest peer2 --policy static')
    elif type == "diff":
        runcmd(f'kubectl exec -i {gwctlPod} -- gwctl create policy --type lb --serviceSrc {srcSvc1} --serviceDst {destSvc} --gwDest peer2 --policy static')
        runcmd(f'kubectl exec -i {gwctlPod} -- gwctl create policy --type lb --serviceSrc {srcSvc2} --serviceDst {destSvc} --gwDest peer3 --policy static')
    elif type == "show":
        runcmd(f'kubectl exec -i {gwctlPod} -- gwctl get policy ')
    elif type == "clean":
        runcmd(f'kubectl exec -i {gwctlPod} -- gwctl delete policy --type lb --serviceSrc {srcSvc2} --serviceDst {destSvc} ')
        runcmd(f'kubectl exec -i {gwctlPod} -- gwctl delete policy --type lb --serviceSrc {srcSvc1} --serviceDst {destSvc} ')
        runcmd(f'kubectl exec -i {gwctlPod} -- gwctl delete policy --type lb --serviceDst {destSvc}')

############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-p','--peer', help='Either peer1/peer2/peer3', required=False, default="peer1")
    parser.add_argument('-t','--type', help='Either ecmp/same/diff/clean/show', required=False, default="ecmp")

    args = vars(parser.parse_args())

    peer = args["peer"]
    type = args["type"]
    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)

    applyPolicy(peer, type)
    