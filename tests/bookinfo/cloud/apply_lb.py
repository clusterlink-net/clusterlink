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
mbglist = { "mbg1gcp" : cluster(name="mbg1", zone = "us-west1-b"    , platform = "gcp", type = "host"),   #Oregon
            "mbg1ibm" : cluster(name="mbg1", zone = "sjc04"         , platform = "ibm", type = "host"),   #San jose
            "mbg2gcp" : cluster(name="mbg2", zone = "us-central1-b" , platform = "gcp", type = "target"), #Iowa
            "mbg2ibm" : cluster(name="mbg2", zone = "dal10"         , platform = "ibm", type = "target"), #Dallas
            "mbg3gcp" : cluster(name="mbg3", zone = "us-east4-b"    , platform = "gcp", type = "target"), #Virginia
            "mbg3ibm" : cluster(name="mbg3", zone = "wdc04"         , platform = "ibm", type = "target")} #Washington DC
    
def applyPolicy(mbg, mbgctlName, type):
    connectToCluster(mbg)
    mbgctlPod=getPodName("mbgctl")
    if type == "ecmp":
        printHeader(f"Set Ecmp poilicy")          
        runcmd(f'kubectl exec -i {mbgctlPod} -- ./mbgctl add policy --myid {mbgctlName} --type lb --serviceDst {destSvc}  --policy ecmp')
    elif type == "same":
        printHeader(f"Set same policy to all services")          
        runcmd(f'kubectl exec -i {mbgctlPod} -- ./mbgctl add policy --myid {mbgctlName} --type lb --serviceDst {destSvc} --mbgDest mbg2 --policy static')
    elif type == "diff":
        runcmd(f'kubectl exec -i {mbgctlPod} -- ./mbgctl add policy --myid {mbgctlName} --type lb --serviceSrc {srcSvc1} --serviceDst {destSvc} --mbgDest mbg2 --policy static')
        runcmd(f'kubectl exec -i {mbgctlPod} -- ./mbgctl add policy --myid {mbgctlName} --type lb --serviceSrc {srcSvc2} --serviceDst {destSvc} --mbgDest mbg3 --policy static')
    elif type == "show":
        runcmd(f'kubectl exec -i {mbgctlPod} -- ./mbgctl get policy')
    elif type == "clean":
        runcmd(f'kubectl exec -i {mbgctlPod} -- ./mbgctl remove policy --myid {mbgctlName} --type lb --serviceSrc {srcSvc2} --serviceDst {destSvc} ')
        runcmd(f'kubectl exec -i {mbgctlPod} -- ./mbgctl remove policy --myid {mbgctlName}--type lb --serviceSrc {srcSvc1} --serviceDst {destSvc} ')
        runcmd(f'kubectl exec -i {mbgctlPod} -- ./mbgctl remove policy --myid {mbgctlName} --type lb --serviceDst {destSvc}')





############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-m','--mbg', help='Either mbg1/mbg2/mbg3', required=False, default="mbg1")
    parser.add_argument('-t','--type', help='Either ecmp/same/diff/show', required=False, default="ecmp")
    parser.add_argument('-cloud','--cloud', help='Cloud setup using gcp/ibm', required=False, default="gcp")

    args = vars(parser.parse_args())

    mbg = mbglist[args["mbg"] + args["cloud"]]
    type = args["type"]
    mbgctlName     = mbg[:-1]+"ctl"+ mbg[-1]

    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)

    applyPolicy(mbg, mbgctlName,type)
    