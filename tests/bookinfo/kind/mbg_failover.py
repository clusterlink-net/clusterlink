#!/usr/bin/env python3
import os,time
import subprocess as sp
import sys
import argparse

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

from tests.utils.mbgAux import runcmd, runcmdb, printHeader, getPodName, waitPod,getMbgPorts,buildMbg,buildMbgctl,getPodIp,getPodNameIp
from tests.utils.kind.kindAux import useKindCluster,startKindClusterMbg,getKindIp

srcSvc1  = "productpage"
srcSvc2  = "productpage2"
destSvc  = "reviews"
    

#MBG3 parameters 
mbg3DataPort    = "30001"
mbg3cPort       = "30443"
mbg3cPortLocal  = "8443"
mbg3crtFlags    = "--rootCa ./mtls/ca.crt --certificate ./mtls/mbg3.crt --key ./mtls/mbg3.key"
mbg3Name        = "mbg3"
mbgctl3Name     = "mbgctl3"

destSvc      = "reviews"
    

def exposeService(mbgName, mbgCtlName, destSvc):
    mbgctlPod = getPodName("mbgctl")
    printHeader(f"\n\nStart exposing {destSvc} service to {mbgName}")
    runcmd(f'kubectl exec -i {mbgctlPod} -- ./mbgctl expose --myid {mbgCtlName} --service {destSvc}')


def applyFail(mbg, mbgctlName, type):
    useKindCluster(mbg)
    mPod=getPodName("mbg-")
    print(mPod)
    mbgKindIp=getKindIp(mbg)
    print(mbgKindIp)

    if type == "fail":
        printHeader(f"Failing MBG")
        runcmd(f'kubectl exec -i {mPod} -- killall mbg')
    elif type == "start":
        printHeader(f"Starting up and Restoring MBG")
        runcmdb(f'kubectl exec -i {mPod} -- ./mbg start --id "{mbg3Name}" --ip {mbgKindIp} --cport {mbg3cPort} --cportLocal {mbg3cPortLocal}  --externalDataPortRange {mbg3DataPort}\
    --dataplane mtls {mbg3crtFlags} --startPolicyEngine {True} --restore {True}')
        time.sleep(2)
        exposeService(mbg, mbgctlName, destSvc)


from tests.utils.mbgAux import runcmd, runcmdb, printHeader, getPodName
from tests.utils.kind.kindAux import useKindCluster

############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-t','--type', help='Either fail/start', required=False, default="fail")

    args = vars(parser.parse_args())

    type = args["type"]
    
    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)

    applyFail(mbg3Name, mbgctl3Name, type)
    