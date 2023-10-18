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

################################################################
#Name: haproxy_setup
#Desc: Create docker image of  haproxy.
#      Upload to the cloud platform and create a pod in the proxy server 
#      (deployment and service from haproxy folder).
#      set the proxy pod to forward all traffic to target ip.
#      In addition create Iperf3 client for option to check 
#      latency from proxy server.
#      Use in proxy servers
#  
#Inputs: cluster_platform,target_ip
################################################################
import os,sys
from demos.utils.mbgAux import runcmd, runcmdb, waitPod, getPodName,getPodNameIp


import subprocess as sp
import sys
import string
from cr_aux_func import *
from PROJECT_PARAMS import PROJECT_PATH
import time

############################### MAIN ##########################

def mbgBuild(mbgcPort="443" ,mbgcPortLocal="443",externalIp=""):

  print("\n\ncreate mbg deploymnet")
  runcmd(f"kubectl create -f {PROJECT_PATH}/config/manifests/mbg/mbg-cloud.yaml")
  runcmd(f"kubectl create -f {PROJECT_PATH}/config/manifests/mbg/mbg-role.yaml")
  runcmd(f"kubectl create -f {PROJECT_PATH}/config/manifests/gwctl/gwctl-cloud.yaml")
    

  waitPod("mbg")
  waitPod("gwctl")
  #Creating loadbalancer  and wait to loadbalncer external ip
  runcmd(f"kubectl expose deployment mbg-deployment --name=mbg-load-balancer --port={mbgcPort} --host-port={mbgcPortLocal} --type=LoadBalancer")
  mbgIp=""
  if externalIp !="":
    runcmd("kubectl patch svc mbg-load-balancer -p "+ "\'{\"spec\": {\"type\": \"LoadBalancer\", \"loadBalancerIP\": \""+ externalIp+ "\"}}\'")
    mbgIp= externalIp
  while mbgIp =="":
    print("Waiting for mbg ip...")
    mbgIp=sp.getoutput('kubectl get svc -l app=mbg  -o jsonpath="{.items[0].status.loadBalancer.ingress[0].ip}"')
    time.sleep(10)

  return mbgIp

def mbgSetup(mbg, dataplane, mbgcrtFlags,gwctlName, mbgIp ,mbgcPort="443" ,mbgcPortLocal="443"):  
  print(f"MBG load balancer ip {mbgIp}")
  
  mbgPod,mbgPodIp= getPodNameIp("mbg")
  gwctlPod,gwctlPodIp= getPodNameIp("gwctl")
  print("\n\nStart MBG (along with PolicyEngine)")
  runcmdb(f'kubectl exec -i {mbgPod} -- ./controlplane start --name {mbg.name} --ip {mbgIp} --cport {mbgcPort} --cportLocal {mbgcPortLocal}  --dataplane {dataplane} {mbgcrtFlags} --startPolicyEngine {True}')
  runcmd(f'kubectl exec -it {gwctlPod} -- ./gwctl init --name {gwctlName}  --gwIP {mbgPodIp} --gwPort {mbgcPortLocal} --dataplane {dataplane} {mbgcrtFlags}')



def pushImage(platform):
  connect_platform_container_reg(platform)
  container_reg = get_plarform_container_reg(platform)
  
  # tag docker image and push it to image container registry
  print("tagging mbg image")
  print(container_reg)
  os.system(f"docker tag mbg:latest {container_reg}/mbg:latest")

  #push docker image container registry
  print("push mbg image to container registry")
  print(f"docker push {container_reg}/mbg:latest")
  os.system(f"docker push {container_reg}/mbg:latest")
