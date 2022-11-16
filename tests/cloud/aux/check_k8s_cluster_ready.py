################################################################
#Name: check_k8s_cluster_ready 
#Desc: Check if the k8s cluster is created and running.
#Inputs: cluster_zone, cluster_type, cluster_name ,cluster_platform
################################################################

import os  
import subprocess as sp
import sys
import time
from meta_data_func import *
from typing import runtime_checkable
import argparse
parser = argparse.ArgumentParser()
from PROJECT_PARAMS import GOOGLE_PROJECT_ID
from clusterClass import cluster

def connect_to_cluster(cluster):
    print(f"\n CONNECT TO: {cluster.name} in zone: {cluster.zone} ,platform: {cluster.platform}\n")
    connect_flag= False
    while (not connect_flag):
        if cluster.platform == "gcp":
            cmd=f"gcloud container clusters  get-credentials {cluster.name} --zone {cluster.zone} --project  {GOOGLE_PROJECT_ID}"
            print(cmd)
        elif cluster.platform == "aws":
            cmd=f"aws eks --region {cluster.zone} update-kubeconfig --name {cluster.name}"
        elif cluster.platform == "ibm":
            cmd=f"ibmcloud ks cluster config --cluster {cluster.name}"
        else:
            print (f"ERROR: Cloud platform {cluster.patform} not supported")
            exit(1)
        
        out_cmd=sp.getoutput(cmd)
        print("connection output: {}".format(out_cmd))
        connect_flag = False if ("ERROR" in out_cmd or "Failed" in out_cmd) else True
        if not connect_flag: 
            time.sleep(30) #wait more time to connection
        return out_cmd

def checkClusterIsReady(cluster):
    connect_flag= False
    while (not connect_flag):
        ret_out=connect_to_cluster(cluster)
        connect_flag = False if ("ERROR" in ret_out or "Failed" in ret_out or "FAILED" in ret_out) else True
        time.sleep(20)

    print(f"\n Cluster Ready: {cluster.name} in zone: {cluster.zone} ,platform: {cluster.platform}\n")
############################### MAIN ##########################


if __name__ == "__main__":

    #Parser
    parser.add_argument("-zone"        , "--cluster_zone"    , default  = "us-east1-b" , help="setting k8s cluster zone")
    parser.add_argument("-type"        , "--cluster_type"    , default  = "host"       , help="setting k8s cluster typw")
    parser.add_argument("-name"        , "--cluster_name"    , default  = ""          , help="setting k8s cluster name")
    parser.add_argument("-platform"    , "--cluster_platform", default = "gcp"         , help="setting k8s cloud platform")

    args = parser.parse_args()
    cluster_zone   = args.cluster_zone
    cluster_type   = args.cluster_type
    cluster_platform  = args.cluster_platform


    if (args.cluster_name == ""):
        cluster_name   = "host-k8s" if (cluster_type == "host") else ("target-k8s" if (cluster_type == "target") else "proxy-k8s" )
    else:
        cluster_name  = args.cluster_name
    checkClusterIsReady(cluster(cluster_name,cluster_zone,cluster_platform,cluster_type))

