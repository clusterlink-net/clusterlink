################################################################
#Name: Service node test
#Desc: create 1 proxy that send data to target ip
###############################################################
import os,sys

file_dir = os.path.dirname(__file__)
sys.path.append(file_dir+"/aux")

from aux.check_k8s_cluster_ready import checkClusterIsReady
from aux.test_setup import clusterSetup,hostServiceSetup, testService, testServiceDB,getFolderRes
from aux.clusterClass import cluster
from aux.delete_k8s_cluster import delete_all_clusters
from aux.PROJECT_PARAMS import PROJECT_PATH
from aux.meta_data_func import getIpPort
from aux.iperf3_setup import iperf3Test, iperf3TestDB

host   = cluster(name="host",   zone = "us-east1-b",    platform = "gcp", type = "host")
target = cluster(name="iperf3-target", zone = "us-west1-b",    platform = "gcp", type = "target")
sn     = cluster(name="sn-k8s",     zone = "us-central1-b", platform = "gcp", type = "servicenode")

# test setup
#clusterSetup(host=host, target=target, sn=sn)

#Test client connection
target_ip, target_port = getIpPort(file=PROJECT_PATH+"/bin/metadata.json", cluster = target)
resFolder=getFolderRes(host,target,sn,resBase=PROJECT_PATH+"/DB/")

testServiceDB(service="Direct", target_ip=target_ip, target_port=target_port, resFolder=resFolder ,time=40)
#Test service Forward
hostServiceSetup(host=host,target=target, sn=sn, service="Forward")
testServiceDB(service="Forward", target_ip=target_ip, target_port=target_port, resFolder=resFolder ,time=40)
#Test service Tcp split
hostServiceSetup(host=host,target=target, sn=sn, service="TCP-split")
testServiceDB(service="TCP-split", target_ip=target_ip, target_port=target_port, resFolder=resFolder ,time=40)


#clean target and source clusters
delete_all_clusters()