#!/usr/bin/env python3
# Copyright (c) The ClusterLink Authors.
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

from demos.utils.common import printHeader
from demos.utils.kind import Cluster
############################### MAIN ##########################
if __name__ == "__main__":
    srcSvc1        = "firefox"
    srcSvc2        = "firefox2"
    destSvc        = "openspeedtest"
    cl1            = Cluster(name="peer1")
    cl2            = Cluster(name="peer2")
    cl3            = Cluster(name="peer3")

    namespace = "default"


    print(f'Working directory {projDir}')
    os.chdir(projDir)

    ###get gw parameters
    cl1.useCluster()
    cl1.setKindIp()
    cl1.set_kube_config()
    cl2.useCluster()
    cl2.set_kube_config()
    cl3.useCluster()
    cl3.setKindIp()
    cl3.set_kube_config()

    #Import service
    printHeader(f"\n\nStart import svc {destSvc}")
    cl1.imports.create(destSvc,namespace,3000,cl2.name,destSvc,namespace)
    cl3.imports.create(destSvc,namespace,3000,cl2.name,destSvc,namespace)

    #Add policy
    printHeader("Applying policies")
    cl1.policies.create(name="allow-all",namespace=namespace , action="allow", from_attribute=[{"workloadSelector": {}}],to_attribute=[{"workloadSelector": {}}])
    cl2.policies.create(name="allow-all",namespace=namespace, action="allow", from_attribute=[{"workloadSelector": {}}],to_attribute=[{"workloadSelector": {}}])
    cl3.policies.create(name="allow-all",namespace=namespace, action="allow", from_attribute=[{"workloadSelector": {}}],to_attribute=[{"workloadSelector": {}}])


    #Firefox communications
    printHeader("Firefox urls")
    print(f"To use the cl1 firefox client, run the command:\n    firefox http://{cl1.ip}:30000/")
    print(f"To use the first cl3 firefox client, run the command:\n    firefox http://{cl3.ip}:30000/")
    print(f"To use the second cl3 firefox client, run the command:\n   firefox http://{cl3.ip}:30001/")
    print(f"The OpenSpeedTest url: http://{destSvc}:3000/ ")


