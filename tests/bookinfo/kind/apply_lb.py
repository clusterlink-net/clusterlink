#!/usr/bin/env python3
import os,time
import subprocess as sp
import sys
import argparse

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

srcSvc1  = "productpage"
srcSvc2  = "productpage2"
destSvc  = "reviews"
    
    
def applyPolicy(mbg,type):
    useKindCluster(mbg)
    mbgctlPod=getPodName("mbgctl")
    if type == "ecmp":
        printHeader(f"Set Ecmp poilicy")          
        runcmd(f'kubectl exec -i {mbgctlPod} -- ./mbgctl policy --command lb_add --serviceDst {destSvc} --mbgDest mbg2 --policy ecmp')
    elif type == "same":
        printHeader(f"Set same policy to all services")          
        runcmd(f'kubectl exec -i {mbgctlPod} -- ./mbgctl policy --command lb_add --serviceDst {destSvc} --mbgDest mbg2 --policy static')
    elif type == "diff":
        runcmd(f'kubectl exec -i {mbgctlPod} -- ./mbgctl policy --command lb_add --serviceSrc {srcSvc1} --serviceDst {destSvc} --mbgDest mbg2 --policy static')
        runcmd(f'kubectl exec -i {mbgctlPod} -- ./mbgctl policy --command lb_add --serviceSrc {srcSvc2} --serviceDst {destSvc} --mbgDest mbg3 --policy static')
    elif type == "show":
        runcmd(f'kubectl exec -i {mbgctlPod} -- ./mbgctl policy --command show')
    elif type == "clean":
        runcmd(f'kubectl exec -i {mbgctlPod} -- ./mbgctl policy --command lb_del --serviceSrc {srcSvc2} --serviceDst {destSvc} ')
        runcmd(f'kubectl exec -i {mbgctlPod} -- ./mbgctl policy --command lb_del --serviceSrc {srcSvc1} --serviceDst {destSvc} ')
        runcmd(f'kubectl exec -i {mbgctlPod} -- ./mbgctl policy --command lb_del --serviceDst {destSvc}')



from tests.utils.mbgAux import runcmd, runcmdb, printHeader, getPodName
from tests.utils.kind.kindAux import useKindCluster

############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-m','--mbg', help='Either mbg1/mbg2/mbg3', required=False, default="mbg1")
    parser.add_argument('-t','--type', help='Either ecmp/same/diff/show', required=False, default="ecmp")

    args = vars(parser.parse_args())

    mbg = args["mbg"]
    type = args["type"]
    
    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)

    applyPolicy(mbg,type)
    