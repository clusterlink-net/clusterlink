
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
import os
from demos.utils.k8s import  cleanCluster, waitPod, getNodeIP
from demos.utils.common import runcmd, printHeader
from demos.utils.clusterlink import ClusterLink, CLUSTELINK_NS


# cluster class represents a cloud cluster for deploying the ClusterLink gateway.
class Cluster(ClusterLink):
    def __init__(self, name, zone,platform, machineType="small",namespace=CLUSTELINK_NS):
        super().__init__(namespace)
        self.name        = name
        self.zone        = zone
        self.platform    = platform
        self.machineType = machineType
        self.port        = 443
        self.ip          = ""
        self.nodeIP      = "" # the IP of the worker node

    # createCluster creates a K8s cluster on cloud platform.
    def createCluster(self, runBg):
        print(f"create {self.name} cluster , zone {self.zone} , platform {self.platform}")
        bgFlag = " &" if runBg else ""
        if self.platform == "gcp":
            flags = "  --machine-type n2-standard-4" if self.machineType=="large" else "" #e2-medium
            cmd=f"gcloud container clusters create {self.name} --zone {self.zone} --num-nodes 1 --tags tcpall {flags} {bgFlag}"
            print(cmd)
            os.system(cmd)
        elif self.platform == "aws": #--instance-selector-vcpus 2  --instance-selector-memory 4 --instance-selector-cpu-architecture arm64
            cmd =f"eksctl create cluster --name {self.name} --region {self.zone} -N 1 {bgFlag}"
            print(cmd)
            os.system(cmd)

        elif self.platform == "ibm":
            vlanPrivateIp=sp.getoutput(f"ibmcloud ks vlans --zone {self.zone} |fgrep private |cut -d ' ' -f 1")
            vlanPublicIp=sp.getoutput(f"ibmcloud ks vlans --zone {self.zone}  |fgrep public |cut -d ' ' -f 1")
            print("vlanPublicIp:",vlanPublicIp)
            vlanPrivateString = "--private-vlan " + vlanPrivateIp  if (vlanPrivateIp != "" and "FAILED" not in vlanPrivateIp) else ""
            if (vlanPublicIp  != "" and "FAILED" not in vlanPublicIp):
                vlanPublicString  = "--public-vlan "  + vlanPublicIp
            else:
                vlanPublicString= ""
                vlanPrivateString = vlanPrivateString + " --private-only " if (vlanPrivateString != "") else ""

            cmd= f"ibmcloud ks cluster create  classic  --name {self.name} --zone={self.zone} --flavor u3c.2x4 --workers=1 {vlanPrivateString} {vlanPublicString} {bgFlag}"
            print(cmd)
            os.system(cmd)
        else:
            print ("ERROR: Cloud platform {} not supported".format(self.platform))

    # startCluster deploys Clusterlink into the cluster.
    def startCluster(self, testOutputFolder, logLevel="info",dataplane="envoy"):
        self.checkClusterIsReady()
        self.useCluster()
        super().set_kube_config()
        self.create_peer_cert(self.name,testOutputFolder)
        self.deploy_peer(self.name, testOutputFolder, logLevel, dataplane)
        self.waitToLoadBalancer()
        self.nodeIP = getNodeIP(num=1)

    # useCluster sets the context for the input kind cluster.
    def useCluster(self):
        print(f"\n CONNECT TO: {self.name} in zone: {self.zone} ,platform: {self.platform}\n")
        if self.platform == "gcp":
            PROJECT_ID=sp.getoutput("gcloud info --format='value(config.project)'")
            cmd=f"gcloud container clusters  get-credentials {self.name} --zone {self.zone} --project {PROJECT_ID}"
        elif self.platform == "aws":
            cmd=f"aws eks --region {self.zone} update-kubeconfig --name {self.name}"
        elif self.platform == "ibm":
            cmd=f"ibmcloud ks cluster config --cluster {self.name}"
        else:
            print (f"ERROR: Cloud platform {self.platform} not supported")
            exit(1)
        print(cmd)
        out=sp.getoutput(cmd)
        print(f"connection output: {out}")


    # checkClusterIsReady set the context for the input kind cluster.
    def checkClusterIsReady(self):
        while (True):
            print(f"\n Check cluster {self.name} in zone: {self.zone} ,platform: {self.platform} is ready.\n")
            if self.platform == "gcp":
                PROJECT_ID=sp.getoutput("gcloud info --format='value(config.project)'")
                cmd = f"gcloud container clusters describe {self.name} --zone {self.zone} --project {PROJECT_ID} --format='value(status)'"
            elif self.platform == "aws":
                cmd = f'aws eks --region {self.zone} describe-cluster --name {self.name} --query "cluster.status"'
            elif self.platform == "ibm":
                cmd = f"ibmcloud ks cluster get --cluster {self.name}"
            else:
                print (f"ERROR: Cloud platform {self.platform} not supported")
            print(cmd)
            out=sp.getoutput(cmd)
            if ("ACTIVE" in out  and  self.platform == "aws") or ("RUNNING" in out and  self.platform == "gcp") or \
               ("normal" in out and self.platform == "ibm"):
                break

            time.sleep(20)

        print(f"\n Cluster is ready: {self.name} in zone: {self.zone} ,platform: {self.platform}\n")

    # replace the container registry ip according to the proxy platform.
    def replaceSourceImage(self,yamlPath,imagePrefix):
        with open(yamlPath, 'r') as file:
            yamlContent = file.read()

        # Replace "image:" with the image prefix
        updatedYamlContent = yamlContent.replace("image: ", "image: " + imagePrefix)

        # Write the updated YAML content to a new file
        with open(yamlPath, 'w') as file:
            file.write(updatedYamlContent)

        print(f"Image prefixes have been added to the updated YAML file: {yamlPath}")


    # loadService deploys image and wait until the pod is ready.
    def loadService(self,name, image, yaml, namespace="default"):
        printHeader(f"Create {name} (client) service in {self.name}")
        self.useCluster()
        runcmd(f"kubectl create -f {yaml}")
        waitPod(name,namespace)

    # deleteCluster deletes the K8s cluster.
    def deleteCluster(self, runBg=False):
        bgFlag= "&" if runBg else ""
        print(f"Deleting cluster {self.name}")
        if self.platform == "gcp" :
            os.system(f"yes |gcloud container clusters delete {self.name} --zone {self.zone} {bgFlag}")
        elif self.platform == "aws":
            os.system(f"eksctl delete cluster --region {self.zone} --name {self.name} {bgFlag}")
        elif self.platform == "ibm":
            os.system(f"yes |ibmcloud ks cluster rm --force-delete-storage --cluster {self.name} {bgFlag}")
        else:
            print ("ERROR: Cloud platform {} not supported".format(self.platform))

    # cleanCluster clean all the services and deployment in the K8s cluster.
    def cleanCluster(self):
        print("Start clean cluster")
        self.useCluster()
        cleanCluster()

    # createLoadBalancer creates load-balancer to external access.
    def waitToLoadBalancer(self):
        gwHost=""
        while gwHost =="":
            print("Waiting for clusterlink loadbalncer ip...")
            if self.platform == "aws":
                gwHost=sp.getoutput(f'kubectl get svc clusterlink -n {self.namespace} ' +' -o jsonpath="{.status.loadBalancer.ingress[0].hostname}"')
            else:
                gwHost=sp.getoutput(f'kubectl get svc clusterlink -n {self.namespace} ' +' -o jsonpath="{.status.loadBalancer.ingress[0].ip}"')
            time.sleep(10)
        self.ip = gwHost
        return gwHost
