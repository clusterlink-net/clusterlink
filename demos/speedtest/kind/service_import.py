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
proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

from demos.utils.common import runcmd, printHeader
from demos.utils.kind import useKindCluster, getKindIp
from demos.utils.k8s import getPodNameIp

############################### MAIN ##########################
if __name__ == "__main__":
    allowAllPolicy =f"{proj_dir}/pkg/policyengine/policytypes/examples/allowAll.json"
    srcSvc1         = "firefox"
    srcSvc2         = "firefox2"
    destSvc         = "openspeedtest"
    gw1Name        = "peer1"
    gw2Name        = "peer2"
    gw3Name        = "peer3"

    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)

    ###get gw parameters
    gw1Ip                = getKindIp(gw1Name)
    gwctl1Pod, gwctl1Ip = getPodNameIp("gwctl")
    gw2Ip                = getKindIp(gw2Name)
    gwctl2Pod, gwctl2Ip = getPodNameIp("gwctl")
    gw3Ip                = getKindIp(gw3Name)
    gwctl3Pod, gwctl3Ip = getPodNameIp("gwctl")

    #Import service
    printHeader(f"\n\nStart import svc {destSvc}")
    useKindCluster(gw1Name)    
    runcmd(f'gwctl create import --myid {gw1Name} --name {destSvc} --host {destSvc} --port 3000')
    useKindCluster(gw3Name)    
    runcmd(f'gwctl create import --myid {gw3Name} --name {destSvc} --host {destSvc} --port 3000')
    #Set K8s network services
    printHeader(f"\n\nStart binding service {destSvc}")
    useKindCluster(gw1Name)
    runcmd(f'gwctl create binding --myid {gw1Name} --import {destSvc} --peer {gw2Name}')
    useKindCluster(gw3Name)
    runcmd(f'gwctl create binding --myid {gw3Name} --import {destSvc} --peer {gw2Name}')
    
    printHeader("\n\nStart get service GW1")
    runcmd(f'gwctl get import  --myid {gw1Name} ')
    printHeader("\n\nStart get service GW3")
    runcmd(f'gwctl get import  --myid {gw3Name} ')

    #Add policy
    printHeader("Applying policies")
    runcmd(f'gwctl --myid {gw1Name} create policy --type access --policyFile {allowAllPolicy}')
    runcmd(f'gwctl --myid {gw2Name} create policy --type access --policyFile {allowAllPolicy}')
    runcmd(f'gwctl --myid {gw3Name} create policy --type access --policyFile {allowAllPolicy}')
    
    #Firefox communications
    printHeader("Firefox urls")
    print(f"To use the gw1 firefox client, run the command:\n    firefox http://{gw1Ip}:30000/")
    print(f"To use the first gw3 firefox client, run the command:\n    firefox http://{gw3Ip}:30000/")
    print(f"To use the second gw3 firefox client, run the command:\n   firefox http://{gw3Ip}:30001/")    
    print(f"The OpenSpeedTest url: http://{destSvc}:3000/ ")


