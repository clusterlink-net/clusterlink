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

    # Add MBG Peer
    useKindCluster(mbg2Name)

    #Expose service
    printHeader(f"\n\nStart exposing svc {destSvc}")
    runcmd(f'gwctl expose  --myid {gwctl2Name} --service {destSvc}')
    
    #Set K8s network services
    printHeader("\n\nStart get service")
    useKindCluster(mbg1Name)
    runcmd(f'gwctl get service --myid {gwctl1Name} ')
    runcmd(f'gwctl add binding  --myid {gwctl1Name} --service {destSvc} --port 3000')

    mbg1LocalPort, mbg1ExternalPort = getMbgPorts(mbg1Pod, destSvc)
    #runcmd(f"kubectl create service clusterip {destSvc} --tcp=3000:{mbg1LocalPort}")
    #runcmd(f"kubectl patch service {destSvc} -p "+  "\'{\"spec\":{\"selector\":{\"app\": \"mbg\"}}}\'") #replacing app name

    printHeader("\n\nStart get service")
    useKindCluster(mbg3Name)
    runcmd(f'gwctl get service  --myid {gwctl3Name} ')
    #mbg3LocalPort, mbg3ExternalPort = getMbgPorts(mbg3Pod, destSvc)
    runcmd(f'gwctl add binding  --myid {gwctl3Name} --service {destSvc} --port 3000')

    #runcmd(f"kubectl create service clusterip {destSvc} --tcp=3000:{mbg3LocalPort}")
    #runcmd(f"kubectl patch service {destSvc} -p "+  "\'{\"spec\":{\"selector\":{\"app\": \"mbg\"}}}\'") #replacing app name
    #Firefox communications
    printHeader(f"Firefox urls")
    print(f"To use the mbg1 firefox client, run the command:\n    firefox http://{mbg1Ip}:30000/")
    print(f"To use the first mbg3 firefox client, run the command:\n    firefox http://{mbg3Ip}:30000/")
    print(f"To use the second mbg3 firefox client, run the command:\n   firefox http://{mbg3Ip}:30000/")
    
    print(f"The OpenSpeedTest url: http://{destSvc}:3000/ ")


