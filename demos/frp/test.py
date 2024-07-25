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

################################################################
# Name: FRP demo that connect 3 clusters using FRP:
# Desc: create 3 kind clusters :
# 1) GW, iPerf3 client, FRP client, and FRP server
# 2) GW, iPerf3 server, and FRP client
# 3) GW, iPerf3 client, and FRP client
###############################################################
import os
import sys
import time
import hashlib
projDir = os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__))))

sys.path.insert(0,f'{projDir}')

from demos.utils.common import  printHeader, runcmd
from demos.utils.kind import Cluster as KindCluster
from demos.utils.common import printHeader

# Folders
folCl=f"{projDir}/demos/iperf3/testdata/manifests/iperf3-client"
folSv=f"{projDir}/demos/iperf3/testdata/manifests/iperf3-server"
folFrp=f"{projDir}/demos/frp/testdata/manifests/"

# Services
srcSvc    = "iperf3-client"
destSvc   = "iperf3-server"
destPort  = 5000
namespace = "default"
frpNamespace= "frp"

def frpTest(cl1:KindCluster, cl2:KindCluster,cl3:KindCluster, testOutputFolder):
    print(f'Working directory {projDir}')
    os.chdir(projDir)

    # Build docker environment
    printHeader("Build docker image")
    os.system("make docker-build")
    os.system("make install")

    # Create Kind clusters environment
    cl1.createCluster(runBg=True)
    cl3.createCluster(runBg=True)
    cl2.createCluster(runBg=False)

    # Start Kind clusters environment
    cl1.create_fabric(testOutputFolder)
    cl1.startCluster(testOutputFolder)
    cl2.startCluster(testOutputFolder)
    cl3.startCluster(testOutputFolder)

    # Create iPerf3 micro-services
    cl1.loadService(srcSvc, "taoyou/iperf3-alpine",f"{folCl}/iperf3-client.yaml" )
    cl2.loadService(destSvc, "taoyou/iperf3-alpine",f"{folSv}/iperf3.yaml" )
    cl3.loadService(srcSvc, "taoyou/iperf3-alpine",f"{folCl}/iperf3-client.yaml" )
    os.environ['FRP_SERVER_IP'] = cl1.ip
    os.environ['FRP_SECRET_KEY'] = hashlib.sha256(str(time.time_ns()).encode()).hexdigest()[:10]
    # Use envsubst to replace the placeholder and apply the ConfigMap
    cl1.useCluster()
    runcmd(f"kubectl apply -f  {folFrp}/frp-ns.yaml")
    runcmd(f"kubectl apply -f {folFrp}/server/frps-configmap.yaml")
    cl1.loadService("frps", "snowdreamtech/frps",f"{folFrp}/server/frps.yaml", frpNamespace)

    # Create peers
    printHeader("Create peers")
    cl1.useCluster()
    runcmd(f"envsubst < {folFrp}/client/peer1/frpc-configmap.yaml| kubectl apply -f -")
    cl1.loadService("frpc", "snowdreamtech/frpc",f"{folFrp}/client/frpc.yaml",frpNamespace )
    runcmd(f"kubectl apply -f  {folFrp}/client/peer1/peer.yaml")
    cl2.useCluster()
    runcmd(f"kubectl apply -f  {folFrp}/frp-ns.yaml")
    runcmd(f"envsubst < {folFrp}/client/peer2/frpc-configmap.yaml| kubectl apply -f -")
    cl2.loadService("frpc", "snowdreamtech/frpc",f"{folFrp}/client/frpc.yaml",frpNamespace )
    runcmd(f"kubectl apply -f  {folFrp}/client/frpc.yaml")
    runcmd(f"kubectl apply -f  {folFrp}/client/peer2/peer.yaml")
    cl3.useCluster()
    runcmd(f"kubectl apply -f  {folFrp}/frp-ns.yaml")
    runcmd(f"envsubst < {folFrp}/client/peer3/frpc-configmap.yaml| kubectl apply -f -")
    cl3.loadService("frpc", "snowdreamtech/frpc",f"{folFrp}/client/frpc.yaml" ,frpNamespace)
    runcmd(f"kubectl apply -f  {folFrp}/client/peer3/peer.yaml")
    # Create exports
    cl2.exports.create(destSvc, namespace, destPort)

    # Import destination service
    printHeader(f"\n\nStart Importing {destSvc} service to {cl1.name}")
    cl1.imports.create(destSvc,namespace,destPort,cl2.name,destSvc,namespace)
    cl3.imports.create(destSvc,namespace,destPort,cl2.name,destSvc,namespace)

    # Add policy
    printHeader("Applying policies")
    cl1.policies.create(name="allow-all",namespace=namespace, action="allow", from_attribute=[{"workloadSelector": {}}],to_attribute=[{"workloadSelector": {}}])
    cl2.policies.create(name="allow-all",namespace=namespace, action="allow", from_attribute=[{"workloadSelector": {}}],to_attribute=[{"workloadSelector": {}}])
    cl3.policies.create(name="allow-all",namespace=namespace, action="allow", from_attribute=[{"workloadSelector": {}}],to_attribute=[{"workloadSelector": {}}])



