#!/usr/bin/env python3
import os,time
import subprocess as sp
import sys
import argparse

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')
sys.path.insert(1,f'{proj_dir}/tests/utils/cloud/')

from tests.utils.mbgAux import runcmd,printHeader, getPodName
from tests.utils.cloud.check_k8s_cluster_ready import connectToCluster
from tests.utils.cloud.clusterClass import cluster

srcSvc1  = "productpage"
srcSvc2  = "productpage2"
destSvc  = "reviews"

mbg1 = cluster(name="mbg1", zone = "us-west1-b"   , platform = "gcp", type = "host")
mbg2 = cluster(name="mbg2", zone = "us-central1-b", platform = "gcp", type = "target")
mbg3 = cluster(name="mbg3", zone = "us-east4-b"   , platform = "gcp", type = "target")
    
def applyPolicy(mbg,type):
    connectToCluster(mbg)
    mbgctlPod=getPodName("mbgctl")
    if type == "ecmp":
        printHeader(f"Set Ecmp poilicy")          
        runcmd(f'kubectl exec -i {mbgctlPod} -- ./mbgctl policy --command lb_set --serviceSrc {srcSvc1} --serviceDst {destSvc} --mbgDest mbg2 --policy ecmp')
        runcmd(f'kubectl exec -i {mbgctlPod} -- ./mbgctl policy --command lb_set --serviceSrc {srcSvc2} --serviceDst {destSvc} --mbgDest mbg2 --policy ecmp')

    elif type == "same":
        printHeader(f"Set same policy to all services")          
        runcmd(f'kubectl exec -i {mbgctlPod} -- ./mbgctl policy --command lb_set --serviceSrc {srcSvc1} --serviceDst {destSvc} --mbgDest mbg2 --policy static')
        runcmd(f'kubectl exec -i {mbgctlPod} -- ./mbgctl policy --command lb_set --serviceSrc {srcSvc2} --serviceDst {destSvc} --mbgDest mbg2 --policy static')
    elif type == "diff":
        runcmd(f'kubectl exec -i {mbgctlPod} -- ./mbgctl policy --command lb_set --serviceSrc {srcSvc1} --serviceDst {destSvc} --mbgDest mbg2 --policy static')
        runcmd(f'kubectl exec -i {mbgctlPod} -- ./mbgctl policy --command lb_set --serviceSrc {srcSvc2} --serviceDst {destSvc} --mbgDest mbg3 --policy static')
    elif type == "show":
        runcmd(f'kubectl exec -i {mbgctlPod} -- ./mbgctl policy --command show')





############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-m','--mbg', help='Either mbg1/mbg2/mbg3', required=False, default="mbg1")
    parser.add_argument('-t','--type', help='Either ecmp/same/diff/show', required=False, default="ecmp")

    args = vars(parser.parse_args())

    mbg = mbg1 if args["mbg"]=="mbg1" else(mbg2 if args["mbg"]=="mbg2" else mbg3)
    type = args["type"]
    
    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)

    applyPolicy(mbg,type)
    