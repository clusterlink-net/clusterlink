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
sys.path.insert(1,f'{proj_dir}/demos/utils/cloud/')

from demos.utils.mbgAux import runcmd,printHeader, getPodName
from demos.utils.cloud.check_k8s_cluster_ready import connectToCluster
from demos.utils.cloud.clusterClass import cluster

srcSvc1  = "productpage"
srcSvc2  = "productpage2"
destSvc  = "reviews"
mbglist = { "mbg1gcp" : cluster(name="mbg1", zone = "us-west1-b"    , platform = "gcp", type = "host"),   #Oregon
            "mbg1ibm" : cluster(name="mbg1", zone = "sjc04"         , platform = "ibm", type = "host"),   #San jose
            "mbg2gcp" : cluster(name="mbg2", zone = "us-central1-b" , platform = "gcp", type = "target"), #Iowa
            "mbg2ibm" : cluster(name="mbg2", zone = "dal10"         , platform = "ibm", type = "target"), #Dallas
            "mbg3gcp" : cluster(name="mbg3", zone = "us-east4-b"    , platform = "gcp", type = "target"), #Virginia
            "mbg3ibm" : cluster(name="mbg3", zone = "wdc04"         , platform = "ibm", type = "target")} #Washington DC
    
def applyPolicy(mbg, gwctlName, type):
    connectToCluster(mbg)
    gwctlPod=getPodName("gwctl")
    if type == "ecmp":
        printHeader(f"Set Ecmp poilicy")          
        runcmd(f'kubectl exec -i {gwctlPod} -- ./gwctl create policy --type lb --serviceDst {destSvc}  --policy ecmp')
    elif type == "same":
        printHeader(f"Set same policy to all services")          
        runcmd(f'kubectl exec -i {gwctlPod} -- ./gwctl create policy --type lb --serviceDst {destSvc} --gwDest mbg2 --policy static')
    elif type == "diff":
        runcmd(f'kubectl exec -i {gwctlPod} -- ./gwctl create policy --type lb --serviceSrc {srcSvc1} --serviceDst {destSvc} --gwDest mbg2 --policy static')
        runcmd(f'kubectl exec -i {gwctlPod} -- ./gwctl create policy --type lb --serviceSrc {srcSvc2} --serviceDst {destSvc} --gwDest mbg3 --policy static')
    elif type == "show":
        runcmd(f'kubectl exec -i {gwctlPod} -- ./gwctl get policy --myid {gwctlName}')
    elif type == "clean":
        runcmd(f'kubectl exec -i {gwctlPod} -- ./gwctl delete policy --type lb --serviceSrc {srcSvc2} --serviceDst {destSvc} ')
        runcmd(f'kubectl exec -i {gwctlPod} -- ./gwctl delete policy --type lb --serviceSrc {srcSvc1} --serviceDst {destSvc} ')
        runcmd(f'kubectl exec -i {gwctlPod} -- ./gwctl delete policy --type lb --serviceDst {destSvc}')





############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-m','--mbg', help='Either mbg1/mbg2/mbg3', required=False, default="mbg1")
    parser.add_argument('-t','--type', help='Either ecmp/same/diff/show', required=False, default="ecmp")
    parser.add_argument('-cloud','--cloud', help='Cloud setup using gcp/ibm', required=False, default="gcp")

    args = vars(parser.parse_args())

    mbg = mbglist[args["mbg"] + args["cloud"]]
    type = args["type"]
    gwctlName     = mbg.name[:-1]+"ctl"+ mbg.name[-1]

    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)

    applyPolicy(mbg, gwctlName,type)
    