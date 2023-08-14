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
from demos.utils.cloud.clusterClass import cluster
from iperf3_setup import iperf3Test, iperf3TestDB




def hostServiceSetup(host, mbg, target ,service):
    checkClusterIsReady(host)
    setupClientService(mbg=mbg, target=target, service=service)

def testService(service,time=10):
    print(f"Start test for service {service}")
    iperf3Test(target_ip="client-mbg-service", target_port="5000",time=time)

def testServiceDB(service,target_ip, target_port, resFolder,time=10):
    print(f"Start test for service {service}")
    if service != "Direct":
        target_ip="client-mbg-service"
        target_port="5000"
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
