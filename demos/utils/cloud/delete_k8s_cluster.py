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
#Name: delete_k8s_cluster
#Desc: Delete k8s cluster according input information.
#      Delete the relevant images for a proxy cluster.
#      if del_all option is specified -delete all clusters in metadata.json  
#Inputs: cluster_zone, cluster_name ,cluster_platform
#        run_in_bg,del_all
################################################################
import argparse
import os
import subprocess as sp
import sys
from PROJECT_PARAMS import GOOGLE_CONT_REGESTRY , IBM_CONT_REGESTRY
from check_k8s_cluster_ready import connectToCluster
from demos.utils.mbgAux import clean_cluster
############################### functions ##########################
def deleteCluster(cluster, run_in_bg):
    bg_flag= "&" if run_in_bg else ""
    print(f"Deleting cluster {cluster}")
    if cluster.platform == "gcp" :
        os.system(f"yes |gcloud container clusters delete {cluster.name} --zone {cluster.zone} {bg_flag}")
    elif cluster.platform == "aws":
        os.system(f"eksctl delete cluster --region {cluster.zone} --name {cluster.name} {bg_flag}")
    elif cluster.platform == "ibm":
        os.system(f"yes |ibmcloud ks cluster rm --force-delete-storage --cluster {cluster.name} {bg_flag}")
    else:
        print ("ERROR: Cloud platform {} not supported".format(cluster.platform))


def deleteProxyDockerImages(cluster):
    print(f"Deleting : all docker images {cluster}")
    images_lists= ["mbg"]
    images_tags_lists= ["latest"]
    for idx,image in enumerate(images_lists):
        if cluster.platform == "gcp" :
            #delete all images except lastone
            cmd="yes | gcloud container images list-tags "+ GOOGLE_CONT_REGESTRY +"/"+image+" --filter='-tags:*' --format='get(digest)' --limit=unlimited |\
                awk '{print \""+ GOOGLE_CONT_REGESTRY +"/"+image+"@\" $1}' | xargs gcloud container images delete --quiet"
            os.system(cmd)
            #delete the latest images
            cmd="yes | gcloud container images delete {}/{}:{}".format(GOOGLE_CONT_REGESTRY,image,images_tags_lists[idx])
            os.system(cmd)        
        elif cluster.platform == "aws":
            #TODO add support to clean docker images
            print("NO support  on aws images clean")
        elif cluster.platform == "ibm":
            os.system("ibmcloud cr image-rm {}/mbg:latest".format(IBM_CONT_REGESTRY))
            os.system("yes| ibmcloud cr image-prune-untagged")
        else:
            print ("ERROR: Cloud platform {} not supported".format(cluster.platform))

def deleteClustersList(clusters):
    print("Start delete all cluster")
    for idx, mbg in enumerate(clusters):
        run_in_bg= False if  idx == len(clusters)-1 else True
        deleteCluster(mbg,run_in_bg=run_in_bg)

def cleanClustersList(clusters):
    print("Start clean all cluster")
    for mbg in clusters:
        connectToCluster(mbg)
        clean_cluster()
