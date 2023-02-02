#!/usr/bin/env python3
import os,time
import subprocess as sp
import sys
import argparse

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))

sys.path.insert(0,f'{proj_dir}/tests/')
print(f"{proj_dir}/tests/")
from aux.kindAux import runcmd, runcmdb, printHeader, waitPod, getPodName, getKindIp, getMbgPorts,buildMbg,buildMbgctl,useKindCluster,getPodIp

############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-d','--dataplane', help='choose which dataplane to use mtls/tcp', required=False, default="mtls")
    parser.add_argument('-m','--mbg', help='Either mbg1/mbg2/mbg3', required=False, default="mbg1")
    parser.add_argument('-b','--build', help='Build Image', required=False, default=False)

    args = vars(parser.parse_args())

    dataplane = args["dataplane"]
    mbg       = args["mbg"]
    build     = args["build"]

    print("Starting mbg ("+mbg+") with dataplane "+ dataplane)
    #MBG1 parameters 
    mbg1DataPort    = "30001"
    mbg1cPort       = "30443"
    mbg1cPortLocal  = "8443"
    mbg1crtFlags    = f"--rootCa ./mtls/ca.crt --certificate ./mtls/mbg1.crt --key ./mtls/mbg1.key"  if dataplane =="mtls" else ""
    mbg1ClusterName ="mbg-agent1"
    mbgctl1Name     = "mbgctl1"
    srcSvc          = "iperf3-client"
    srcDefaultGW    = "10.244.0.1"
    srck8sSvcPort   = "5000"
    
    #MBG2 parameters 
    mbg2DataPort    = "30001"
    mbg2cPort       = "30443"
    mbg2cPortLocal  = "8443"
    mbg2crtFlags    = f"--rootCa ./mtls/ca.crt --certificate ./mtls/mbg2.crt --key ./mtls/mbg2.key"  if dataplane =="mtls" else ""
    mbg2ClusterName = "mbg-agent2"
    mbgctl2Name     = "mbgctl2"
    destSvc         = "iperf3-server"
    iperf3DestPort  = "30001"
    
    #MBG3 parameters 
    mbg3DataPort    = "30001"
    mbg3cPort       = "30443"
    mbg3cPortLocal  = "8443"
    mbg3crtFlags    = f"--rootCa ./mtls/ca.crt --certificate ./mtls/mbg3.crt --key ./mtls/mbg3.key"  if dataplane =="mtls" else ""
    mbg3ClusterName = "mbg-agent3"
    mbgctl3Name     = "mbgctl3"
    srcSvc          = "iperf3-client"
    srcDefaultGW    = "10.244.0.1"
    srck8sSvcPort   = "5000"
        
    #print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)

    
    ### build docker environment 
    if build:
        printHeader(f"Building docker image")
        os.system("make docker-build")
    
    
    ### build Kind clusters environment 
    if mbg == "mbg1" :
        print(f"Clean old kinds")
        os.system("kind delete cluster --name=mbg-agent1")
        ###first Mbg
        printHeader("\n\nStart loading MBG1")
        podMbg1, mbg1Ip= buildMbg(mbg1ClusterName,f"{proj_dir}/manifests/kind/mbg-config1.yaml")
        mbgctl1Pod, mbgctl1Ip= buildMbgctl(mbgctl1Name,mbgMode="inside")
        #Set First MBG
        printHeader("\n\nStart MBG1 (along with PolicyEngine)")
        useKindCluster(mbg1ClusterName)
        runcmdb(f'kubectl exec -i {podMbg1} -- ./mbg start --id "MBG1" --ip {mbg1Ip} --cport {mbg1cPort} --cportLocal {mbg1cPortLocal}  --externalDataPortRange {mbg1DataPort}\
        --dataplane {args["dataplane"]} {mbg1crtFlags}')
        runcmd(f"kubectl create service nodeport mbg --tcp={mbg1cPortLocal}:{mbg1cPortLocal} --node-port={mbg1cPort}")
        runcmdb(f'kubectl exec -i {podMbg1} -- ./mbg addPolicyEngine --target {getPodIp(podMbg1)}:9990 --start')
        destMbg1Ip = f"{getPodIp(podMbg1)}:{mbg1cPortLocal}"
        runcmdb(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl start --id {mbgctl1Name}  --ip {mbgctl1Ip} --mbgIP {destMbg1Ip}  --dataplane {args["dataplane"]} {mbg1crtFlags} ')
        runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl addPolicyEngine --target {getPodIp(podMbg1)}:9990')

        f = open(".env", "w")
        f.write("podMbg1="+podMbg1+"\n")
        f.write("mbg1Ip="+mbg1Ip+"\n")
        f.write("mbgctl1Pod="+mbgctl1Pod+"\n")
        f.flush()
        f.close()
    elif  mbg == "mbg2": 
        print(f"Clean old kinds")
        os.system("kind delete cluster --name=mbg-agent2")
        ###Second Mbg
        printHeader("\n\nStart loading MBG2")
        podMbg2, mbg2Ip= buildMbg(mbg2ClusterName,f"{proj_dir}/manifests/kind/mbg-config2.yaml")
        mbgctl2Pod, mbgctl2Ip= buildMbgctl(mbgctl2Name, mbgMode="inside")   
        #Set Second MBG
        printHeader("\n\nStart MBG2 (along with PolicyEngine)")
        useKindCluster(mbg2ClusterName)
        runcmdb(f'kubectl exec -i {podMbg2} -- ./mbg start --id "MBG2" --ip {mbg2Ip} --cport {mbg2cPort} --cportLocal {mbg2cPortLocal} --externalDataPortRange {mbg2DataPort} \
        --dataplane {args["dataplane"]} {mbg2crtFlags}')
        runcmd(f"kubectl create service nodeport mbg --tcp={mbg2cPortLocal}:{mbg2cPortLocal} --node-port={mbg2cPort}")
        runcmdb(f'kubectl exec -i {podMbg2} -- ./mbg addPolicyEngine --target {getPodIp(podMbg2)}:9990 --start')
        destMbg2Ip = f"{getPodIp(podMbg2)}:{mbg2cPortLocal}"  
        runcmdb(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl start --id {mbgctl2Name}  --ip {mbgctl2Ip}  --mbgIP {destMbg2Ip} --dataplane {args["dataplane"]} {mbg2crtFlags}')
        runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl addPolicyEngine --target {getPodIp(podMbg2)}:9990')
        
        f = open(".env", "a")
        f.write("podMbg2="+podMbg2+"\n")
        f.write("mbg2Ip="+mbg2Ip+"\n")
        f.write("mbgctl2Pod="+mbgctl2Pod+"\n")
        f.flush()
        f.close()
    elif mbg == "mbg3": 
        print(f"Clean old kinds")
        os.system("kind delete cluster --name=mbg-agent3")
        ###Third Mbg
        printHeader("\n\nStart loading MBG3")
        podMbg3, mbg3Ip= buildMbg(mbg3ClusterName)
        mbgctl3Pod, mbgctl3Ip= buildMbgctl(mbgctl3Name,mbgMode="inside")
        #Set Third MBG
        printHeader("\n\nStart MBG3 (along with PolicyEngine)")
        useKindCluster(mbg3ClusterName)
        runcmdb(f'kubectl exec -i {podMbg3} --  ./mbg start --id "MBG3" --ip {mbg3Ip} --cport {mbg3cPort} --cportLocal {mbg3cPortLocal} --externalDataPortRange {mbg3DataPort}\
        --dataplane {args["dataplane"]}  {mbg3crtFlags}')
        runcmd(f"kubectl create service nodeport mbg --tcp={mbg3cPortLocal}:{mbg3cPortLocal} --node-port={mbg3cPort}")
        runcmdb(f'kubectl exec -i {podMbg3} -- ./mbg addPolicyEngine --target {getPodIp(podMbg3)}:9990 --start')
        destMbg3Ip = f"{getPodIp(podMbg3)}:{mbg3cPortLocal}"
        runcmdb(f'kubectl exec -i {mbgctl3Pod} -- ./mbgctl start --id {mbgctl3Name}  --ip {mbgctl3Ip} --mbgIP {destMbg3Ip}  --dataplane {args["dataplane"]} {mbg3crtFlags} ')
        runcmd(f'kubectl exec -i {mbgctl3Pod} -- ./mbgctl addPolicyEngine --target {getPodIp(podMbg3)}:9990')
        
        f = open(".env", "a")
        f.write("podMbg3="+podMbg3+"\n")
        f.write("mbg3Ip="+mbg3Ip+"\n")
        f.write("mbgctl3Pod="+mbgctl3Pod+"\n")
        f.flush()
        f.close()
    else:
        print("mbg value should be mbg1, mbg2 or mbg3")
