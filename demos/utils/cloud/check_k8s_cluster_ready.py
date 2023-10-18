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
#Name: check_k8s_cluster_ready 
#Desc: Check if the k8s cluster is created and running.
#Inputs: cluster_zone, cluster_type, cluster_name ,cluster_platform
################################################################


import subprocess as sp
import time,os,sys
proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')
from PROJECT_PARAMS import GOOGLE_PROJECT_ID
from demos.utils.cloud.clusterClass import cluster

def connectToCluster(cluster):
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
        connect_flag = False if ("ERROR" in out_cmd or "WARNING" in out_cmd or "Failed" in out_cmd) else True
        if not connect_flag: 
            time.sleep(30) #wait more time to connection
        return out_cmd

def checkClusterIsReady(cluster):
    connect_flag= False
    while (not connect_flag):
        ret_out=connectToCluster(cluster)
        connect_flag = False if ("ERROR" in ret_out or "Failed" in ret_out or "FAILED" in ret_out) else True
        time.sleep(20)

    print(f"\n Cluster Ready: {cluster.name} in zone: {cluster.zone} ,platform: {cluster.platform}\n")

############################### MAIN ##########################
if __name__ == "__main__":
    checkClusterIsReady(cluster(name="mbg1", zone = "syd01"        , platform = "ibm", type = "target"))