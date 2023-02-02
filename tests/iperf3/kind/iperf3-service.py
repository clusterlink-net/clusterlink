#!/usr/bin/env python3
import os,time
import subprocess as sp
import sys
import argparse

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))

sys.path.insert(0,f'{proj_dir}/tests/')
print(f"{proj_dir}/tests/")
from aux.kindAux import runcmd, runcmdb, printHeader, waitPod, getPodName, getKindIp, getMbgPorts,buildMbg,buildMbgctl,useKindCluster,getPodIp
from dotenv import load_dotenv

############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-d','--dataplane', help='choose which dataplane to use mtls/tcp', required=False, default="tcp")
    args = vars(parser.parse_args())

    dataplane = args["dataplane"]
    #parameters 
    mbg1ClusterName ="mbg-agent1"
    srcSvc          = "iperf3-client"
    mbg2ClusterName = "mbg-agent2"
    mbgctl2Name     = "mbgctl2"
    destSvc         = "iperf3-server"
    mbg3ClusterName = "mbg-agent3"
        
    #folders
    folCl=f"{proj_dir}/tests/iperf3/manifests/iperf3-client"
    folSv=f"{proj_dir}/tests/iperf3/manifests/iperf3-server"
    
    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)

    load_dotenv()
    podMbg1 = os.getenv("podMbg1")
    mbg1Ip = os.getenv("mbg1Ip")
    mbgctl1Pod = os.getenv("mbgctl1Pod")
    podMbg2 = os.getenv("podMbg2")
    mbg2Ip = os.getenv("mbg2Ip")
    mbgctl2Pod = os.getenv("mbgctl2Pod")
    podMbg3 = os.getenv("podMbg3")
    mbg3Ip = os.getenv("mbg3Ip")
    mbgctl3Pod = os.getenv("mbgctl3Pod")
    
    
    ###Set mbgctl1
    printHeader(f"Create {srcSvc} (client) service in MBG1")
    useKindCluster(mbg1ClusterName)
    runcmd(f"kubectl create -f {folCl}/iperf3-client.yaml")
    waitPod(srcSvc)
    srcSvcIp =getPodIp(srcSvc)
    runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl addService --id {srcSvc} --ip {srcSvcIp}')

    ##Set mbgctl2
    printHeader(f"Add {destSvc} (server) service in MBG2")
    useKindCluster(mbg2ClusterName)
    runcmd(f"kubectl create -f {folSv}/iperf3.yaml")
    waitPod(destSvc)
    destSvcIp = f"{getPodIp(destSvc)}:5000"
    destkindIp=getKindIp(mbg2ClusterName)
    runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl addService --id {destSvc} --ip {destSvcIp} --description iperf3-server')

    ###Set mbgctl3
    printHeader(f"Create {srcSvc} (client) service in MBG3")
    useKindCluster(mbg3ClusterName)
    runcmd(f"kubectl create -f {folCl}/iperf3-client.yaml")
    waitPod(srcSvc)
    srcSvcIp =getPodIp(srcSvc)
    runcmd(f'kubectl exec -i {mbgctl3Pod} -- ./mbgctl addService --id {srcSvc} --ip {srcSvcIp}')

    
    #Expose destination service
    useKindCluster(mbg2ClusterName)
    printHeader("\n\nStart exposing {destSvc} service to MBG2")
    runcmdb(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl expose --serviceId {destSvc}')

    #Get services
    useKindCluster(mbg1ClusterName)
    printHeader("\n\Query service from MBG1")
    runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl getService')

    useKindCluster(mbg3ClusterName)
    printHeader("\n\Query service from MBG3")
    runcmd(f'kubectl exec -i {mbgctl3Pod} -- ./mbgctl getService')

    useKindCluster(mbg2ClusterName)
    waitPod("iperf3-server")
    