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

import os
import subprocess as sp
import sys
from meta_data_func import *
from cr_aux_func import *
from typing import runtime_checkable
from PROJECT_PARAMS import PROJECT_PATH
import argparse
import time

############################### MAIN ##########################

def serviceNodeSetup(platform):

  #create haproxy cfg
  # os.system(f"python3 haproxy/haproxy_cfg_gen.py -ip {target_ip} -port {target_port}")
  connect_platform_container_reg(platform)
  container_reg = get_plarform_container_reg(platform)
  # #build docker image
  # print("Start build my-haproxy image")
  # os.system("docker build -t my-haproxy:custom haproxy/ --no-cache")

  # # tag docker image and push it to image container registry
  print("tagging mbg image")
  print(container_reg)
  os.system(f"docker tag mbg:latest {container_reg}/mbg:latest")

  # #push docker image container registry
  print("push mbg image to container registry")
  print(f"docker push {container_reg}/mbg:latest")
  os.system(f"docker push {container_reg}/mbg:latest")

  #creating tcp-split deplyment and service
  #replace_source_image("haproxy/haproxy.yaml","my-haproxy:custom",platform)
  print("\n\ncreate mbg deploymnet")
  os.system(f"kubectl create -f {PROJECT_PATH}/manifests/mbg/mbg.yaml")

  mbg_start_cond=False
  while( not mbg_start_cond):
      mbg_start_cond =sp.getoutput("kubectl get pods -l app=tcp-split -o jsonpath='{.items[0].status.containerStatuses[0].ready}'")
      print(mbg_start_cond)
      print ("Waiting for mbg to start...")
      time.sleep(5)

  #Creating mbg-svc will be reeady
  os.system(f"kubectl create -f  {PROJECT_PATH}/manifests/mbg/mbg-svc.yaml")
  os.system(f"kubectl create -f  {PROJECT_PATH}/manifests/mbg/mbg-client-svc.yaml")
  external_ip=""
  while external_ip =="":
    print("Waiting for mbg ip...")
    external_ip=sp.getoutput('kubectl get nodes -o jsonpath="{.items[*].status.addresses[?(@.type==\'ExternalIP\')].address}"')
    time.sleep(10)

  print("mbg-svc is ready, external_id: {}".format(external_ip))

  ##Create TCP-split service
  os.system(f"docker tag tcp-split:latest {container_reg}/tcp-split:latest")
  os.system(f"docker push {container_reg}/tcp-split:latest")
  os.system(f"kubectl create -f  {PROJECT_PATH}/manifests/tcp-split/tcp-split.yaml")
  os.system(f"kubectl create -f  {PROJECT_PATH}/manifests/tcp-split/tcp-split-svc.yaml")
  
  return external_ip



if __name__ == "__main__":
  parser = argparse.ArgumentParser()

  parser.add_argument("-platform"    , "--cluster_platform", default = "gcp"         , help="setting k8s cloud platform")
  parser.add_argument("-target_ip"    , "--target_ip"      , default  = ""          , help="target ip test")
  parser.add_argument("-target_port"    , "--target_port"      , default  = ""          , help="target port test")


  args = parser.parse_args()
  platform    = args.cluster_platform
  target_ip   = args.target_ip
  target_port = args.target_port