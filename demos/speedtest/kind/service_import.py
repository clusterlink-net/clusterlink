#!/usr/bin/env python3
##############################################################################################
# Name: Bookinfo
# Info: support bookinfo application with gwctl inside the clusters 
#       In this we create three kind clusters
#       1) MBG1- contain mbg, gwctl,product and details microservices (bookinfo services)
#       2) MBG2- contain mbg, gwctl, review-v2 and rating microservices (bookinfo services)
#       3) MBG3- contain mbg, gwctl, review-v3 and rating microservices (bookinfo services)
##############################################################################################

import os,time
import subprocess as sp
import sys
import argparse
proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

from demos.utils.mbgAux import runcmd, printHeader, getPodNameIp
from demos.utils.kind.kindAux import useKindCluster,getKindIp



############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')

    srcSvc1         = "firefox"
    srcSvc2         = "firefox2"
    destSvc         = "openspeedtest"

    mbg1Name        = "mbg1"
    mbg2Name        = "mbg2"
    mbg3Name        = "mbg3"
    gwctl1Name     = "gwctl1"
    gwctl2Name     = "gwctl2"
    gwctl3Name     = "gwctl3"


    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)

    ###get mbg parameters
    useKindCluster(mbg1Name)
    mbg1Pod, _           = getPodNameIp("mbg")
    mbg1Ip               = getKindIp("mbg1")
    gwctl1Pod, gwctl1Ip= getPodNameIp("gwctl")
    useKindCluster(mbg2Name)
    mbg2Pod, _            = getPodNameIp("mbg")
    gwctl2Pod, gwctl2Ip = getPodNameIp("gwctl")
    mbg2Ip                =getKindIp(mbg2Name)
    useKindCluster(mbg3Name)
    mbg3Pod, _            = getPodNameIp("mbg")
    mbg3Ip                = getKindIp("mbg3")
    gwctl3Pod, gwctl3Ip = getPodNameIp("gwctl")



    #Import service
    printHeader(f"\n\nStart import svc {destSvc}")
    useKindCluster(mbg1Name)    
    runcmd(f'gwctl create import --myid {gwctl1Name} --name {destSvc} --host {destSvc} --port 3000')
    useKindCluster(mbg3Name)    
    runcmd(f'gwctl create import --myid {gwctl3Name} --name {destSvc} --host {destSvc} --port 3000')
    #Set K8s network services
    printHeader("\n\nStart binding service {destSvc}")
    useKindCluster(mbg1Name)
    runcmd(f'gwctl create binding --myid {gwctl1Name} --import {destSvc} --peer {mbg2Name}')
    useKindCluster(mbg3Name)
    runcmd(f'gwctl create binding --myid {gwctl3Name} --import {destSvc} --peer {mbg2Name}')
    
    printHeader("\n\nStart get service GW1")
    runcmd(f'gwctl get import  --myid {gwctl1Name} ')
    printHeader("\n\nStart get service GW3")
    runcmd(f'gwctl get import  --myid {gwctl3Name} ')


    #Firefox communications
    printHeader(f"Firefox urls")
    print(f"To use the mbg1 firefox client, run the command:\n    firefox http://{mbg1Ip}:30000/")
    print(f"To use the first mbg3 firefox client, run the command:\n    firefox http://{mbg3Ip}:30000/")
    print(f"To use the second mbg3 firefox client, run the command:\n   firefox http://{mbg3Ip}:30000/")
    
    print(f"The OpenSpeedTest url: http://{destSvc}:3000/ ")


