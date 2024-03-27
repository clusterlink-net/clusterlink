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
projDir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{projDir}')

from demos.utils.common import runcmd, printHeader
from demos.utils.kind import cluster
from demos.utils.k8s import getPodNameIp

############################### MAIN ##########################
if __name__ == "__main__":
    srcSvc1        = "firefox"
    srcSvc2        = "firefox2"
    destSvc        = "openspeedtest"
    cl1            = cluster(name="peer1")
    cl2            = cluster(name="peer2")
    cl3            = cluster(name="peer3")
    allowAllPolicy = f"{projDir}/pkg/policyengine/examples/allowAll.json"
    
    print(f'Working directory {projDir}')
    os.chdir(projDir)

    ###get gw parameters
    cl1.useCluster()
    cl1.setKindIp()
    gwctl1Pod, gwctl1Ip = getPodNameIp("gwctl")
    cl2.useCluster()
    gwctl2Pod, gwctl2Ip = getPodNameIp("gwctl")
    cl3.useCluster()
    cl3.setKindIp()
    gwctl3Pod, gwctl3Ip = getPodNameIp("gwctl")

    #Import service
    printHeader(f"\n\nStart import svc {destSvc}")
    cl1.useCluster()    
    runcmd(f'gwctl create import --myid {cl1.name} --name {destSvc} --port 3000 --peer {cl2.name}')
    cl3.useCluster()     
    runcmd(f'gwctl create import --myid {cl3.name} --name {destSvc} --port 3000 --peer {cl2.name}')
    
    printHeader("\n\nStart get service cl1")
    runcmd(f'gwctl get import --myid {cl1.name} ')
    printHeader("\n\nStart get service cl3")
    runcmd(f'gwctl get import --myid {cl3.name} ')

    #Add policy
    printHeader("Applying policies")
    runcmd(f'gwctl --myid {cl1.name} create policy --type access --policyFile {allowAllPolicy}')
    runcmd(f'gwctl --myid {cl2.name} create policy --type access --policyFile {allowAllPolicy}')
    runcmd(f'gwctl --myid {cl3.name} create policy --type access --policyFile {allowAllPolicy}')
    
    #Firefox communications
    printHeader("Firefox urls")
    print(f"To use the cl1 firefox client, run the command:\n    firefox http://{cl1.ip}:30000/")
    print(f"To use the first cl3 firefox client, run the command:\n    firefox http://{cl3.ip}:30000/")
    print(f"To use the second cl3 firefox client, run the command:\n   firefox http://{cl3.ip}:30001/")    
    print(f"The OpenSpeedTest url: http://{destSvc}:3000/ ")


