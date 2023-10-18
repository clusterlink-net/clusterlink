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

from demos.utils.mbgAux import runcmd, runcmdb, printHeader, getPodName
from demos.utils.kind.kindAux import useKindCluster


def importService(mbgName,gwctlName,destSvc,destPort, peer):
    printHeader(f"\n\nStart Importing {destSvc} service to {mbgName}")
    runcmd(f'gwctl --myid {gwctlName} create import --name {destSvc} --host {destSvc} --port {destPort}')
    printHeader(f"\n\nStart binding {destSvc} service to {mbgName}")
    runcmd(f'gwctl --myid {gwctlName} create binding --import {destSvc} --peer {peer}')

############################### MAIN ##########################
if __name__ == "__main__":
    #parameters 
    mbg1Name    = "mbg1"
    gwctl1Name  = "gwctl1"
    mbg2Name    = "mbg2"
    gwctl2Name  = "gwctl2"
    destSvc     = "iperf3-server"
    destPort    = "5000"

        
    
    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)

    importService(mbg2Name, gwctl2Name, destSvc,destPort,mbg2Name)
    