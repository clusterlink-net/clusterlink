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

import subprocess as sp
import time
import json

# getPodNameIp gets the application pod's name and IP.
def getPodNameIp(app):
    podName = getPodName(app)
    podIp   =  getPodIp(podName)
    return podName, podIp

# getPodName gets the application pod's name.
def getPodName(app, namespace="default"):
    cmd=f"kubectl get pods -l app={app} -n {namespace} "+'-o jsonpath="{.items[0].metadata.name}"'
    podName=sp.getoutput(cmd)
    return podName

#getPodNameIp gets the application pod's name.
def getPodIp(name):
    name=getPodName(name)
    podIp=sp.getoutput(f"kubectl get pod {name}"+" --template '{{.status.podIP}}'")
    return podIp

#createK8sService creates k8s service.
def createK8sService(name, namespace, port, targetPort):
    sp.getoutput(f"kubectl delete service {name} -n {namespace}")
    sp.getoutput(f"kubectl create service clusterip {name} -n {namespace} --tcp={port}:{targetPort}")

# waitPod waits until pod starts
def waitPod(name, namespace="default"):
    time.sleep(2) #Initial start
    podStatus=""
    while("Running" not in podStatus):
        cmd=f"kubectl get pods -l app={name} -n {namespace} "+ '--no-headers -o custom-columns=":status.phase"'
        print(cmd)
        podStatus =sp.getoutput(cmd)
        if ("Running" not in podStatus):
            print (f"Waiting for pod {name} in namespace {namespace} to start current status: {podStatus}")
            time.sleep(7)
        else:
            time.sleep(5)
            break
# getNodeIp returns the ip of node the worker is running
def getNodeIP(num=0):
    clJson=json.loads(sp.getoutput('kubectl get nodes -o json'))
    ip = clJson["items"][0]["status"]["addresses"][num]["address"]
    return ip

# cleanCluster removes all deployments and services
def cleanCluster():
    sp.getoutput('kubectl delete instances.clusterlink.net --all -A')
    sp.getoutput('kubectl delete accesspolicies.clusterlink.net --all -A')
    sp.getoutput('kubectl delete privilegedaccesspolicies.clusterlink.net')
    sp.getoutput('kubectl delete imports.clusterlink.net --all -A')
    sp.getoutput('kubectl delete exports.clusterlink.net --all -A')
    sp.getoutput('kubectl delete peers.clusterlink.net --all -A')
    sp.getoutput('kubectl delete --all secrets -A')
    sp.getoutput('kubectl delete --all deployments -n clusterlink-operator')
