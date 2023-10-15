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

from demos.utils.mbgAux import runcmd, runcmdb, printHeader, waitPod, getPodName, getPodNameApp, getMbgPorts,getPodIp,clean_cluster,getPodNameIp

from demos.utils.cloud.check_k8s_cluster_ready import checkClusterIsReady,connectToCluster
from demos.utils.cloud.mbg_setup import mbgSetup,pushImage,mbgBuild
from demos.utils.cloud.create_k8s_cluster import createCluster
from demos.utils.cloud.clusterClass import cluster
from demos.utils.cloud.delete_k8s_cluster import deleteClustersList, cleanClustersList
from demos.utils.cloud.PROJECT_PARAMS import PROJECT_PATH
import argparse

srcSvc1  = "productpage"
srcSvc2  = "productpage2"
destSvc  = "reviews"
    

#MBG3 parameters
mbg1gcp = cluster(name="mbg1", zone = "us-west1-b"   , platform = "gcp", type = "host")   #Oregon
mbg1ibm = cluster(name="mbg1", zone = "sjc04"        , platform = "ibm", type = "host")   #San jose
mbg2gcp = cluster(name="mbg2", zone = "us-central1-b", platform = "gcp", type = "target") #Iowa
mbg2ibm = cluster(name="mbg2", zone = "dal10"        , platform = "ibm", type = "target") #Dallas
mbg3gcp = cluster(name="mbg3", zone = "us-east4-b"   , platform = "gcp", type = "target") #Virginia
mbg3ibm = cluster(name="mbg3", zone = "wdc04"        , platform = "ibm", type = "target") #Washington DC

mbg             = "mbg3"
mbg3DataPort    = "30001"
mbg3cPort       = "443"
mbg3cPortLocal  = "443"
mbg3crtFlags    = "--certca ./mtls/ca.crt --cert ./mtls/mbg3.crt --key ./mtls/mbg3.key"
mbg3Name        = "mbg3"
destSvc      = "reviews"
    

def exposeService(mbgName,destSvc):
    gwctlPod = getPodName("gwctl")
    printHeader(f"\n\nStart exposing {destSvc} service to {mbgName}")
    runcmd(f'kubectl exec -i {gwctlPod} -- ./gwctl expose --service {destSvc}')


def applyFail(mbg,type):
    connectToCluster(mbg)
    mPod=getPodName("mbg-")
    print(mPod)
    mbgIp=sp.getoutput('kubectl get svc -l app=mbg  -o jsonpath="{.items[0].status.loadBalancer.ingress[0].ip}"')
    print(mbgIp)

    if type == "fail":
        printHeader(f"Failing MBG")
        runcmd(f'kubectl exec -i {mPod} -- killall mbg')
    elif type == "start":
        printHeader(f"Starting up and Restoring MBG")
        runcmdb(f'kubectl exec -i {mPod} -- ./controlplane start --name "{mbg.name}" --ip {mbgIp} --cport {mbg3cPort} --cportLocal {mbg3cPortLocal}\
        --dataplane mtls {mbg3crtFlags} --startPolicyEngine {True} --restore {True}')
        time.sleep(2)
        exposeService(mbg,destSvc)


from demos.utils.mbgAux import runcmd, runcmdb, printHeader, getPodName
from demos.utils.kind.kindAux import useKindCluster

############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-t','--type', help='Either fail/start', required=False, default="fail")
    parser.add_argument('-cloud','--cloud', help='Cloud setup using gcp/ibm', required=False, default="ibm")

    args = vars(parser.parse_args())

    type = args["type"]
    mbg3 = mbg3gcp if args["cloud"] in ["gcp"]        else mbg3ibm

    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)
    
    applyFail(mbg3,type)
    
