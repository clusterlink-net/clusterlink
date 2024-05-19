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

# Copyright (c) 2022 The ClusterLink Authors.
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

# Copyright (C) The ClusterLink Authors.
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

################################################################
# Name: Simple iperf3  test
# Desc: create 2 kind clusters :
# 1) GW and iPerf3 client
# 2) GW and iPerf3 server
###############################################################
import os
import sys

projDir = os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__))))
sys.path.insert(0,f'{projDir}')

from demos.utils.common import  printHeader
from demos.utils.kind import Cluster

# Folders
folCl=f"{projDir}/demos/iperf3/testdata/manifests/iperf3-client"
folSv=f"{projDir}/demos/iperf3/testdata/manifests/iperf3-server"

#services
srcSvc    = "iperf3-client"
destSvc   = "iperf3-server"
destPort  = 5000
namespace = "default"
# iperf3Test setup two cluster for creating iPerf3 test.
def iperf3Test(cl1:Cluster, cl2:Cluster, testOutputFolder,logLevel="info" ,dataplane="envoy"):
    print(f'Working directory {projDir}')
    os.chdir(projDir)

    # build docker environment
    printHeader("Build docker image")
    os.system("make docker-build")
    os.system("make install")

    # Create Kind clusters environment
    cl1.createCluster(runBg=True)
    cl2.createCluster(runBg=False)

    # Start Kind clusters environment
    cl1.create_fabric(testOutputFolder)
    cl1.startCluster(testOutputFolder,logLevel, dataplane)
    cl2.startCluster(testOutputFolder,logLevel, dataplane)

    # Create iPerf3 micto-services
    cl1.loadService(srcSvc, "mlabbe/iperf3",f"{folCl}/iperf3-client.yaml" )
    cl2.loadService(destSvc, "mlabbe/iperf3",f"{folSv}/iperf3.yaml" )

    # Create peers
    printHeader("Create peers")
    cl1.peers.create(cl2.name, cl2.ip, cl2.port)
    cl2.peers.create(cl1.name, cl1.ip, cl1.port)
    # Create exports
    cl2.exports.create(destSvc, namespace, destPort)

    #Import destination service
    printHeader(f"\n\nStart Importing {destSvc} service to {cl1.name}")
    cl1.imports.create(destSvc,namespace,destPort,cl2.name,destSvc,namespace)

    #Add policy
    printHeader("Applying policies")
    cl1.policies.create(name="allow-all",namespace=namespace, action="allow", from_attribute=[{"workloadSelector": {}}],to_attribute=[{"workloadSelector": {}}])
    cl2.policies.create(name="allow-all",namespace=namespace, action="allow", from_attribute=[{"workloadSelector": {}}],to_attribute=[{"workloadSelector": {}}])
