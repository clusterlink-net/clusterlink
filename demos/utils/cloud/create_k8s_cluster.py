################################################################
#Name: create_k8s_cluster
#Desc: Create k8s cluster on specific platform and region
#Inputs: cluster_zone, cluster_type, cluster.name ,cluster.platform
#        run_in_bg
################################################################
import os  
import subprocess as sp
#sys.path.insert(1, 'project_metadata/')
from demos.utils.cloud.clusterClass import cluster


############################### functions ##########################
def createCluster(cluster,run_in_bg, machineType="small"):
    print(f"create {cluster.name} cluster , zone {cluster.zone} , platform {cluster.platform}")
    bg_flags = " &" if run_in_bg else ""
    if cluster.platform == "gcp":
        flags = "  --machine-type n2-standard-4" if machineType=="large" else "" #e2-medium
        cmd=f"gcloud container clusters create {cluster.name} --zone {cluster.zone} --num-nodes 1 --tags tcpall {flags} {bg_flags}"
        print(cmd)
        os.system(cmd)
    elif cluster.platform == "aws": #--instance-selector-vcpus 2  --instance-selector-memory 4 --instance-selector-cpu-architecture arm64
        cmd =f"eksctl create cluster --name {cluster.name} --region {cluster.zone} -N 1  {flags}  {bg_flags}"
        print(cmd)
        os.system(cmd)

    elif cluster.platform == "ibm":
        vlan_private_ip=sp.getoutput(f"ibmcloud ks vlans --zone {cluster.zone} |fgrep private |cut -d ' ' -f 1")
        vlan_public_ip=sp.getoutput(f"ibmcloud ks vlans --zone {cluster.zone}  |fgrep public |cut -d ' ' -f 1")
        print("vlan_public_ip:",vlan_public_ip)
        vlan_private_string = "--private-vlan " + vlan_private_ip  if (vlan_private_ip != "" and "FAILED" not in vlan_private_ip) else ""
        if (vlan_public_ip  != "" and "FAILED" not in vlan_public_ip):
            vlan_public_string  = "--public-vlan "  + vlan_public_ip    
        else:
            vlan_public_string= ""
            vlan_private_string = vlan_private_string + " --private-only " if (vlan_private_string != "") else ""
        
        cmd= f"ibmcloud ks cluster create  classic  --name {cluster.name} --zone={cluster.zone} --flavor u3c.2x4 --workers=1 {vlan_private_string} {vlan_public_string} {bg_flags}"
        print(cmd)
        os.system(cmd)
    else:
        print ("ERROR: Cloud platform {} not supported".format(cluster.platform))