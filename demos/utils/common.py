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
clAdm    = ProjDir + "/bin/cl-adm "
folMfst=f"{ProjDir}/config/manifests"

# Init Functions
# createFabric creates fabric certificates using cl-adm
def createFabric(dir):
    createFolder(dir)
    runcmdDir(f"{clAdm} create fabric",dir)

# createGw creates peer certificates and yaml and deploys it to the cluster. 
def createGw(name, dir, logLevel="info",dataplane="envoy",localImage=False):
    createSite(name, dir, logLevel, dataplane,localImage)
    applySite(name, dir)

# createSite creates site certificates and yaml
def createSite(name, dir, logLevel="info", dataplane="envoy",localImage=False):
    flag = "--container-registry=""" if localImage else ""
    runcmdDir(f"{clAdm} create site --name {name} --log-level {logLevel} --dataplane-type {dataplane} {flag} --namespace default",dir)
    
# applySite deploys the site certificates and yaml to the cluster. 
def applySite(name,dir):
    runcmd(f"kubectl apply -f {dir}/{name}/k8s.yaml")
    waitPod("cl-controlplane")
    waitPod("cl-dataplane")
    waitPod("gwctl")
    
# startGwctl sets gwctl configuration
def startGwctl(name,geIP, gwPort, testOutputFolder):
    runcmd(f'gwctl init --id {name} --gwIP {geIP} --gwPort {gwPort}  --dataplane mtls \
    --certca {testOutputFolder}/cert.pem --cert {testOutputFolder}/{name}/gwctl/cert.pem --key {testOutputFolder}/{name}/gwctl/key.pem') 

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