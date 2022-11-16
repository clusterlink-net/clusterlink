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
from  PROJECT_PARAMS import GOOGLE_CONT_REGESTRY , IBM_CONT_REGESTRY, METADATA_FILE

sys.path.insert(1, 'project_metadata/')
from meta_data_func import *

try:
    from typing import runtime_checkable
except ImportError:
    from typing_extensions import runtime_checkable
parser = argparse.ArgumentParser()

############################### functions ##########################
def delete_cluster(name,zone,platform, run_in_bg):
    bg_flag= "&" if run_in_bg else ""
    print("Deleting clustre {} zone {} platform {} {}".format(name, zone, platform,bg_flag) )
    if platform == "gcp" :
        os.system("yes |gcloud container clusters delete {} --zone {} {}".format(name,zone,bg_flag))
    elif platform == "aws":
        os.system("eksctl delete cluster --region {} --name {} {}".format(zone,name,bg_flag))
    elif platform == "ibm":
            os.system("yes |ibmcloud ks cluster rm --force-delete-storage --cluster {} {}".format(name,bg_flag))
    else:
        print ("ERROR: Cloud platform {} not supported".format(platform))
    if "proxy" in name:
        delete_proxy_docker_images(name,zone,platform)
    delete_cluster_meta_data(METADATA_FILE,name,zone)

def delete_proxy_docker_images(name,zone,platform):
    print("DELETING: all docker images {} zone {} platform {} ".format(name, zone, platform))
    images_lists= ["my-haproxy","forwarding-proxy"]
    images_tags_lists= ["custom","latest"]
    for idx,image in enumerate(images_lists):
        if platform == "gcp" :
            #delete all images except lastone
            cmd="yes | gcloud container images list-tags "+ GOOGLE_CONT_REGESTRY +"/"+image+" --filter='-tags:*' --format='get(digest)' --limit=unlimited |\
                awk '{print \""+ GOOGLE_CONT_REGESTRY +"/"+image+"@\" $1}' | xargs gcloud container images delete --quiet"
            os.system(cmd)
            #delete the latest images
            cmd="yes | gcloud container images delete {}/{}:{}".format(GOOGLE_CONT_REGESTRY,image,images_tags_lists[idx])
            os.system(cmd)        
        elif platform == "aws":
            #TODO add support to clean docker images
            print("NO support  on aws images clean")
        elif platform == "ibm":
            os.system("ibmcloud cr image-rm {}/forwarding-proxy".format(IBM_CONT_REGESTRY))
            os.system("ibmcloud cr image-rm {}/my-haproxy:custom".format(IBM_CONT_REGESTRY))
            os.system("yes| ibmcloud cr image-prune-untagged")
        else:
            print ("ERROR: Cloud platform {} not supported".format(platform))

def delete_all_clusters():
    print("Finish start delete all clusters")

    while (not is_empty_metadata(METADATA_FILE)):
        print("Start removing all clusters")
        cluster_name,cluster_zone, cluster_platform =get_first_item_metadata(METADATA_FILE)
        bg= True if len_metadata(METADATA_FILE) > 1 else False
        delete_cluster(cluster_name, cluster_zone, cluster_platform, bg)
        print("Finish removing all clusters")

############################### MAIN ##########################
if __name__ == "__main__":

    #Parser
    parser.add_argument("-zone"     , "--cluster_zone"     , default = "us-east1-b", help="describe clustr zone")
    parser.add_argument("-name"     , "--cluster_name"     , default = "host_k8s"  , help="describe cluster name")
    parser.add_argument("-platform" , "--cluster_platform" , default = ""          , help="describe cluster platform")

    parser.add_argument("-all" ,  "--del_all"      , default = False       , help="Delete all cluster exist")
    parser.add_argument("-bg"   , "--run_in_bg"    , default = False     , help="Run in background")
    #CMD example: python3 ./steps/delete_k8s_cluster.py -name proxy-k8s  -zone us-central1-a

    args = parser.parse_args()
    cluster_zone     = args.cluster_zone
    cluster_name     = args.cluster_name
    cluster_platform = args.cluster_platform
    bg               = args.run_in_bg
    del_all          = args.del_all
    run_in_bg        = args.run_in_bg

    if (del_all == False):
        if (cluster_platform == ""):
            cluster_platform =get_platform(METADATA_FILE,cluster_name,cluster_zone)
        delete_cluster(cluster_name,cluster_zone,cluster_platform,run_in_bg)
    else:
        delete_all_clusters()