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
from demos.utils.manifests.kind.flannel.create_cni_bridge import createCniBridge,createKindCfgForflunnel
from demos.utils.common import runcmd, createGw, printHeader, ProjDir
from demos.utils.k8s import waitPod, getNodeIP

# cluster class represents a kind cluster for deploying the ClusterLink gateway. 
class cluster:
    def __init__(self, name, cni="default", cfgFile=""):
        self.name    = name
        self.cni     = cni
        self.port    = 30443
        self.ip      = ""
        self.cfgFile = cfgFile
    
    # createCluster creates a kind cluster.
    def createCluster(self, runBg=False):
        os.system(f"kind delete cluster --name={self.name}")
        printHeader(f"\n\nStart building {self.name}")
        bgFlag = " &" if runBg and self.cni=="" else ""
        #Set config file
        cfgFlag = f" --config {self.cfgFile}" if self.cfgFile != "" else  ""
        cfgFlag = f" --config {ProjDir}/demos/utils/manifests/kind/calico/calico-config.yaml" if (self.cfgFile == "" and self.cni== "calico")  else cfgFlag
        if  self.cni == "flannel" and self.cfgFile =="":
            cfgFlag = f" --config {ProjDir}/bin/plugins/flannel-config.yaml"
            createCniBridge()
            createKindCfgForflunnel()

        runcmd(f"kind create cluster  --name={self.name} {cfgFlag} {bgFlag}")
        if  self.cni == "flannel":
            runcmd("kubectl apply -f https://raw.githubusercontent.com/flannel-io/flannel/master/Documentation/kube-flannel.yml")
        if  self.cni == "calico":
            runcmd("kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.26.4/manifests/tigera-operator.yaml")
            runcmd("kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.26.4/manifests/custom-resources.yaml")

    # startCluster deploy a Clusterlink gateway.
    def startCluster(self, testOutputFolder, logLevel="info", dataplane="envoy"):
        self.useCluster()
        runcmd(f"kind load docker-image cl-controlplane --name={self.name}")
        runcmd(f"kind load docker-image cl-dataplane    --name={self.name}")
        runcmd(f"kind load docker-image cl-go-dataplane --name={self.name}")
        runcmd(f"kind load docker-image gwctl --name={self.name}")
        createGw(self.name, testOutputFolder, logLevel, dataplane, localImage=True)
        self.setKindIp()
        runcmd("kubectl delete service cl-dataplane")
        runcmd("kubectl create service nodeport cl-dataplane --tcp=443:443 --node-port=30443")

    # useCluster sets the context for the input kind cluster.
    def useCluster(self):
        os.system(f'kubectl config use-context kind-{self.name}')

    # getKindIp gets a Kind cluster IP.
    def setKindIp(self):
        self.ip = getNodeIP()

    # loadService loads image to cluster, deploy it and wait until pod is ready.
    def loadService(self,name, image, yaml, namespace="default"):
        printHeader(f"Create {name} (client) service in {self.name}")
        self.useCluster()
        runcmd(f"kind load docker-image {image} --name={self.name}")
        runcmd(f"kubectl create -f {yaml}")
        waitPod(name,namespace)
    
    # deleteCluster deletes the K8s cluster.
    def deleteCluster(self, runBg=False):
        bgFlag= "&" if runBg else ""
        os.system(f"kind delete cluster --name={self.name} {bgFlag}")
