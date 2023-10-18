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
from demos.utils.kind.kindAux import useKindCluster, getKindIp

# Add MBG Peer
def connectMbgs( gwctlName, peerName, peerIp, peercPort):
    runcmd(f'gwctl create peer --myid {gwctlName} --name {peerName} --host {peerIp} --port {peercPort}')
    
############################### MAIN ##########################
if __name__ == "__main__":
    
    #MBG1 parameters 
    mbg1cPort = "30443"
    mbg1Name  = "mbg1"
    
    #MBG2 parameters 
    mbg2cPort = "30443"
    mbg2Name  = "mbg2"
    gwctl2Name = "gwctl2"
        
    #MBG3 parameters 
    mbg3cPort = "30443"
    mbg3Name  = "mbg3"
        
    useKindCluster(mbg1Name)
    mbg1Ip=getKindIp(mbg1Name)
    useKindCluster(mbg3Name)
    mbg3Ip=getKindIp(mbg3Name)
    
    useKindCluster(mbg2Name)
    printHeader("Add MBG2, MBG3 peer to MBG1")
    connectMbgs(gwctl2Name, mbg1Name, mbg1Ip, mbg1cPort)
    connectMbgs(gwctl2Name, mbg3Name, mbg3Ip, mbg3cPort)
