import os,time
import subprocess as sp
import sys
import argparse

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))

sys.path.insert(0,f'{proj_dir}/tests/')
print(f"{proj_dir}/tests/")
from aux.kindAux import runcmd, runcmdb, printHeader, waitPod, getPodName, getKindIp, getMbgPorts,buildMbg,buildMbgctl,useKindCluster,getPodIp

def iperf3Test(cmd):
    print(cmd)
    direct_output = sp.check_output(cmd,shell=True) #could be anything here.  
    printHeader(f"Iperf3 Test Results:\n") 
    print(f"{direct_output.decode()}")
    print("***************************************")
    if "iperf Done" in direct_output.decode():
        print(f'Test Pass')
    else:
        print(f'Test Fail')
    print("***************************************")


############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-d','--dataplane', help='choose which dataplane to use mtls/tcp', required=False, default="tcp")
    parser.add_argument('-m','--mbgmode', help='mbg mode inside or outside the cluste', required=False, default="inside")
    args = vars(parser.parse_args())

    printHeader("\n\nStart Kind Test\n\n")
    printHeader("Start pre-setting")

    dataplane = args["dataplane"]
    mbgMode= args["mbgmode"]
    #MBG1 parameters 
    mbg1DataPort    = "30001"
    mbg1cPort       = "30443"
    mbg1cPortLocal  = "8443"
    mbg1crtFlags    = f"--rootCa ./mtls/ca.crt --certificate ./mtls/mbg1.crt --key ./mtls/mbg1.key"  if dataplane =="mtls" else ""
    mbg1ClusterName ="mbg-agent1"
    
    #MBG2 parameters 
    mbg2DataPort    = "30001"
    mbg2cPort       = "30443"
    mbg2cPortLocal  = "8443"
    mbg2crtFlags    = f"--rootCa ./mtls/ca.crt --certificate ./mtls/mbg2.crt --key ./mtls/mbg2.key"  if dataplane =="mtls" else ""
    mbg2ClusterName = "mbg-agent2"
    #Host parameters
    srcSvc          = "iperf3-client"
    srcDefaultGW    = "10.244.0.1"
    srck8sSvcPort   = "5000"
    
    hostcrtFlags    = f"--rootCa ./mtls/ca.crt --certificate ./mtls/mbg1.crt --key ./mtls/mbg1.key"  if dataplane =="mtls" else ""
    hostClusterName = "mbg-agent1" if mbgMode =="inside" else"host-cluster"
    #Destination parameters
    destSvc         = "iperf3-server"
    iperf3DestPort  = "30001"
    destcrtFlags    = f"--rootCa ./mtls/ca.crt --certificate ./mtls/mbg2.crt --key ./mtls/mbg2.key"  if dataplane =="mtls" else ""
    destClusterName = "mbg-agent2" if mbgMode =="inside" else "dest-cluster" 
    
    #folders
    folCl=f"{proj_dir}/tests/iperf3/manifests/iperf3-client"
    folSv=f"{proj_dir}/tests/iperf3/manifests/iperf3-server"
    
    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)
    ### clean 
    print(f"Clean old kinds")
    os.system("make clean-kind-iperf3")
    
    ### build docker environment 
    printHeader(f"Build docker image")
    os.system("make docker-build")
    
    
    ### build Kind clusters environment 
    ###first Mbg
    printHeader("\n\nStart building MBG1")
    podMbg1, mbg1Ip= buildMbg(mbg1ClusterName,f"{proj_dir}/manifests/kind/mbg-config1.yaml")
    ###Second Mbg
    printHeader("\n\nStart building MBG2")
    podMbg2, mbg2Ip= buildMbg(mbg2ClusterName,f"{proj_dir}/manifests/kind/mbg-config2.yaml")
    if mbgMode !="inside":
        ###host cluster
        printHeader("\n\nStart building host-cluster")
        runcmd(f"kind create cluster --config {folCl}/kind-config.yaml --name=host-cluster")
        runcmd(f"kind load docker-image mbg --name={hostClusterName}")
        #dest cluster
        printHeader("\n\nStart building dest-cluster")
        runcmd(f"kind create cluster --config {folSv}/kind-config.yaml --name=dest-cluster")
        runcmd(f"kind load docker-image mbg --name={destClusterName}")
    
    #Set First MBG
    useKindCluster(mbg1ClusterName)
    runcmdb(f'kubectl exec -i {podMbg1} -- ./mbg start --id "MBG1" --ip {mbg1Ip} --cport {mbg1cPort} --cportLocal {mbg1cPortLocal}  --externalDataPortRange {mbg1DataPort}\
    --dataplane {args["dataplane"]} {mbg1crtFlags}')
    runcmd(f"kubectl create service nodeport mbg --tcp={mbg1cPortLocal}:{mbg1cPortLocal} --node-port={mbg1cPort}")

    #Set Second MBG
    useKindCluster(mbg2ClusterName)
    runcmdb(f'kubectl exec -i {podMbg2} -- ./mbg start --id "MBG2" --ip {mbg2Ip} --cport {mbg2cPort} --cportLocal {mbg2cPortLocal} --externalDataPortRange {mbg2DataPort} \
    --dataplane {args["dataplane"]} {mbg2crtFlags}')
    runcmd(f"kubectl create service nodeport mbg --tcp={mbg2cPortLocal}:{mbg2cPortLocal} --node-port={mbg2cPort}")
    
    ###Set host
    useKindCluster(hostClusterName)
    runcmd(f"kubectl create -f {folCl}/iperf3-client.yaml")
    podhost, hostIp= buildMbgctl("host Cluster",mbgMode)
    hostMbgIp = f"{getPodIp(podMbg1)}:{mbg1cPortLocal}" if mbgMode =="inside" else f"{mbg1Ip}:{mbg1cPort}"
    runcmdb(f'kubectl exec -i {podhost} -- ./mbgctl start --id "hostCluster"  --ip {hostIp} --mbgIP {hostMbgIp}  --dataplane {args["dataplane"]} {hostcrtFlags} ')
    printHeader(f"Add {srcSvc} (client) service to host cluster")
    waitPod(srcSvc)
    srcSvcIp =getPodIp(srcSvc)  if mbgMode =="inside" else srcDefaultGW
    runcmd(f'kubectl exec -i {podhost} -- ./mbgctl addService --serviceId {srcSvc} --serviceIp {srcSvcIp}')

    # Add MBG Peer
    printHeader("Add MBG2 peer to MBG1")
    runcmd(f'kubectl exec -i {podhost} -- ./mbgctl addPeer --id "MBG2" --ip {mbg2Ip} --cport {mbg2cPort}')
    
    # Send Hello
    printHeader("Send Hello commands")
    runcmd(f'kubectl exec -i {podhost} -- ./mbgctl hello')
    
    ##Set dest
    useKindCluster(destClusterName)
    runcmd(f"kubectl create -f {folSv}/iperf3.yaml")
    podest, destIp= buildMbgctl("dest Cluster",mbgMode)   
    runcmd(f"kubectl create service nodeport iperf3-server --tcp=5000:5000 --node-port={iperf3DestPort}")
    destMbgIp = f"{getPodIp(podMbg2)}:{mbg2cPortLocal}" if mbgMode =="inside" else f"{mbg2Ip}:{mbg2cPort}"
    runcmdb(f'kubectl exec -i {podest} -- ./mbgctl start --id "destCluster"  --ip {destIp}  --mbgIP {destMbgIp} --dataplane {args["dataplane"]} {destcrtFlags}')
    printHeader(f"Add {destSvc} (server) service to destination cluster")
    waitPod(destSvc)
    destSvcIp = f"{getPodIp(destSvc)}:5000" if mbgMode =="inside" else f"{destIp}:{iperf3DestPort}"
    destkindIp=getKindIp(destClusterName)
    runcmd(f'kubectl exec -i {podest} -- ./mbgctl addService --serviceId {destSvc} --serviceIp {destSvcIp}')

    #Add host cluster to MBG1
    useKindCluster(hostClusterName)
    printHeader("Add host cluster to MBG1")
    runcmd(f'kubectl exec -i {podMbg1} -- ./mbg addMbgctl --id "hostCluster" --ip {hostIp}')

    #Add dest cluster to MBG2
    useKindCluster(mbg2ClusterName)
    printHeader("Add dest cluster to MBG2")
    runcmd(f'kubectl exec -i {podMbg2} -- ./mbg addMbgctl --id "destCluster" --ip {destIp}')

    #Expose destination service
    useKindCluster(destClusterName)
    printHeader("\n\nStart exposing connection")
    runcmdb(f'kubectl exec -i {podest} -- ./mbgctl expose --serviceId {destSvc}')

    #Get services
    useKindCluster(hostClusterName)
    printHeader("\n\nStart get service")
    runcmdb(f'kubectl exec -i {podhost} -- ./mbgctl getService')
    # Create Nodeport inside mbg1
    useKindCluster(mbg1ClusterName)
    mbg1LocalPort, mbg1ExternalPort = getMbgPorts(podMbg1,destSvc)
    
    if mbgMode !="inside":
        runcmd(f"kubectl create service nodeport {srcSvc} --tcp={mbg1LocalPort}:{mbg1LocalPort} --node-port={mbg1ExternalPort}")
        runcmd(f"kubectl patch service {srcSvc} -p "+  "\'{\"spec\":{\"selector\":{\"app\": \"mbg\"}}}\'") #replacing app name

        # Create connect from cluster to MBG
        printHeader(f"\n\nStart Data plan connection {srcSvc} to {destSvc}")
        useKindCluster(hostClusterName)
        runcmd(f"kubectl create -f {folCl}/iperf3-svc.yaml")
        runcmdb(f'kubectl exec -i {podhost} -- ./mbgctl connect --serviceId {srcSvc} --serviceIp :{srck8sSvcPort} --serviceIdDest {destSvc}')


    #Testing
    printHeader("\n\nStart Iperf3 testing")
    useKindCluster(destClusterName)
    waitPod("iperf3-server")
    useKindCluster(hostClusterName)
    waitPod("iperf3-client")
    podIperf3= getPodName("iperf3-clients")

    printHeader("The Iperf3 test connects directly to the destination")
    cmd = f'kubectl exec -i {podIperf3} --  iperf3 -c {destkindIp} -p {iperf3DestPort}'
    iperf3Test(cmd)
    
    if mbgMode !="inside":
        printHeader("The Iperf3 test connects to MBG1")
        cmd = f'kubectl exec -i {podIperf3} --  iperf3 -c {mbg1Ip} -p {mbg1ExternalPort}'
        iperf3Test(cmd)
    
    printHeader("Full Iperf3 test clinet-> MBG1-> MBG2-> dest")
    testport =  mbg1LocalPort if mbgMode =="inside" else srck8sSvcPort
    testip   =  getPodIp(podMbg1) if mbgMode =="inside" else "mbgctl-iperf3-service"
    cmd = f'kubectl exec -i {podIperf3} --  iperf3 -c {testip} -p {testport}'
    iperf3Test(cmd)

    #Close connection
    printHeader("\n\nClose Iperf3 connection")
    runcmd(f'kubectl exec -i {podhost} -- ./mbgctl disconnect --serviceId {srcSvc} --serviceIdDest {destSvc}')