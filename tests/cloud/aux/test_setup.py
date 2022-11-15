################################################################
#Name: single_test
#Desc: Run iperf3 test between 1 host and 1 target via 1 mbg.
#      this test do the following:
#      -Create host, mbg and target k8s clusters     
#      -Deploy iperf3 pods on each target and host cluster
#      -Deploy forwarding and hambg pods on mbg cluster
#      -Run 3 types of iperf3 test between host to target:
#           - direct connection
#           - using hambg pod(tcp splitting) 
#           - using forwarding pod 
#      -Deleate all the clusters.
#Inputs: host_zone ,host_platform, target_zone ,target_platform,
#        mbg_zone,mbg_platform
################################################################
from datetime import datetime
from pytz import timezone  
import os
import argparse
import sys
from time_func import test_start_time, test_end_time
from PROJECT_PARAMS import PROJECT_PATH
from create_k8s_cluster import create_k8s_cluster
from check_k8s_cluster_ready import checkClusterIsReady
from set_k8s_cluster import deployTarget,deployServiceNode,deployHost,setupClientService
from mbg_setup import serviceNodeSetup
from clusterClass import cluster
from iperf3_setup import iperf3Test, iperf3TestDB

def clusterSetup(host, mbg,target):
    #creating source, target, mbg clusters
    create_k8s_cluster(cluster=host,run_in_bg=True)
    create_k8s_cluster(cluster=mbg,run_in_bg=True)
    create_k8s_cluster(cluster=target,run_in_bg=False)


    #setup clusters
    checkClusterIsReady(target)
    deployTarget(target)

    checkClusterIsReady(mbg)
    deployServiceNode(mbg)

    checkClusterIsReady(host)
    deployHost(host)
    print("Finisd setup all clusters")


def hostServiceSetup(host, mbg, target ,service):
    checkClusterIsReady(host)
    setupClientService(mbg=mbg, target=target, service=service)

def testService(service,time=10):
    print(f"Start test for service {service}")
    iperf3Test(target_ip="client-mbg-service", target_port="5001",time=time)

def testServiceDB(service,target_ip, target_port, resFolder,time=10):
    print(f"Start test for service {service}")
    if service != "Direct":
        target_ip="client-mbg-service"
        target_port="5001"
    resFile=resFolder+f"/{service}_res.json"
    iperf3TestDB(target_ip = target_ip, target_port=target_port, resFile=resFile ,time=time)




def getFolderRes(host,target,mbg,resBase=""):
    time_s = getTime()
    #get folder path
    resDir= f'{resBase}//host-{host.zone}_{host.platform}/target-{target.zone}_{target.platform}/mbg-{mbg.zone}_{mbg.platform}/time-{time_s}/'
    if not os.path.exists(resDir):
        os.makedirs(resDir)
    return resDir

def getTime():
    Israel_tz = timezone('Asia/Jerusalem')
    IL_time = datetime.now(Israel_tz)
    dt_string = IL_time.strftime("%d-%m-%Y_%H-%M")
    print("date and time =", dt_string)
    return dt_string

if __name__ == "__main__":

    parser = argparse.ArgumentParser()


    parser.add_argument("-mbg_zone"    , "--mbg_zone"     , default = "us-east1"   , help="describe mbg zone")
    parser.add_argument("-mbg_platform", "--mbg_platform" , default = "gcp"        , help="describe mbg k8s cloud platform")
    parser.add_argument("-h_zone"    , "--host_zone"      , default = "us-central1", help="describe host zone")
    parser.add_argument("-h_platform", "--host_platform"  , default = "gcp"        , help="describe host k8s cloud platform")
    parser.add_argument("-t_zone"    , "--target_zone"    , default = "us-west1"   , help="describe target zone")
    parser.add_argument("-t_platform", "--target_platform", default = "gcp"        , help="describe target k8s cloud platform")
    parser.add_argument("-folder_res" , "--folder_res"    , default = ""          , help="prefix for folder result")

    #python3 tests/scripts/single_test.py  -h_zone us-east1 -p_zone us-central1 -t_zone us-west1
    start_time=test_start_time()
    print("start single run BW tests")
    args = parser.parse_args()
    host_zone       = args.host_zone 
    host_platform   = args.host_platform 
    host_name       = "host-cluster"

    mbg_zone         = args.mbg_zone 
    mbg_platform     = args.mbg_platform
    mbg_name         = "mbg-cluster"

    target_zone     = args.target_zone  
    target_platform = args.target_platform
    target_name   = "iperf3-traget"

