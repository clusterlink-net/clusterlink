################################################################
#Name: set_k8s_cluster
#Desc: set the k8s cluster on the server according to his type:
#      host/target - iperf3 deployment
#      proxy       - forwarding/haproxy deployment
#Inputs: cluster_zone, cluster_name ,cluster_platform, cluster_type ,proxy_target_name
#       
################################################################

import imp
import os  
import subprocess as sp
import sys
sys.path.insert(1,os.path.dirname(os.path.dirname(os.path.realpath(__file__))))

from iperf3_setup import setupIperf3Target,setupIperf3Host
from mbg_setup import serviceNodeSetup

from PROJECT_PARAMS import METADATA_FILE,PROJECT_PATH
from meta_data_func import update_metadata, getIpPort

try:
    from typing import runtime_checkable
except ImportError:
    from typing_extensions import runtime_checkable
import argparse
parser = argparse.ArgumentParser()

############################### functions ##########################

def deployTarget(cluster):
    #creating iperf3
    if cluster.platform == "aws":
        setupIperf3Target(cluster.platform)
        target_ip= sp.getoutput('nslookup `kubectl describe svc iperf3-loadbalancer-service |fgrep "Ingress"| cut -d \':\' -f 2` |awk \'/Address/ { addr[cnt++]=$2 } END { print addr[1] }\'')
    else:#gcp/ibm
        setupIperf3Target(cluster.platform)
        target_ip= sp.getoutput('kubectl get svc iperf3-loadbalancer-service --template="{{range .status.loadBalancer.ingress}}{{.ip}}{{end}}"')
    target_port = "5500"
    data_dic={"ip": target_ip, "port" : target_port}
    dicUpdate(data_dic, cluster)
    return data_dic

def deployHost(cluster):
    #creating iperf3
    if cluster.platform == "aws":
        os.system(f"python3 ./steps/iperf3_setup.py -platform {cluster.platform}")
        host_ip= sp.getoutput('nslookup `kubectl describe svc iperf3-loadbalancer-service |fgrep "Ingress"| cut -d \':\' -f 2` |awk \'/Address/ { addr[cnt++]=$2 } END { print addr[1] }\'')
    else: #gcp/ibm
        setupIperf3Host(cluster.platform)

    data_dic = {"ip": "", "port" : ""}
    dicUpdate(data_dic, cluster)
    return data_dic
    
def deployServiceNode(cluster):
    
    # creating mbg ip
    print("\n Setup mbg")
    mbg_ip=serviceNodeSetup(platform=cluster.platform)
    
    mbg_port="30000"
    
    data_dic={"mbg_ip": mbg_ip, "mbg_port" : mbg_port}
    dicUpdate(data_dic, cluster)
    return data_dic

def dicUpdate(data_dic, cluster):
    #update meta_data file
    data_dic.update({"cluster_zone" :  cluster.zone })
    data_dic.update({"cluster_type" :  cluster.type })
    data_dic.update({"cluster_name" :  cluster.name })
    data_dic.update({"cluster_platform" :  cluster.platform })
    cluster_key=cluster.name+"_"+cluster.zone
    update_metadata(METADATA_FILE,cluster_key ,data_dic)


def setupClientService(mbg, target,service):
    print("\n\ncreate client configmap deploymnet and service")
    cleanService()
    #createClientConfigFile(PROJECT_PATH+"/manifests/host/gateway-configmap.yaml", mbg, target,service)
    os.system(f"kubectl create -f {PROJECT_PATH}/manifests/host/cluster.yaml")
    os.system(f"kubectl create -f {PROJECT_PATH}/manifests/host/cluster-svc.yaml")


def createClientConfigFile(file, mbg, target, service):
#file creating
    mbg_ip,mbg_port         = getIpPort(file=PROJECT_PATH+"/bin/metadata.json", cluster = mbg)
    target_ip,target_port = getIpPort(file=PROJECT_PATH+"/bin/metadata.json", cluster = target)


    print(f"Start creating cfg map mbg_ip: {mbg_ip} target ip: {target_ip}:{target_port}")

    f = open(file, "w")
    f.write("apiVersion: v1\n")
    f.write("kind: ConfigMap\n")
    f.write("metadata:\n")
    f.write("  name: gateway-config\n")
    f.write("data:\n")
    f.write(f"  app.mbg_ip: \"{mbg_ip + ':' + mbg_port}\"\n")
    f.write(f"  app.dest_ip: \"{target_ip}\"\n")
    f.write(f"  app.dest_port: \"{target_port}\"\n")
    f.write(f"  app.service: \"{service}\"\n")
    
    f.close()

    print("Finish creating client-configmap.yaml ")
def cleanService():
    os.system(f"kubectl delete --all deployments")
    os.system(f"kubectl delete configmap client-config")

############################### MAIN ##########################
if __name__ == "__main__":

    #Parser
    parser.add_argument("-zone"        , "--cluster_zone"    , default  = "us-east1-b" , help="setting k8s cluster zone")
    parser.add_argument("-type"        , "--cluster_type"    , default  = "host"       , help="setting k8s cluster typw")
    parser.add_argument("-name"        , "--cluster_name"    , default  = ""          , help="setting k8s cluster name")
    parser.add_argument("-platform"    , "--cluster_platform", default = "gcp"         , help="setting k8s cloud platform")
    parser.add_argument("-p_target"    , "--proxy_target_name" , default  = "target-k8s" , help="getting proxy target name")
    parser.add_argument("-f_target_ip" , "--forward_target_ip"   , default  = "" , help="getting forwarding target ip")
    parser.add_argument("-f_target_port","--forward_target_port" , default  = "" , help="getting forwarding target port")
    parser.add_argument("-p_target_ip" , "--proxy_target_ip"   , default  = "" , help="getting proxy target ip")
    parser.add_argument("-p_target_port","--proxy_target_port" , default  = "" , help="getting proxy target port")


    args = parser.parse_args()
    cluster_zone   = args.cluster_zone
    cluster_type   = args.cluster_type
    cluster_platform  = args.cluster_platform


    if (args.cluster_name == ""):
        cluster_name   = "host-k8s" if (cluster_type == "host") else ("target-k8s" if (cluster_type == "target") else "proxy-k8s" )
    else:
        cluster_name  = args.cluster_name

    connect_to_cluster(cluster_name, cluster_zone, cluster_platform)

    if (cluster_type == "host") :
        data_dic  = deploy_host(cluster_zone, cluster_type, cluster_name, cluster_platform)
    elif (cluster_type == "target"):
        data_dic  = deploy_target(cluster_zone, cluster_type, cluster_name, cluster_platform)
    else: # cluster_type == "proxy-k8s" 
        data_dic  = deploy_proxy(cluster_zone, cluster_type, cluster_name, cluster_platform,args.proxy_target_name,\
                                args.proxy_target_ip, args.proxy_target_port, args.forward_target_ip, args.forward_target_port)


