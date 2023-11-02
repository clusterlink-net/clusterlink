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
proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

from demos.utils.common import runcmd, createFabric, printHeader
from demos.utils.kind import startKindCluster,useKindCluster, getKindIp,loadService
from demos.utils.k8s import getPodNameIp,getPodIp

############################### MAIN ##########################
if __name__ == "__main__":
   printHeader("\n\nStart Kind Test\n\n")
   printHeader("Start pre-setting")
   
   folpdct   = f"{proj_dir}/demos/bookinfo/manifests/product/"
   folReview = f"{proj_dir}/demos/bookinfo/manifests/review"
   allowAllPolicy =f"{proj_dir}/pkg/policyengine/policytypes/examples/allowAll.json"
   testOutputFolder = f"{proj_dir}/bin/tests/bookinfo" 

   #GW parameters 
   gwPort        = "30443"    
   gwNS          = "default"    
   gw1Name       = "peer1"
   gw2Name       = "peer2"
   gw3Name       = "peer3"
   reviewSvc       = "reviews"
   srcSvc1       = "productpage"
   srcSvc2       = "productpage2"
   srcK8sSvcPort = "9080"
   review2pod    = "reviews-v2"
   review3pod    = "reviews-v3"



   print(f'Working directory {proj_dir}')
   os.chdir(proj_dir)

   ### clean 
   print(f"Clean old kinds")
   os.system("make clean-kind-bookinfo")
   
   ### build docker environment 
   printHeader(f"Build docker image")
   os.system("make docker-build")
   
   ## build Kind clusters environment
   createFabric(testOutputFolder) 
   startKindCluster(gw1Name, testOutputFolder)        
   startKindCluster(gw2Name, testOutputFolder)
   startKindCluster(gw3Name, testOutputFolder)       
            
   
   ###get gw parameters
   gw1Ip               = getKindIp(gw1Name)
   gwctl1Pod, gwctl1Ip = getPodNameIp("gwctl")
   gw2Ip               = getKindIp(gw2Name)
   gwctl2Pod, gwctl2Ip = getPodNameIp("gwctl")
   gw3Ip               = getKindIp(gw3Name)
   gwctl3Pod, gwctl3Ip = getPodNameIp("gwctl")

   # Add GW Peers
   printHeader("Add GW2, GW3 peer to GW1")
   useKindCluster(gw1Name)
   runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl create peer --name {gw2Name} --host {gw2Ip} --port {gwPort}')
   runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl create peer --name {gw3Name} --host {gw3Ip} --port {gwPort}')
   printHeader("Add gw1 peer to gw2")
   useKindCluster(gw2Name)
   runcmd(f'kubectl exec -i {gwctl2Pod} -- gwctl create peer --name {gw1Name} --host {gw1Ip} --port {gwPort}')
   useKindCluster(gw3Name)
   printHeader("Add gw3 peer to gw1")
   runcmd(f'kubectl exec -i {gwctl3Pod} -- gwctl create peer --name {gw1Name} --host {gw1Ip} --port {gwPort}')

   ###Set GW1 services
   useKindCluster(gw1Name)
   printHeader(f"Add {srcSvc1} {srcSvc2}  services to host cluster")
   loadService(srcSvc1, gw1Name, "maistra/examples-bookinfo-productpage-v1",f"{folpdct}/product.yaml")
   loadService(srcSvc2, gw1Name, "maistra/examples-bookinfo-productpage-v1",f"{folpdct}/product2.yaml")
   loadService(srcSvc1, gw1Name, "maistra/examples-bookinfo-details-v1:0.12.0",f"{folpdct}/details.yaml")

   runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl create export --name {srcSvc1} --host {srcSvc1} --port {srcK8sSvcPort}')
   runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl create export --name {srcSvc2} --host {srcSvc2} --port {srcK8sSvcPort}')
   
   ###Set gw2 service
   useKindCluster(gw2Name)
   loadService(reviewSvc, gw2Name, "maistra/examples-bookinfo-reviews-v2",f"{folReview}/review-v2.yaml")
   loadService("ratings", gw2Name, "maistra/examples-bookinfo-ratings-v1:0.12.0",f"{folReview}/rating.yaml")
   review2Ip = f"{getPodIp(reviewSvc)}"
   review2Port = f"{srcK8sSvcPort}"
   runcmd(f'kubectl exec -i {gwctl2Pod} -- gwctl create export --name {reviewSvc} --host {review2Ip} --port {review2Port}')
   

   ###Set gwctl3
   useKindCluster(gw3Name)
   loadService(reviewSvc, gw3Name, "maistra/examples-bookinfo-reviews-v3",f"{folReview}/review-v3.yaml")
   loadService("ratings", gw3Name, "maistra/examples-bookinfo-ratings-v1:0.12.0",f"{folReview}/rating.yaml")
   review3Ip = f"{getPodIp(reviewSvc)}"
   review3Port = f"{srcK8sSvcPort}"
   runcmd(f'kubectl exec -i {gwctl3Pod} -- gwctl create export --name {reviewSvc} --host {review3Ip} --port {review3Port}')

   #Import service
   useKindCluster(gw1Name)
   printHeader(f"\n\nStart import svc {reviewSvc}")
   runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl create import --name {reviewSvc}  --host {reviewSvc} --port {srcK8sSvcPort} ')
   
   #Import service
   printHeader(f"\n\nStart binding svc {reviewSvc}")
   runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl create binding --import {reviewSvc}  --peer {gw2Name}')
   runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl create binding --import {reviewSvc}  --peer {gw3Name}')
   
   #Get services
   useKindCluster(gw1Name)
   printHeader("\n\nStart get service")
   runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl get import')
   runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl get policy')
   
   # Set policies
   printHeader(f"\n\nApplying policy file {allowAllPolicy}")
   policyFile ="/tmp/allowAll.json"
   useKindCluster(gw1Name)
   runcmd(f'kubectl cp {allowAllPolicy} gwctl:{policyFile}')
   runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl create policy --type access --policyFile {policyFile}')
   useKindCluster(gw2Name)
   runcmd(f'kubectl cp {allowAllPolicy} gwctl:{policyFile}')
   runcmd(f'kubectl exec -i {gwctl2Pod} -- gwctl create policy --type access --policyFile {policyFile}')
   useKindCluster(gw3Name)
   runcmd(f'kubectl cp {allowAllPolicy} gwctl:{policyFile}')
   runcmd(f'kubectl exec -i {gwctl3Pod} -- gwctl create policy --type access --policyFile {policyFile}')

   print(f"Proctpage1 url: http://{gw1Ip}:30001/productpage")
   print(f"Proctpage2 url: http://{gw1Ip}:30002/productpage")


