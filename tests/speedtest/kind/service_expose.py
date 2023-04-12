##############################################################################################
# Name: Bookinfo
# Info: support bookinfo application with mbgctl inside the clusters 
#       In this we create three kind clusters
#       1) MBG1- contain mbg, mbgctl,product and details microservices (bookinfo services)
#       2) MBG2- contain mbg, mbgctl, review-v2 and rating microservices (bookinfo services)
#       3) MBG3- contain mbg, mbgctl, review-v3 and rating microservices (bookinfo services)
##############################################################################################

import os,time
import subprocess as sp
import sys
import argparse
proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

from tests.utils.mbgAux import runcmd, runcmdb, printHeader, getPodName, waitPod,getMbgPorts,buildMbg,buildMbgctl,getPodIp,getPodNameIp
from tests.utils.kind.kindAux import useKindCluster,startKindClusterMbg,getKindIp



############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')

    srcSvc1         = "firefox"
    srcSvc2         = "firefox2"
    destSvc         = "openspeedtest"

    mbg1Name        = "mbg1"
    mbg2Name        = "mbg2"
    mbg3Name        = "mbg3"
    mbgctl1Name     = "mbgctl1"
    mbgctl2Name     = "mbgctl2"
    mbgctl3Name     = "mbgctl3"


    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)

    ###get mbg parameters
    useKindCluster(mbg1Name)
    mbg1Pod, _           = getPodNameIp("mbg")
    mbg1Ip               = getKindIp("mbg1")
    mbgctl1Pod, mbgctl1Ip= getPodNameIp("mbgctl")
    useKindCluster(mbg2Name)
    mbg2Pod, _            = getPodNameIp("mbg")
    mbgctl2Pod, mbgctl2Ip = getPodNameIp("mbgctl")
    mbg2Ip                =getKindIp(mbg2Name)
    useKindCluster(mbg3Name)
    mbg3Pod, _            = getPodNameIp("mbg")
    mbg3Ip                = getKindIp("mbg3")
    mbgctl3Pod, mbgctl3Ip = getPodNameIp("mbgctl")

    # Add MBG Peer
    useKindCluster(mbg2Name)

    #Expose service
    printHeader(f"\n\nStart exposing svc {destSvc}")
    runcmd(f'mbgctl expose  --myid {mbgctl2Name} --service {destSvc}')
    
    #Set K8s network services
    printHeader("\n\nStart get service")
    runcmd(f'mbgctl get service --myid {mbgctl1Name} ')
    useKindCluster(mbg1Name)
    mbg1LocalPort, mbg1ExternalPort = getMbgPorts(mbg1Pod, destSvc)
    runcmd(f"kubectl create service clusterip {destSvc} --tcp=3000:{mbg1LocalPort}")
    runcmd(f"kubectl patch service {destSvc} -p "+  "\'{\"spec\":{\"selector\":{\"app\": \"mbg\"}}}\'") #replacing app name

    printHeader("\n\nStart get service")
    useKindCluster(mbg3Name)
    runcmd(f'mbgctl get service  --myid {mbgctl3Name} ')
    mbg3LocalPort, mbg3ExternalPort = getMbgPorts(mbg3Pod, destSvc)
    runcmd(f"kubectl create service clusterip {destSvc} --tcp=3000:{mbg3LocalPort}")
    runcmd(f"kubectl patch service {destSvc} -p "+  "\'{\"spec\":{\"selector\":{\"app\": \"mbg\"}}}\'") #replacing app name
    
    #Firefox communications
    printHeader(f"Firefox urls")
    print(f"To use the mbg1 firefox client, run the command:\n    firefox http://{mbg1Ip}:30000/")
    print(f"To use the first mbg3 firefox client, run the command:\n    firefox http://{mbg3Ip}:30000/")
    print(f"To use the second mbg3 firefox client, run the command:\n   firefox http://{mbg3Ip}:30000/")
    
    print(f"The OpenSpeedTest url: http://{destSvc}:3000/ ")


