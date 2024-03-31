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
import shutil
import subprocess as sp
from colorama import Fore
from colorama import Style
from demos.utils.k8s import waitPod

ProjDir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
CL_CLI    = ProjDir + "/bin/clusterlink "
folMfst=f"{ProjDir}/config/manifests"

# Init Functions
# createFabric creates fabric certificates using clusterlink
def createFabric(dir):
    createFolder(dir)
    runcmdDir(f"{CL_CLI} create fabric",dir)

# createGw creates peer certificates and yaml and deploys it to the cluster.
def createGw(name, dir, logLevel="info",dataplane="envoy",localImage=False):
    createPeer(name, dir, logLevel, dataplane,localImage)
    applyPeer(name, dir)

# createPeer creates peer certificates and yaml
def createPeer(name, dir, logLevel="info", dataplane="envoy",localImage=False):
    flag = "--container-registry=""" if localImage else ""
    runcmdDir(f"{CL_CLI} create peer-cert --name {name} --log-level {logLevel} --dataplane-type {dataplane} {flag} --namespace default",dir)

# applyPeer deploys the peer certificates and yaml to the cluster.
def applyPeer(name,dir):
    runcmd(f"kubectl apply -f {dir}/default_fabric/{name}/k8s.yaml")
    waitPod("cl-controlplane")
    waitPod("cl-dataplane")
    waitPod("gwctl")

# startGwctl sets gwctl configuration
def startGwctl(name,geIP, gwPort, testOutputFolder):
    runcmd(f'gwctl init --id {name} --gwIP {geIP} --gwPort {gwPort}  --dataplane mtls \
    --certca {testOutputFolder}/default_fabric/cert.pem --cert {testOutputFolder}/default_fabric/{name}/gwctl/cert.pem --key {testOutputFolder}/default_fabric/{name}/gwctl/key.pem')

# Log Functions
# runcmd runs os system command.
def runcmd(cmd):
    print(f'{Fore.YELLOW}{cmd} {Style.RESET_ALL}')
    os.system(cmd)

# runcmdDir runs os system command in specific directory.
def runcmdDir(cmd,dir):
    print(f'{Fore.YELLOW}{cmd} {Style.RESET_ALL}')
    sp.run(cmd, shell=True, cwd=dir, check=True)

# runcmdb runs os system command in the background.
def runcmdb(cmd):
    print(f'{Fore.YELLOW}{cmd} {Style.RESET_ALL}')
    os.system(cmd + ' &')

# printHeader runs os system command in the background.
def printHeader(msg):
    print(f'{Fore.GREEN}{msg} {Style.RESET_ALL}')

# createFolder creates folder.
def createFolder(name):
    if os.path.exists(name):
        shutil.rmtree(name)
    os.makedirs(name)

# app cluster contains the application service information.
class app:
    def __init__(self, name, namespace, host, port):
        self.name      = name
        self.namespace = namespace
        self.host      = host
        self.port      = port