################################################################
#Name: iperf3_setup
#Desc: Create iperf3 server pod and iperf3 client pod 
#      (deplyment and service from iperf3 folder)
#      Use in phost or targetroxy servers
#
#Inputs: cluster_platform
#TODO -replace it with python script
################################################################
import argparse
import time
import os
import subprocess as sp
from PROJECT_PARAMS import PROJECT_PATH
from cr_aux_func import *
from tests.utils.cloud.clusterClass import cluster
parser = argparse.ArgumentParser()
from meta_data_func import update_metadata


#Setting iPerf3 target
def setupIperf3Target(platform):
    print("\n\ncreate iperf3 deploymnet and client")
    os.system(f"kubectl create -f {PROJECT_PATH}/config/manifests/iperf3/iperf3.yaml")
    os.system(f"kubectl create -f {PROJECT_PATH}/config/manifests/host/iperf3-client.yaml")


    iperf3_start_cond=False
    while( not iperf3_start_cond):
        iperf3_start_cond =sp.getoutput("kubectl get pods -l app=iperf3-server -o jsonpath='{.items[0].status.containerStatuses[0].ready}'")
        print(iperf3_start_cond)
        print ("Waiting for iperf3-server  start...")
        time.sleep(5)

    print("iperf3 Server is running")
    #Creating iperf3-svc will be reeady
    os.system(f"kubectl  create -f {PROJECT_PATH}/config/manifests/iperf3/iperf3-svc.yaml")
    #


    external_ip=""
    while external_ip =="":
        print("Waiting for iperf3 LoadBalancer...")
        if platform == "aws":
            print("set iperf3 in AWS platform")
            time.sleep(30)
            external_addr =sp.getoutput('kubectl describe svc iperf3-loadbalancer-service | fgrep "Ingress" | cut -d ":" -f 2')
            count_addr =sp.getoutput(f'nslookup {external_addr} | fgrep Address | wc -l')
            print(f'external_addr: {external_addr} ,count_addr: {count_addr}')
            if (int(count_addr) > 1):
                external_ip =sp.getoutput(f'nslookup {external_addr} |'+" awk '/Address/ { addr[cnt++]=$2 } END { print addr[1] }'")
        else:
            external_ip=sp.getoutput('kubectl get svc  iperf3-loadbalancer-service --template="{{range .status.loadBalancer.ingress}}{{.ip}}{{end}}"')
        time.sleep(10)
    print("Iperf3 LoadBalancer is ready, external_id: {}".format(external_ip))

#Setting Host mode
def setupIperf3Host(platform):
    os.system(f"kubectl create -f {PROJECT_PATH}/config/manifests/host/iperf3-client.yaml")
    container_reg = get_plarform_container_reg(platform)
    os.system(f"docker tag mbg:latest {container_reg}/mbg:latest")
    os.system(f"docker push {container_reg}/mbg:latest")



def iperf3Test(target_ip,target_port,time=10):
    pod= sp.getoutput('kubectl get pods -l app=iperf3-client -o name | head -n 1| cut -d\'/\' -f2')
    cmd='kubectl get pod {} '.format(pod) +'-o jsonpath=\'{.status.containerStatuses[0].ready}\''
    pod_status= sp.getoutput(cmd)
    while (not pod_status):
        print("Waiting for {} to start...".format(pod))
        time.sleep(5)

    cmd = 'kubectl exec -i {} -- iperf3 -c {} -p {} -t {}'.format(pod,target_ip,target_port,time)
    print("\nRUN: {}\n".format(cmd))

    os.system(cmd)

def iperf3TestDB(target_ip,target_port, resFile,time=10):
    pod= sp.getoutput('kubectl get pods -l app=iperf3-client -o name | head -n 1| cut -d\'/\' -f2')
    cmd='kubectl get pod {} '.format(pod) +'-o jsonpath=\'{.status.containerStatuses[0].ready}\''
    pod_status= sp.getoutput(cmd)
    while (not pod_status):
        print("Waiting for {} to start...".format(pod))
        time.sleep(5)

    cmd = 'kubectl exec -i {} -- iperf3 -c {} -p {} -t {} -J > {}'.format(pod,target_ip,target_port,time,resFile)
    print("\nRUN: {}\n".format(cmd))

    os.system(cmd)
############################### MAIN ##########################
if __name__ == "__main__":
    #Parser
    parser.add_argument("-platform"    , "--cluster_platform", default = "gcp"         , help="setting k8s cloud platform")

    args = parser.parse_args()
    platform    = args.cluster_platform
    setupIperf3Target(platform)