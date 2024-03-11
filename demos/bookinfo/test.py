#!/usr/bin/env python3
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

##############################################################################################
# Name: Bookinfo
# Info: support bookinfo application with gwctl inside the clusters 
#       In this we create three kind clusters
#       1) cluster1- contain gw, gwctl,product and details microservices (bookinfo services)
#       2) cluster2- contain gw, gwctl, review-v2 and rating microservices (bookinfo services)
#       3) cluster3- contain gw, gwctl, review-v3 and rating microservices (bookinfo services)
##############################################################################################

import os
import sys
projDir = os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__))))
sys.path.insert(0,f'{projDir}')

from demos.utils.common import runcmd, createFabric, printHeader,applyPeer
from demos.utils.kind import cluster
from demos.utils.k8s import getPodName,getPodIp

   
folpdct   = f"{projDir}/demos/bookinfo/manifests/product/"
folReview = f"{projDir}/demos/bookinfo/manifests/review"
allowAllPolicy =f"{projDir}/pkg/policyengine/policytypes/examples/allowAll.json"

reviewSvc     = "reviews"
srcSvc1       = "productpage"
srcSvc2       = "productpage2"
srcK8sSvcPort = "9080"
review2pod    = "reviews-v2"
review3pod    = "reviews-v3"

# bookInfoDemo runs the bookinfo demo.
def bookInfoDemo(cl1:cluster, cl2:cluster, cl3:cluster, testOutputFolder,logLevel="info" ,dataplane="envoy"):    
    print(f'Working directory {projDir}')
    os.chdir(projDir)
    ### build docker environment 
    printHeader("Build docker image")
    runcmd("make docker-build")
    
    # Create Kind clusters environment 
    cl1.createCluster(runBg=True)        
    cl2.createCluster(runBg=True)
    cl3.createCluster(runBg=False)  

    # Start Kind clusters environment 
    createFabric(testOutputFolder) 
    cl1.startCluster(testOutputFolder, logLevel, dataplane)        
    cl2.startCluster(testOutputFolder, logLevel, dataplane)        
    cl3.startCluster(testOutputFolder, logLevel, dataplane)        
        
    # Get cl parameters
    cl1.useCluster()
    gwctl1Pod = getPodName("gwctl")
    cl2.useCluster()
    gwctl2Pod = getPodName("gwctl")
    cl3.useCluster()
    gwctl3Pod = getPodName("gwctl")

    # Set GW services
    cl1.useCluster()
    cl1.loadService(srcSvc1, "maistra/examples-bookinfo-productpage-v1",f"{folpdct}/product.yaml")
    cl1.loadService(srcSvc2, "maistra/examples-bookinfo-productpage-v1",f"{folpdct}/product2.yaml")
    cl1.loadService(srcSvc1, "maistra/examples-bookinfo-details-v1:0.12.0",f"{folpdct}/details.yaml")
    cl2.useCluster()
    cl2.loadService(reviewSvc, "maistra/examples-bookinfo-reviews-v2",f"{folReview}/review-v2.yaml")
    cl2.loadService("ratings", "maistra/examples-bookinfo-ratings-v1:0.12.0",f"{folReview}/rating.yaml")
    cl3.useCluster()
    cl3.loadService(reviewSvc, "maistra/examples-bookinfo-reviews-v3",f"{folReview}/review-v3.yaml")
    cl3.loadService("ratings", "maistra/examples-bookinfo-ratings-v1:0.12.0",f"{folReview}/rating.yaml")
    
    # Add GW Peers
    printHeader("Add cl2, cl3 peer to cl1")
    cl1.useCluster()
    runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl create peer --name {cl2.name} --host {cl2.ip} --port {cl2.port}')
    runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl create peer --name {cl3.name} --host {cl3.ip} --port {cl3.port}')
    printHeader("Add cl1 peer to cl2")
    cl2.useCluster()
    runcmd(f'kubectl exec -i {gwctl2Pod} -- gwctl create peer --name {cl1.name} --host {cl1.ip} --port {cl1.port}')
    cl3.useCluster()
    printHeader("Add cl3 peer to cl1")
    runcmd(f'kubectl exec -i {gwctl3Pod} -- gwctl create peer --name {cl1.name} --host {cl1.ip} --port {cl1.port}')

    # Set exports  
    cl1.useCluster()
    printHeader(f"create exports {srcSvc1} {srcSvc2}")
    runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl create export --name {srcSvc1} --host {srcSvc1} --port {srcK8sSvcPort}')
    runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl create export --name {srcSvc2} --host {srcSvc2} --port {srcK8sSvcPort}')

    cl2.useCluster()
    review2Ip = f"{getPodIp(reviewSvc)}"
    review2Port = f"{srcK8sSvcPort}"
    runcmd(f'kubectl exec -i {gwctl2Pod} -- gwctl create export --name {reviewSvc} --host {review2Ip} --port {review2Port}')
    

    cl3.useCluster()
    review3Ip = f"{getPodIp(reviewSvc)}"
    review3Port = f"{srcK8sSvcPort}"
    runcmd(f'kubectl exec -i {gwctl3Pod} -- gwctl create export --name {reviewSvc} --host {review3Ip} --port {review3Port}')

    # Import service
    cl1.useCluster()
    printHeader(f"\n\nStart import svc {reviewSvc}")
    runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl create import --name {reviewSvc} --port {srcK8sSvcPort} --peer {cl2.name} --peer {cl3.name}')
    
    # Get services
    cl1.useCluster()
    printHeader("\n\nStart get service")
    runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl get import')
    runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl get policy')
    
    # Set policies
    printHeader(f"\n\nApplying policy file {allowAllPolicy}")
    policyFile ="/tmp/allowAll.json"
    cl1.useCluster()
    runcmd(f'kubectl cp {allowAllPolicy} gwctl:{policyFile}')
    runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl create policy --type access --policyFile {policyFile}')
    cl2.useCluster()
    runcmd(f'kubectl cp {allowAllPolicy} gwctl:{policyFile}')
    runcmd(f'kubectl exec -i {gwctl2Pod} -- gwctl create policy --type access --policyFile {policyFile}')
    cl3.useCluster()
    runcmd(f'kubectl cp {allowAllPolicy} gwctl:{policyFile}')
    runcmd(f'kubectl exec -i {gwctl3Pod} -- gwctl create policy --type access --policyFile {policyFile}')

# applyPolicy apply policy for the bookInfo demo
def applyPolicy(cl:cluster, type):
    cl.useCluster()
    gwctlPod=getPodName("gwctl")
    if type == "ecmp":
        printHeader("Set ECMP poilicy")
        runcmd(f'kubectl exec -i {gwctlPod} -- gwctl create policy --type lb --serviceDst {reviewSvc} --gwDest peer2 --policy ecmp')
    elif type == "same":
        printHeader("Set same policy to all services")          
        runcmd(f'kubectl exec -i {gwctlPod} -- gwctl create policy  --type lb --serviceDst {reviewSvc} --gwDest peer2 --policy static')
    elif type == "diff":
        runcmd(f'kubectl exec -i {gwctlPod} -- gwctl create policy --type lb --serviceSrc {srcSvc1} --serviceDst {reviewSvc} --gwDest peer2 --policy static')
        runcmd(f'kubectl exec -i {gwctlPod} -- gwctl create policy --type lb --serviceSrc {srcSvc2} --serviceDst {reviewSvc} --gwDest peer3 --policy static')
    elif type == "show":
        runcmd(f'kubectl exec -i {gwctlPod} -- gwctl get policy ')
    elif type == "clean":
        runcmd(f'kubectl exec -i {gwctlPod} -- gwctl delete policy --type lb --serviceSrc {srcSvc2} --serviceDst {reviewSvc} ')
        runcmd(f'kubectl exec -i {gwctlPod} -- gwctl delete policy --type lb --serviceSrc {srcSvc1} --serviceDst {reviewSvc} ')
        runcmd(f'kubectl exec -i {gwctlPod} -- gwctl delete policy --type lb --serviceDst {reviewSvc}')

def applyFailover(cl:cluster, type, testOutputFolder):
    cl.useCluster()
    clPod=getPodName("cl-dataplane")
    print(clPod)
    if type == "fail":
        printHeader(f"Failing {cl.name} dataplane")
        runcmd("kubectl delete deployment cl-dataplane")
    elif type == "start":
        printHeader(f"Restoring {cl.name} dataplane")
        applyPeer(cl.name,testOutputFolder)

