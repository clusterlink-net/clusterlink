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

from demos.utils.mbgAux import runcmd, runcmdb, printHeader, waitPod, getPodName, getMbgPorts,buildMbg,buildMbgctl,getPodIp
from demos.utils.kind.kindAux import useKindCluster,getKindIp

iperf3DestPort  = "30001"
#folders
folCl=f"{proj_dir}/demos/iperf3/testdata/manifests/iperf3-client"
folSv=f"{proj_dir}/demos/iperf3/testdata/manifests/iperf3-server"
    
def setIperf3client(mbgName, gwctlName,srcSvc):
    printHeader(f"Create {srcSvc} (client) service in {mbgName}")
    useKindCluster(mbgName)
    runcmd(f"kind load docker-image mlabbe/iperf3 --name={mbgName}")
    runcmd(f"kubectl create -f {folCl}/{srcSvc}.yaml")
    waitPod(srcSvc)
    runcmd(f'gwctl create export --myid {gwctlName} --name {srcSvc} --host {srcSvc} --port {5000}')

def setIperf3Server(mbgName, gwctlName, destSvc):
    printHeader(f"Add {destSvc} (server) service in {mbgName}")
    useKindCluster(mbgName)
    runcmd(f"kind load docker-image mlabbe/iperf3 --name={mbgName}")
    runcmd(f"kubectl create -f {folSv}/iperf3.yaml")
    waitPod(destSvc)
    runcmd(f"kubectl create service nodeport iperf3-server --tcp=5000:5000 --node-port={iperf3DestPort}")
    destSvcPort = f"5000"
    gwctlPod =getPodName("gwctl")
    destSvcIp  = "iperf3-server"
    runcmd(f'gwctl --myid {gwctlName} create export --name {destSvc} --host {destSvcIp} --port {destSvcPort}')

############################### MAIN ##########################
if __name__ == "__main__":
    #parameters 
    mbg1Name    = "mbg1"
    gwctl1Name = "gwctl2"
    srcSvc      = "iperf3-client"
    srcSvc2     = "iperf3-client2"
    mbg2Name    = "mbg2"
    gwctl2Name = "gwctl2"
    destSvc     = "iperf3-server"
    mbg3Name    = "mbg3"
    gwctl3Name = "gwctl3"
    destPort        = "5000"

    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)
    
    # Set iperf3-client in MBG1
    setIperf3client(mbg1Name, gwctl1Name, srcSvc)
    
    #Set iperf3-service in MBG2
    setIperf3Server(mbg2Name,gwctl2Name, destSvc)

    # Set iperf3-client in MBG3
    setIperf3client(mbg3Name, gwctl3Name, srcSvc)
    setIperf3client(mbg3Name, gwctl3Name, srcSvc2)
    