import os,time
import subprocess as sp
import sys
import argparse

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))

sys.path.insert(0,f'{proj_dir}/tests/')
print(f"{proj_dir}/tests/")
from aux.kindAux import runcmd, runcmdb, printHeader, waitPod, getPodName, getKindIp, getMbgPorts,buildMbg,buildMbgctl,useKindCluster,getPodIp


def iperf3Test(cmd ,blockFlag=False):
    print(cmd)
    testPass=False
    try:
        direct_output = sp.check_output(cmd,shell=True) #could be anything here.  
        printHeader(f"Iperf3 Test Results:\n") 
        print(f"{direct_output.decode()}")
        if "iperf Done" in direct_output.decode():
            testPass=True
    
    except sp.CalledProcessError as e:
        print(f"Test Code:{e.returncode}")
        if blockFlag and e.returncode == 1:
            testPass =True
            printHeader(f"Test block succeed") 

    print("***************************************")
    if testPass:
        print(f'Test Pass')
    else:
        print(f'Test Fail')
    print("***************************************")


############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-d','--dataplane', help='choose which dataplane to use mtls/tcp', required=False, default="tcp")
    args = vars(parser.parse_args())

    printHeader("\n\nStart Kind Test\n\n")
    printHeader("Start pre-setting")

    dataplane = args["dataplane"]
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
    ###Third Mbg
    printHeader("\n\nStart building MBG3")
    podMbg3, mbg3Ip= buildMbg(mbg3ClusterName)

    #Set First MBG
    useKindCluster(mbg1ClusterName)
    runcmdb(f'kubectl exec -i {podMbg1} -- ./mbg start --id "MBG1" --ip {mbg1Ip} --cport {mbg1cPort} --cportLocal {mbg1cPortLocal}  --externalDataPortRange {mbg1DataPort}\
    --dataplane {args["dataplane"]} {mbg1crtFlags}')
    runcmd(f"kubectl create service nodeport mbg --tcp={mbg1cPortLocal}:{mbg1cPortLocal} --node-port={mbg1cPort}")
    runcmdb(f'kubectl exec -i {podMbg1} -- ./mbg addPolicyEngine --target {getPodIp(podMbg1)}:9990 --start')


    #Set Second MBG
    useKindCluster(mbg2ClusterName)
    runcmdb(f'kubectl exec -i {podMbg2} -- ./mbg start --id "MBG2" --ip {mbg2Ip} --cport {mbg2cPort} --cportLocal {mbg2cPortLocal} --externalDataPortRange {mbg2DataPort} \
    --dataplane {args["dataplane"]} {mbg2crtFlags}')
    runcmd(f"kubectl create service nodeport mbg --tcp={mbg2cPortLocal}:{mbg2cPortLocal} --node-port={mbg2cPort}")
    runcmdb(f'kubectl exec -i {podMbg2} -- ./mbg addPolicyEngine --target {getPodIp(podMbg2)}:9990 --start')


    #Set Third MBG
    useKindCluster(mbg3ClusterName)
    runcmdb(f'kubectl exec -i {podMbg3} --  ./mbg start --id "MBG3" --ip {mbg3Ip} --cport {mbg3cPort} --cportLocal {mbg3cPortLocal} --externalDataPortRange {mbg3DataPort}\
    --dataplane {args["dataplane"]}  {mbg3crtFlags}')
    runcmd(f"kubectl create service nodeport mbg --tcp={mbg3cPortLocal}:{mbg3cPortLocal} --node-port={mbg3cPort}")
    runcmdb(f'kubectl exec -i {podMbg3} -- ./mbg addPolicyEngine --target {getPodIp(podMbg3)}:9990 --start')
    
        
    ###Set mbgctl1
    useKindCluster(mbg1ClusterName)
    runcmd(f"kubectl create -f {folCl}/iperf3-client.yaml")
    mbgctl1Pod, mbgctl1Ip= buildMbgctl(mbgctl1Name,mbgMode="inside")
    destMbg1Ip = f"{getPodIp(podMbg1)}:{mbg1cPortLocal}"
    runcmdb(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl start --id {mbgctl1Name}  --ip {mbgctl1Ip} --mbgIP {destMbg1Ip}  --dataplane {args["dataplane"]} {mbg1crtFlags} ')
    printHeader(f"Add {srcSvc} (client) service to host cluster")
    waitPod(srcSvc)
    srcSvcIp =getPodIp(srcSvc)
    runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl addService --id {srcSvc} --ip {srcSvcIp}')
    runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl addPolicyEngine --target {getPodIp(podMbg1)}:9990')

    ##Set mbgctl2
    useKindCluster(mbg2ClusterName)
    runcmd(f"kubectl create -f {folSv}/iperf3.yaml")
    mbgctl2Pod, mbgctl2Ip= buildMbgctl(mbgctl2Name, mbgMode="inside")   
    destMbg2Ip = f"{getPodIp(podMbg2)}:{mbg2cPortLocal}"    
    runcmd(f"kubectl create service nodeport iperf3-server --tcp=5000:5000 --node-port={iperf3DestPort}")
    runcmdb(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl start --id {mbgctl2Name}  --ip {mbgctl2Ip}  --mbgIP {destMbg2Ip} --dataplane {args["dataplane"]} {mbg2crtFlags}')
    printHeader(f"Add {destSvc} (server) service to destination cluster")
    waitPod(destSvc)
    destSvcIp = f"{getPodIp(destSvc)}:5000"
    destkindIp=getKindIp(mbg2ClusterName)
    runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl addService --id {destSvc} --ip {destSvcIp} --description iperf3-server')
    runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl addPolicyEngine --target {getPodIp(podMbg2)}:9990')
    
    ###Set mbgctl3
    useKindCluster(mbg3ClusterName)
    runcmd(f"kubectl create -f {folCl}/iperf3-client.yaml")
    mbgctl3Pod, mbgctl3Ip= buildMbgctl(mbgctl3Name,mbgMode="inside")
    destMbg3Ip = f"{getPodIp(podMbg3)}:{mbg3cPortLocal}"
    runcmdb(f'kubectl exec -i {mbgctl3Pod} -- ./mbgctl start --id {mbgctl3Name}  --ip {mbgctl3Ip} --mbgIP {destMbg3Ip}  --dataplane {args["dataplane"]} {mbg3crtFlags} ')
    printHeader(f"Add {srcSvc} (client) service to host cluster")
    waitPod(srcSvc)
    srcSvcIp =getPodIp(srcSvc)
    runcmd(f'kubectl exec -i {mbgctl3Pod} -- ./mbgctl addService --id {srcSvc} --ip {srcSvcIp}')
    runcmd(f'kubectl exec -i {mbgctl3Pod} -- ./mbgctl addPolicyEngine --target {getPodIp(podMbg3)}:9990')
    
    #Add host cluster to MBG1
    useKindCluster(mbg1ClusterName)
    printHeader("Add mbgctl to MBG1")
    runcmd(f'kubectl exec -i {podMbg1} -- ./mbg addMbgctl --id {mbgctl1Name} --ip {mbgctl1Ip}')

    #Add dest cluster to MBG2
    useKindCluster(mbg2ClusterName)
    printHeader("Add mbgctl2 to MBG2")
    runcmd(f'kubectl exec -i {podMbg2} -- ./mbg addMbgctl --id {mbgctl2Name} --ip {mbgctl2Ip}')
    
    #Add dest cluster to MBG3
    useKindCluster(mbg3ClusterName)
    printHeader("Add mbgctl3 to MBG3")
    runcmd(f'kubectl exec -i {podMbg3} -- ./mbg addMbgctl --id {mbgctl3Name} --ip {mbgctl3Ip}')
    
    # Add MBG Peer
    useKindCluster(mbg2ClusterName)
    printHeader("Add MBG2, MBG3 peer to MBG1")
    runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl addPeer --id "MBG1" --ip {mbg1Ip} --cport {mbg1cPort}')
    runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl addPeer --id "MBG3" --ip {mbg3Ip} --cport {mbg3cPort}')
        
    # Send Hello
    printHeader("Send Hello commands")
    runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl hello')
    
    #Expose destination service
    useKindCluster(mbg2ClusterName)
    printHeader("\n\nStart exposing connection")
    runcmdb(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl expose --serviceId {destSvc}')

    #Get services
    useKindCluster(mbg1ClusterName)
    printHeader("\n\nStart get service")
    runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl getService')
    
    #Testing
    printHeader("\n\nStart Iperf3 testing")
    useKindCluster(mbg2ClusterName)
    waitPod("iperf3-server")
    
    #Test MBG1
    useKindCluster(mbg1ClusterName)
    waitPod("iperf3-client")
    podIperf3= getPodName("iperf3-clients")
    mbg1LocalPort, mbg1ExternalPort = getMbgPorts(podMbg1,destSvc+"-MBG2")

    printHeader("The Iperf3 test connects directly to the destination")
    cmd = f'kubectl exec -i {podIperf3} --  iperf3 -c {destkindIp} -p {iperf3DestPort}'
    iperf3Test(cmd)

    printHeader("Full Iperf3 test clinet-> MBG1-> MBG2-> dest")
    cmd = f'kubectl exec -i {podIperf3} --  iperf3 -c {getPodIp(podMbg1) } -p {mbg1LocalPort}'
    iperf3Test(cmd)

    #Test MBG3
    useKindCluster(mbg3ClusterName)
    waitPod("iperf3-client")
    podIperf3= getPodName("iperf3-clients")
    mbg3LocalPort, mbg3ExternalPort = getMbgPorts(podMbg3,destSvc+"-MBG2")

    printHeader("The Iperf3 test connects directly to the destination")
    cmd = f'kubectl exec -i {podIperf3} --  iperf3 -c {destkindIp} -p {iperf3DestPort}'
    iperf3Test(cmd)

    printHeader("Full Iperf3 test clinet-> MBG1-> MBG2-> dest")
    cmd = f'kubectl exec -i {podIperf3} --  iperf3 -c {getPodIp(podMbg3)} -p {mbg3LocalPort}'
    iperf3Test(cmd)

    #Block Traffic in MBG3
    printHeader("Start Block Traffic in MBG3")
    print("Block Traffic in MBG3")
    runcmd(f'kubectl exec -i {mbgctl3Pod} -- ./mbgctl policy --command acl_add --serviceSrc {srcSvc} --serviceDst {destSvc} --mbgDest MBG2 --priority 0 --action 1')
    cmd = f'kubectl exec -i {podIperf3} --  iperf3 -c {getPodIp(podMbg3)} -p {mbg3LocalPort}'
    iperf3Test(cmd, blockFlag=True)
    print("Allow Traffic in MBG3")
    runcmd(f'kubectl exec -i {mbgctl3Pod} -- ./mbgctl policy --command acl_del --serviceSrc {srcSvc} --serviceDst {destSvc} --mbgDest MBG2 --priority 0 --action 1')
    cmd = f'kubectl exec -i {podIperf3} --  iperf3 -c {getPodIp(podMbg3)} -p {mbg3LocalPort}'
    iperf3Test(cmd)
    
    #Block Traffic in MBG2
    printHeader("Start Block Traffic in MBG2")
    print("Block Traffic in MBG2")
    useKindCluster(mbg2ClusterName)
    runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl policy --command acl_add --mbgDest MBG3 --priority 0 --action 1')
    useKindCluster(mbg3ClusterName)
    cmd = f'kubectl exec -i {podIperf3} --  iperf3 -c {getPodIp(podMbg3)} -p {mbg3LocalPort}'
    iperf3Test(cmd, blockFlag=True)
    useKindCluster(mbg2ClusterName)
    runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl policy --command acl_del --mbgDest MBG3 --priority 0 --action 1')
    useKindCluster(mbg3ClusterName)
    cmd = f'kubectl exec -i {podIperf3} --  iperf3 -c {getPodIp(podMbg3)} -p {mbg3LocalPort}'
    iperf3Test(cmd)
    

    # #Close connection
    # #printHeader("\n\nClose Iperf3 connection")
    # #runcmd(f'kubectl exec -i {mbgctl3Pod} -- ./mbgctl disconnect --serviceId {srcSvc} --serviceIdDest {destSvc}')