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
import time
import subprocess as sp
from colorama import Fore
from colorama import Style

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
folMfst=f"{proj_dir}/config/manifests"

def waitPod(name, namespace="default"):
    time.sleep(2) #Initial start
    podStatus=""
    while("Running" not in podStatus):
        #cmd=f"kubectl get pods -l app={name} -o jsonpath" + "=\'{.items[0].status.containerStatuses[0].ready}\'"
        cmd=f"kubectl get pods -l app={name} -n {namespace} "+ '--no-headers -o custom-columns=":status.phase"'
        print(cmd)
        podStatus =sp.getoutput(cmd)
        if ("Running" not in podStatus):
            print (f"Waiting for pod {name} in namespace {namespace} to start current status: {podStatus}")
            time.sleep(7)
        else:
            time.sleep(5)
            break

def getPodNameIp(app):
    podName = getPodNameApp(app)
    podIp   =  getPodIp(podName)  
    return podName, podIp

def getPodNameApp(app):
    cmd=f"kubectl get pods -l app={app} "+'-o jsonpath="{.items[0].metadata.name}"'
    podName=sp.getoutput(cmd)
    return podName


def runcmd(cmd):
    print(f'{Fore.YELLOW}{cmd} {Style.RESET_ALL}')
    #sp.Popen(cmd,shell=True)
    os.system(cmd)
    
def runcmdb(cmd):
    print(f'{Fore.YELLOW}{cmd} {Style.RESET_ALL}')
    #sp.Popen(cmd,shell=True)
    os.system(cmd + ' &')
    time.sleep(7)

def printHeader(msg):
    print(f'{Fore.GREEN}{msg} {Style.RESET_ALL}')
    #print(msg)

def getMbgPorts(podMbg, destSvc):
    mbgJson =sp.getoutput(f' kubectl exec -i {podMbg} -- cat ./root/.mbg/mbgApp')
    mbgJson=json.loads(mbgJson)
    localPort =(mbgJson["Connections"][destSvc]["Local"]).split(":")[1]
    externalPort =(mbgJson["Connections"][destSvc]["External"]).split(":")[1]
    print(f"Service nodeport will use local Port: {localPort} and externalPort:{externalPort}")
    return localPort, externalPort

def buildMbg(name):
    runcmd(f"kubectl apply -f {folMfst}/mbg/mbg-role.yaml")
    runcmd(f"kubectl create -f {folMfst}/mbg/mbg.yaml")
    runcmd(f"kubectl create -f {folMfst}/mbg/dataplane.yaml")
    waitPod("mbg")
    podMbg, mbgIp= getPodNameIp("mbg")
    return podMbg, mbgIp

def buildMbgctl(name):
    runcmd(f"kubectl create -f {folMfst}/gwctl/gwctl.yaml")
    waitPod("gwctl")
    name,ip= getPodNameIp("gwctl")
    return name, ip 

#Creating k8s service for svc name
def createMbgK8sService(appName,svcName, namespace, port):
    podMbg= getPodName("mbg-deployment")        
    mbgLocalPort, _ = getMbgPorts(podMbg, appName)
    runcmd(f"kubectl delete service {svcName} -n {namespace}")
    runcmd(f"kubectl create service clusterip {svcName} -n {namespace} --tcp={port}:{mbgLocalPort}")
    runcmd(f"kubectl patch service {svcName} -n {namespace} -p "+  "\'{\"spec\":{\"selector\":{\"app\": \"mbg\"}}}\'") #replacing app name
    #runcmd(f"kubectl create endpoints {svcName} --namespace={namespace} --addreses=mbg.{mbgNS}.:{mbgLocalPort}")

def createK8sService(name, namespace, port, targetPort):
    runcmd(f"kubectl delete service {name} -n {namespace}")
    runcmd(f"kubectl create service clusterip {name} -n {namespace} --tcp={port}:{targetPort}")
    
def clean_cluster():
    runcmd(f'kubectl delete --all deployments')
    runcmd(f'kubectl delete --all svc')

class app:
    def __init__(self, name, namespace, host, port):  
        self.name       = name
        self.namespace  = namespace
        self.host     = host
        self.port       = port