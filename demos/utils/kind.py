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

import json,os
import subprocess as sp
from demos.utils.manifests.kind.flannel.create_cni_bridge import createCniBridge,createKindCfgForflunnel
from demos.utils.common import runcmd, createGw, printHeader, ProjDir
from demos.utils.k8s import waitPod

# BuildKindCluster builds kind cluster.
def BuildKindCluster(name, cni="default", cfg="" ):
    #Set config file
    cfgFlag = f" --config {cfg}" if cfg != "" else  ""
    cfgFlag = f" --config {ProjDir}/demos/utils/manifests/kind/calico/calico-config.yaml" if (cfg == "" and cni== "calico")  else cfgFlag
    if  cni == "flannel" and cfg =="":
        cfgFlag = f" --config {ProjDir}/bin/plugins/flannel-config.yaml"
        createCniBridge()
        createKindCfgForflunnel()

    runcmd(f"kind create cluster  --name={name} {cfgFlag}")
    if  cni == "flannel":
        runcmd("kubectl apply -f https://raw.githubusercontent.com/flannel-io/flannel/master/Documentation/kube-flannel.yml")
    if  cni == "calico":
        runcmd("kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.25.0/manifests/tigera-operator.yaml")
        runcmd("kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.25.0/manifests/custom-resources.yaml")

    ip=getKindIp(name)
    return ip

# useKindCluster set the context for the input kind cluster.
def useKindCluster(name):
    os.system(f'kubectl config use-context kind-{name}')

# getKindIp get Kind cluster IP.
def getKindIp(name):
    useKindCluster(name)
    clJson=json.loads(sp.getoutput(f' kubectl get nodes -o json'))
    ip = clJson["items"][0]["status"]["addresses"][0]["address"]
    return ip

# loadService loads image to cluster, deploy it and wait until pod is ready.
def loadService(name, gwName, image, yaml, namespace="default"):
    printHeader(f"Create {name} (client) service in {gwName}")
    useKindCluster(gwName)
    runcmd(f"kind load docker-image {image} --name={gwName}")
    runcmd(f"kubectl create -f {yaml}")
    waitPod(name,namespace)

# startKindCluster build Kind cluster and deploy Clusterlink.
def startKindCluster(name, testOutputFolder, cni="default"):
    os.system(f"kind delete cluster --name={name}")
    printHeader(f"\n\nStart building {name}")
    BuildKindCluster(name,cni)
    runcmd(f"kind load docker-image cl-controlplane cl-dataplane cl-go-dataplane gwctl --name={name}")
    createGw(name,testOutputFolder)
    
    runcmd("kubectl delete service cl-dataplane")
    runcmd("kubectl create service nodeport cl-dataplane --tcp=443:443 --node-port=30443")