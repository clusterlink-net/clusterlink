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
from tests.utils.mbgAux import runcmd, runcmdb, waitPod, getPodName,getPodNameIp


import subprocess as sp
import sys
import string
from cr_aux_func import *
from PROJECT_PARAMS import PROJECT_PATH
import time

############################### MAIN ##########################

def mbgBuild(mbgcPort="8443" ,mbgcPortLocal="8443",externalIp=""):

  print("\n\ncreate mbg deploymnet")
  runcmd(f"kubectl create -f {PROJECT_PATH}/manifests/mbg/mbg-cloud.yaml")
  runcmd(f"kubectl create -f {PROJECT_PATH}/manifests/mbgctl/mbgctl-cloud.yaml")
    

  waitPod("mbg")
  waitPod("mbgctl")
  #Creating loadbalancer  and wait to loadbalncer external ip
  runcmd(f"kubectl expose deployment mbg-deployment --name=mbg-load-balancer --port={mbgcPort} --target-port={mbgcPortLocal} --type=LoadBalancer")
  mbgIp=""
  if externalIp !="":
    runcmd("kubectl patch svc mbg-load-balancer -p "+ "\'{\"spec\": {\"type\": \"LoadBalancer\", \"loadBalancerIP\": \""+ externalIp+ "\"}}\'")
    mbgIp= externalIp
  while mbgIp =="":
    print("Waiting for mbg ip...")
    mbgIp=sp.getoutput('kubectl get svc -l app=mbg  -o jsonpath="{.items[0].status.loadBalancer.ingress[0].ip}"')
    time.sleep(10)

  return mbgIp

def mbgSetup(mbg, dataplane, mbgcrtFlags,mbgctlName, mbgIp ,mbgcPort="8443" ,mbgcPortLocal="8443"):  
  print(f"MBG load balancer ip {mbgIp}")
  
  mbgPod,mbgPodIp= getPodNameIp("mbg")
  mbgctlPod,mbgctlPodIp= getPodNameIp("mbgctl")
  print("\n\nStart MBG (along with PolicyEngine)")
  runcmdb(f'kubectl exec -i {mbgPod} -- ./mbg start --id {mbg.name} --ip {mbgIp} --cport {mbgcPort} --cportLocal {mbgcPortLocal}  --dataplane {dataplane} {mbgcrtFlags} --startPolicyEngine {True}')
  destMbgIp = f"{mbgPodIp}:{mbgcPortLocal}" 

  runcmd(f'kubectl exec -it {mbgctlPod} -- ./mbgctl start --id {mbgctlName}  --ip {mbgctlPodIp}  --mbgIP {destMbgIp} --dataplane {dataplane} {mbgcrtFlags}')
  runcmd(f'kubectl exec -it {mbgctlPod} -- ./mbgctl addPolicyEngine --target {mbgPodIp}:9990')



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
