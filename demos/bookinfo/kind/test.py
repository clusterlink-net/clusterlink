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
#       1) MBG1- contain mbg, gwctl,product and details microservices (bookinfo services)
#       2) MBG2- contain mbg, gwctl, review-v2 and rating microservices (bookinfo services)
#       3) MBG3- contain mbg, gwctl, review-v3 and rating microservices (bookinfo services)
##############################################################################################

import os,time
import subprocess as sp
import sys
import argparse
proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

from demos.utils.mbgAux import runcmd, runcmdb, printHeader, getPodName, waitPod,getMbgPorts,buildMbg,buildMbgctl,getPodIp,getPodNameIp
from demos.utils.kind.kindAux import useKindCluster,startKindClusterMbg,getKindIp


############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-d','--dataplane', help='choose which dataplane to use mtls/tcp', required=False, default="mtls")

    parser.add_argument('-src','--src', help='Source service name', required=False)
    parser.add_argument('-dst','--dest', help='Destination service name', required=False)
    args = vars(parser.parse_args())

    printHeader("\n\nStart Kind Test\n\n")
    printHeader("Start pre-setting")
    
    folpdct   = f"{proj_dir}/demos/bookinfo/manifests/product/"
    folReview = f"{proj_dir}/demos/bookinfo/manifests/review"
    allowAllPolicy =f"{proj_dir}/pkg/policyengine/policytypes/examples/allowAll.json"
    dataplane = args["dataplane"]
 

    destSvc         = "reviews"
    #MBG1 parameters 
    mbg1DataPort    = "30001"
    mbg1cPort       = "30443"
    mbg1cPortLocal  = "443"
    mbg1Name        = "mbg1"
    mbg1crtFlags    = f"--certca ./mtls/ca.crt --cert ./mtls/mbg1.crt --key ./mtls/mbg1.key"  if dataplane =="mtls" else ""
    gwctl1Name     = "gwctl1"
    srcSvc1         = "productpage"
    srcSvc2         = "productpage2"
    srcK8sSvcPort   = "9080"
    srcK8sSvcIp     = ":"+srcK8sSvcPort
    srcDefaultGW    = "10.244.0.1"
    

    #MBG2 parameters 
    mbg2DataPort    = "30001"
    mbg2cPort       = "30443"
    mbg2cPortLocal  = "443"
    mbg2crtFlags    = f"--certca ./mtls/ca.crt --cert ./mtls/mbg2.crt --key ./mtls/mbg2.key"  if dataplane =="mtls" else ""
    mbg2Name        = "mbg2"
    gwctl2Name     = "gwctl2"
    review2DestPort = "30001"
    review2pod      = "reviews-v2"
    
    #MBG3 parameters 
    mbg3DataPort    = "30001"
    mbg3cPort       = "30443"
    mbg3cPortLocal  = "443"
    mbg3crtFlags    = f"--certca ./mtls/ca.crt --cert ./mtls/mbg3.crt --key ./mtls/mbg3.key"  if dataplane =="mtls" else ""
    mbg3Name        = "mbg3"
    gwctl3Name     = "gwctl3"
    review3DestPort = "30001"
    review3pod      = "reviews-v3"

    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)

    ### clean 
    print(f"Clean old kinds")
    os.system("make clean-kind-bookinfo")
    
    ### build docker environment 
    printHeader(f"Build docker image")
    os.system("make docker-build")
    
    ## build Kind clusters environment 
    startKindClusterMbg(mbg1Name, gwctl1Name, mbg1cPortLocal, mbg1cPort, mbg1DataPort, dataplane ,mbg1crtFlags)        
    startKindClusterMbg(mbg2Name, gwctl2Name, mbg2cPortLocal, mbg2cPort, mbg2DataPort,dataplane ,mbg2crtFlags)        
    startKindClusterMbg(mbg3Name, gwctl3Name, mbg3cPortLocal, mbg3cPort, mbg3DataPort,dataplane ,mbg3crtFlags)        
    
    ###get mbg parameters
    useKindCluster(mbg1Name)
    mbg1Pod, _           = getPodNameIp("mbg")
    mbg1Ip               = getKindIp("mbg1")
    gwctl1Pod, gwctl1Ip= getPodNameIp("gwctl")
    useKindCluster(mbg2Name)
    mbg2Pod, _            = getPodNameIp("mbg")
    gwctl2Pod, gwctl2Ip = getPodNameIp("gwctl")
    mbg2Ip                =getKindIp(mbg2Name)
    useKindCluster(mbg3Name)
    mbg3Pod, _            = getPodNameIp("mbg")
    mbg3Ip                = getKindIp("mbg3")
    gwctl3Pod, gwctl3Ip = getPodNameIp("gwctl")

    ###Set mbg1 services
    useKindCluster(mbg1Name)
    runcmd(f"kind load docker-image maistra/examples-bookinfo-productpage-v1 --name={mbg1Name}")
    runcmd(f"kind load docker-image maistra/examples-bookinfo-details-v1:0.12.0 --name={mbg1Name}")
    runcmd(f"kubectl create -f {folpdct}/product.yaml")
    runcmd(f"kubectl create -f {folpdct}/product2.yaml")
    runcmd(f"kubectl create -f {folpdct}/details.yaml")
    printHeader(f"Add {srcSvc1} {srcSvc2}  services to host cluster")
    waitPod(srcSvc1)
    waitPod(srcSvc2)
    _ , srcSvcIp1 =getPodNameIp(srcSvc1)
    _ , srcSvcIp2 =getPodNameIp(srcSvc2)
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl create export --name {srcSvc1} --port {srcK8sSvcPort}')
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl create export --name {srcSvc2} --port {srcK8sSvcPort}')

    

    # Add GW Peers
    printHeader("Add GW2, GW3 peer to GW1")
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl create peer --name {mbg2Name} --host {mbg2Ip} --port {mbg2cPort}')
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl create peer --name {mbg3Name} --host {mbg3Ip} --port {mbg3cPort}')
    printHeader("Add MBG1 peer to MBG2")
    runcmd(f'kubectl exec -i {gwctl2Pod} -- ./gwctl create peer --name {mbg1Name} --host {mbg1Ip} --port {mbg1cPort}')
    useKindCluster(mbg3Name)
    printHeader("Add MBG3 peer to MBG1")
    runcmd(f'kubectl exec -i {gwctl3Pod} -- ./gwctl create peer --name {mbg1Name} --host {mbg1Ip} --port {mbg1cPort}')

    
    ###Set mbg2 service
    useKindCluster(mbg2Name)
    runcmd(f"kind load docker-image maistra/examples-bookinfo-reviews-v2 --name={mbg2Name}")
    runcmd(f"kind load docker-image maistra/examples-bookinfo-ratings-v1:0.12.0 --name={mbg2Name}")
    runcmd(f"kubectl create -f {folReview}/review-v2.yaml")
    runcmd(f"kubectl create -f {folReview}/rating.yaml")
    printHeader(f"Add {destSvc} (server) service to destination cluster")
    waitPod(destSvc)
    destSvcReview2Ip = f"{getPodIp(destSvc)}"
    destSvcReview2Port = f"{srcK8sSvcPort}"
    runcmd(f'kubectl exec -i {gwctl2Pod} -- ./gwctl create export --name {destSvc} --host {destSvcReview2Ip} --port {destSvcReview2Port}')
    

    ###Set gwctl3
    useKindCluster(mbg3Name)
    runcmd(f"kind load docker-image maistra/examples-bookinfo-reviews-v3 --name={mbg3Name}")
    runcmd(f"kind load docker-image maistra/examples-bookinfo-ratings-v1:0.12.0 --name={mbg3Name}")
    runcmd(f"kubectl create -f {folReview}/review-v3.yaml")
    runcmd(f"kubectl create -f {folReview}/rating.yaml")
    printHeader(f"Add {destSvc} (server) service to destination cluster")
    waitPod(destSvc)
    destSvcReview3Ip = f"{getPodIp(destSvc)}"
    destSvcReview3Port = f"{srcK8sSvcPort}"
    runcmd(f'kubectl exec -i {gwctl3Pod} -- ./gwctl create export --name {destSvc} --host {destSvcReview3Ip} --port {destSvcReview3Port}')

    #Import service
    useKindCluster(mbg1Name)
    printHeader(f"\n\nStart import svc {destSvc}")
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl create import --name {destSvc}  --host {destSvc} --port {srcK8sSvcPort} ')
    #Import service
    printHeader(f"\n\nStart binding svc {destSvc}")
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl create binding --import {destSvc}  --peer {mbg2Name}')
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl create binding --import {destSvc}  --peer {mbg3Name}')
    
    #Get services
    useKindCluster(mbg1Name)
    printHeader("\n\nStart get service")
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl get import')
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl get policy')
    
 # Set policies
    printHeader(f"\n\nApplying policy file {allowAllPolicy}")
    useKindCluster(mbg1Name)
    runcmd(f'gwctl --myid {gwctl1Name} create policy --type access --policyFile {allowAllPolicy}')
    runcmd(f'gwctl --myid {gwctl2Name} create policy --type access --policyFile {allowAllPolicy}')
    runcmd(f'gwctl --myid {gwctl3Name} create policy --type access --policyFile {allowAllPolicy}')


    print(f"Proctpage1 url: http://{mbg1Ip}:30001/productpage")
    print(f"Proctpage2 url: http://{mbg1Ip}:30002/productpage")


