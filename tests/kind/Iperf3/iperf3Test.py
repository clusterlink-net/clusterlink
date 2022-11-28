import os,sys,time,json
import subprocess as sp
import netifaces as ni
from colorama import Fore
from colorama import Style

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))

def waitPod(name):
    start_cond="false"
    time.sleep(3) #Initial start
    while(start_cond != "true"):
        cmd=f"kubectl get pods -l app={name} -o jsonpath" + "=\'{.items[0].status.containerStatuses[0].ready}\'"
        start_cond =sp.getoutput(cmd)
        if (start_cond != "true"):
            print (f"Waiting for pod  {name} to start")
            time.sleep(7)
        else:
            time.sleep(5)
            break

def getPodName(prefix):
    podName=sp.getoutput(f'kubectl get pods -o name | fgrep {prefix}| cut -d\'/\' -f2')
    return podName

def runcmd(cmd):
    print(cmd)
    #sp.Popen(cmd,shell=True)
    os.system(cmd)

def runcmdb(cmd):
    print(cmd)
    #sp.Popen(cmd,shell=True)
    os.system(cmd + ' &')
    time.sleep(7)

def getKindIp(name):
    clJson=json.loads (sp.getoutput(f' kubectl get nodes -o json'))
    ip = clJson["items"][0]["status"]["addresses"][0]["address"]
    print(f"Kind Cluster {name} ip address:{ip}")
    return ip

def printHeader(msg):
    print(f'{Fore.BLUE}{msg} {Style.RESET_ALL}')
    #print(msg)

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
    printHeader("\n\nStart Kind Test\n\n")
    printHeader("Start pre-setting")

    iperf3DestPort="30001"
    mbg1DataPort= "30001"
    mbg2DataPort= "30001"
    srcSvc ="iperfIsrael"
    destSvc ="iperfIndia"

    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)
    ### clean 
    print(f"Clean old kinds")
    os.system("make clean-kind-mbg")
    
    ### build docker environment 
    printHeader(f"Build docker image")
    os.system("make docker-build")
    ###Run first Mbg
    printHeader("\n\nStart building MBG1")
    os.system("make run-kind-mbg1")
    waitPod("mbg")
    podMbg1= getPodName("mbg")
    mbg1Ip=getKindIp("MBG1")
    runcmdb(f'kubectl exec -i {podMbg1} -- ./mbg start --id "MBG1" --ip {mbg1Ip} --cport "30000" --externalDataPortRange {mbg1DataPort}')
    
    ###Run Second Mbg
    printHeader("\n\nStart building MBG2")
    os.system("make run-kind-mbg2")
    waitPod("mbg")
    podMbg2 = getPodName("mbg")
    mbg2Ip=getKindIp("MBG2")

    runcmdb(f'kubectl exec -i {podMbg2} --  ./mbg start --id "MBG2" --ip {mbg2Ip} --cport "30000" --externalDataPortRange {mbg2DataPort}')
    printHeader("Add MBG1 neighbor to MBG2")
    runcmd(f'kubectl exec -i {podMbg2} -- ./mbg addMbg --id "MBG1" --ip {mbg1Ip} --cport "30000"')
    printHeader("Send Hello commands")
    runcmd(f'kubectl exec -i {podMbg2} -- ./mbg hello')
    
    ###Run host
    printHeader("\n\nStart building cluster-host")
    os.system("make run-kind-host")
    waitPod("cluster-mbg")
    podhost= getPodName("cluster-mbg")
    hostIp=getKindIp("hostCluster")
    runcmdb(f'kubectl exec -i {podhost} -- ./cluster start --id "hostCluster"  --ip {hostIp} --cport 30000 --mbgIP {mbg1Ip}:30000')
    printHeader(f"Add {srcSvc} (client) service to host cluster")
    runcmd(f'kubectl exec -i {podhost} -- ./cluster addService --serviceId {srcSvc} --serviceIp :5000')
    
    ###Run dest
    printHeader("\n\nStart building cluster-destination")
    os.system("make run-kind-dest")
    waitPod("cluster-mbg")
    podest= getPodName("cluster-mbg")
    destIp=getKindIp("destCluster")
    runcmd(f"kubectl create service nodeport iperf3-server --tcp=5000:5000 --node-port={iperf3DestPort}")
    runcmdb(f'kubectl exec -i {podest} -- ./cluster start --id "destCluster"  --ip {destIp} --cport 30000 --mbgIP {mbg2Ip}:30000')
    printHeader(f"Add {destSvc} (server) service to destination cluster")
    runcmd(f'kubectl exec -i {podest} -- ./cluster addService --serviceId {destSvc} --serviceIp {destIp}:{iperf3DestPort}')
    
    
    #Add host cluster to MBG1
    runcmd(f'kubectl config use-context kind-mbg-agent1')
    printHeader("Add host cluster to MBG1")
    runcmd(f'kubectl exec -i {podMbg1} -- ./mbg addCluster --id "hostCluster" --ip {hostIp}:30000')

    #Add dest cluster to MBG2
    runcmd(f'kubectl config use-context kind-mbg-agent2')
    printHeader("Add dest cluster to MBG2")
    runcmd(f'kubectl exec -i {podMbg2} -- ./mbg addCluster --id "destCluster" --ip {destIp}:30000')

    #Expose service
    runcmd(f'kubectl config use-context kind-cluster-dest')
    printHeader("\n\nStart exposing connection")
    runcmd(f'kubectl exec -i {podest} -- ./cluster expose --serviceId {destSvc}')

    #Connect service
    printHeader(f"\n\nStart Data plan connection {srcSvc} to {destSvc}")
    runcmd(f'kubectl config use-context kind-cluster-host')
    runcmdb(f'kubectl exec -i {podhost} -- ./cluster connect --serviceId {srcSvc}  --serviceIdDest {destSvc}')
    time.sleep(20)
    
    
    #Create Nodeports inside mbg
    printHeader(f"\n\nCreate nodeports for data-plane connection")
    runcmd(f'kubectl config use-context kind-mbg-agent2')
    runcmd("kubectl create service nodeport mbg --tcp=5081:5081 --node-port=30082")
    runcmd(f'kubectl config use-context kind-mbg-agent1')
    runcmd("kubectl create service nodeport mbg --tcp=5000:5000 --node-port=30001")
    
    #runcmd(f'kubectl exec -i {podhost} --  cat /root/.clusterApp')

    #Testing
    printHeader("\n\nStart Iperf3 testing")
    runcmd(f'kubectl config use-context kind-cluster-host')
    podIperf3= getPodName("iperf3-clients")
    
    printHeader("The Iperf3 test connects directly to the destination")
    cmd = f'kubectl exec -i {podIperf3} --  iperf3 -c {destIp} -p {iperf3DestPort}'
    iperf3Test(cmd)

    printHeader("The Iperf3 test connects to MBG1")
    cmd = f'kubectl exec -i {podIperf3} --  iperf3 -c {mbg1Ip} -p {mbg1DataPort}'
    iperf3Test(cmd)
    
    printHeader("fULL Iperf3 test clinet-> MBG1-> MBG2-> dest")
    cmd = f'kubectl exec -i {podIperf3} --  iperf3 -c cluster-iperf3-service -p 5000'
    iperf3Test(cmd)

    #Close connection
    printHeader("\n\nClose Iperf3 connection")
    runcmd(f'kubectl exec -i {podhost} -- ./cluster disconnect --serviceId {srcSvc} --serviceIdDest {destSvc}')
