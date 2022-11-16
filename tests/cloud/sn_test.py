################################################################
#Name: Service node test
#Desc: create 1 proxy that send data to target ip
###############################################################
import os,sys

file_dir = os.path.dirname(__file__)
sys.path.append(file_dir+"/aux")

from aux.check_k8s_cluster_ready import checkClusterIsReady
from aux.test_setup import clusterSetup,hostServiceSetup, testService
from aux.clusterClass import cluster
from aux.delete_k8s_cluster import delete_all_clusters
from aux.PROJECT_PARAMS import PROJECT_PATH
from aux.meta_data_func import getIpPort
from aux.iperf3_setup import iperf3Test

host   = cluster(name="host",   zone = "us-east1-b",    platform = "gcp", type = "host")
target = cluster(name="iperf3-target", zone = "us-west1-b",    platform = "gcp", type = "target")
mbg     = cluster(name="mbg-k8s",     zone = "us-central1-b", platform = "gcp", type = "mbg")

# test setup
#clusterSetup(host=host, target=target, mbg=mbg)

#Test client connection
checkClusterIsReady(host)
target_ip, target_port = getIpPort(file=PROJECT_PATH+"/bin/metadata.json", cluster = target)
#iperf3Test(target_ip=target_ip, target_port=target_port, time=40)

#Test service Forward
hostServiceSetup(host=host,target=target, mbg=mbg, service="Forward")
testService(service="Forward", time=40)
#Test service Tcp split
hostServiceSetup(host=host,target=target, mbg=mbg, service="TCP-split")
testService(service="TCP-split", time=40)


#clean target and source clusters
#delete_all_clusters()