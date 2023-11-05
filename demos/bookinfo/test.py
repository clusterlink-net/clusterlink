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

from demos.utils.common import runcmd, createFabric, printHeader
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
def bookInfoDemo(gw1:cluster, gw2:cluster, gw3:cluster, testOutputFolder,logLevel="info" ,dataplane="envoy"):    
    print(f'Working directory {projDir}')
    os.chdir(projDir)
    ### build docker environment 
    printHeader("Build docker image")
    runcmd("make docker-build")
    
    # Create Kind clusters environment 
    gw1.createCluster(runBg=True)        
    gw2.createCluster(runBg=True)
    gw3.createCluster(runBg=False)  

    # Start Kind clusters environment 
    createFabric(testOutputFolder) 
    gw1.startCluster(testOutputFolder, logLevel, dataplane)        
    gw2.startCluster(testOutputFolder, logLevel, dataplane)        
    gw3.startCluster(testOutputFolder, logLevel, dataplane)        
        
    # Get gw parameters
    gw1.useCluster()
    gwctl1Pod = getPodName("gwctl")
    gw2.useCluster()
    gwctl2Pod = getPodName("gwctl")
    gw3.useCluster()
    gwctl3Pod = getPodName("gwctl")

    # Set GW services
    gw1.useCluster()
    gw1.loadService(srcSvc1, "maistra/examples-bookinfo-productpage-v1",f"{folpdct}/product.yaml")
    gw1.loadService(srcSvc2, "maistra/examples-bookinfo-productpage-v1",f"{folpdct}/product2.yaml")
    gw1.loadService(srcSvc1, "maistra/examples-bookinfo-details-v1:0.12.0",f"{folpdct}/details.yaml")
    gw2.useCluster()
    gw2.loadService(reviewSvc, "maistra/examples-bookinfo-reviews-v2",f"{folReview}/review-v2.yaml")
    gw2.loadService("ratings", "maistra/examples-bookinfo-ratings-v1:0.12.0",f"{folReview}/rating.yaml")
    gw3.useCluster()
    gw3.loadService(reviewSvc, "maistra/examples-bookinfo-reviews-v3",f"{folReview}/review-v3.yaml")
    gw3.loadService("ratings", "maistra/examples-bookinfo-ratings-v1:0.12.0",f"{folReview}/rating.yaml")
    
    # Add GW Peers
    printHeader("Add GW2, GW3 peer to GW1")
    gw1.useCluster()
    runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl create peer --name {gw2.name} --host {gw2.ip} --port {gw2.port}')
    runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl create peer --name {gw3.name} --host {gw3.ip} --port {gw3.port}')
    printHeader("Add gw1 peer to gw2")
    gw2.useCluster()
    runcmd(f'kubectl exec -i {gwctl2Pod} -- gwctl create peer --name {gw1.name} --host {gw1.ip} --port {gw1.port}')
    gw3.useCluster()
    printHeader("Add gw3 peer to gw1")
    runcmd(f'kubectl exec -i {gwctl3Pod} -- gwctl create peer --name {gw1.name} --host {gw1.ip} --port {gw1.port}')

    # Set exports  
    gw1.useCluster()
    printHeader(f"create exports {srcSvc1} {srcSvc2}")
    runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl create export --name {srcSvc1} --host {srcSvc1} --port {srcK8sSvcPort}')
    runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl create export --name {srcSvc2} --host {srcSvc2} --port {srcK8sSvcPort}')

    gw2.useCluster()
    review2Ip = f"{getPodIp(reviewSvc)}"
    review2Port = f"{srcK8sSvcPort}"
    runcmd(f'kubectl exec -i {gwctl2Pod} -- gwctl create export --name {reviewSvc} --host {review2Ip} --port {review2Port}')
    

    gw3.useCluster()
    review3Ip = f"{getPodIp(reviewSvc)}"
    review3Port = f"{srcK8sSvcPort}"
    runcmd(f'kubectl exec -i {gwctl3Pod} -- gwctl create export --name {reviewSvc} --host {review3Ip} --port {review3Port}')

    # Import service
    gw1.useCluster()
    printHeader(f"\n\nStart import svc {reviewSvc}")
    runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl create import --name {reviewSvc}  --host {reviewSvc} --port {srcK8sSvcPort} ')
    
    # Binding
    printHeader(f"\n\nStart binding svc {reviewSvc}")
    runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl create binding --import {reviewSvc}  --peer {gw2.name}')
    runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl create binding --import {reviewSvc}  --peer {gw3.name}')
    
    # Get services
    gw1.useCluster()
    printHeader("\n\nStart get service")
    runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl get import')
    runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl get policy')
    
    # Set policies
    printHeader(f"\n\nApplying policy file {allowAllPolicy}")
    policyFile ="/tmp/allowAll.json"
    gw1.useCluster()
    runcmd(f'kubectl cp {allowAllPolicy} gwctl:{policyFile}')
    runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl create policy --type access --policyFile {policyFile}')
    gw2.useCluster()
    runcmd(f'kubectl cp {allowAllPolicy} gwctl:{policyFile}')
    runcmd(f'kubectl exec -i {gwctl2Pod} -- gwctl create policy --type access --policyFile {policyFile}')
    gw3.useCluster()
    runcmd(f'kubectl cp {allowAllPolicy} gwctl:{policyFile}')
    runcmd(f'kubectl exec -i {gwctl3Pod} -- gwctl create policy --type access --policyFile {policyFile}')

# applyPolicy apply policy for the bookInfo demo
def applyPolicy(gw:cluster, type):
    gw.useCluster()
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

def applyFailover(gw, type):
    gw.useCluster()
    clPod=getPodName("cl-dataplane")
    print(clPod)
    if type == "fail":
        printHeader(f"Failing {gw.name} network connection")
        runcmd("kubectl delete service cl-dataplane")
    elif type == "start":
        printHeader(f"Restoring {gw.name} network connection")
        runcmd("kubectl create service nodeport cl-dataplane --tcp=443:443 --node-port=30443")