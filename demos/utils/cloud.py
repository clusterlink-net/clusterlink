
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
from demos.utils.common import runcmd , createPeer, applyPeer, printHeader

# cluster class represents a cloud cluster for deploying the ClusterLink gateway. 
class cluster:
    def __init__(self, name, zone,platform,machineType="small"):
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
            cmd =f"eksctl create cluster --name {self.name} --region {self.zone} -N 1  {flags}  {bgFlag}"
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
        createPeer(self.name,testOutputFolder, logLevel, dataplane)
        applyPeer(self.name,testOutputFolder)
        self.createLoadBalancer()
        self.nodeIP = getNodeIP(num=1)
    
    # useCluster sets the context for the input kind cluster.
    def useCluster(self):
        print(f"\n CONNECT TO: {self.name} in zone: {self.zone} ,platform: {self.platform}\n")
        connectFlag= False
        while (not connectFlag):
            if self.platform == "gcp":
                PROJECT_ID=sp.getoutput("gcloud info --format='value(config.project)'")
                cmd=f"gcloud container clusters  get-credentials {self.name} --zone {self.zone} --project {PROJECT_ID}"
                print(cmd)
            elif self.platform == "aws":
                cmd=f"aws eks --region {self.zone} update-kubeconfig --name {self.name}"
            elif self.platform == "ibm":
                cmd=f"ibmcloud ks cluster config --cluster {self.name}"
            else:
                print (f"ERROR: Cloud platform {self.platform} not supported")
                exit(1)
            
            out=sp.getoutput(cmd)
            print(f"connection output: {out}")
            connectFlag = False if ("ERROR" in out or "WARNING" in out or "Failed" in out) else True
            if not connectFlag: 
                time.sleep(30) #wait more time to connection
            return out
    
    # checkClusterIsReady set the context for the input kind cluster.
    def checkClusterIsReady(self):
        connectFlag= False
        while (not connectFlag):
            out=self.useCluster()
            connectFlag = False if ("ERROR" in out or "Failed" in out or "FAILED" in out) else True
            time.sleep(20)

        print(f"\n Cluster Ready: {self.name} in zone: {self.zone} ,platform: {self.platform}\n")
        
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
    def createLoadBalancer(self,port="443", externalIp=""):
        runcmd(f"kubectl expose deployment cl-dataplane --name=cl-dataplane-load-balancer --port={port} --target-port={port} --type=LoadBalancer")
        gwIp=""
        if externalIp !="":
            runcmd("kubectl patch svc cl-dataplane-load-balancer -p "+ "\'{\"spec\": {\"type\": \"LoadBalancer\", \"loadBalancerIP\": \""+ externalIp+ "\"}}\'")
            gwIp= externalIp
        while gwIp =="":
            print("Waiting for cl-dataplane ip...")
            gwIp=sp.getoutput('kubectl get svc -l app=cl-dataplane  -o jsonpath="{.items[0].status.loadBalancer.ingress[0].ip}"')
            time.sleep(10)
        self.ip = gwIp
        return gwIp
