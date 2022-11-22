import os,sys,time
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

def getIp(Interface):
    ip = ni.ifaddresses(Interface)[ni.AF_INET][0]['addr']
    print(f"local Ip addrees:{ip}")
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
    ipAddr=getIp("eth0")
    iperf3DestPort="20201"
    mbg1DataPort= "30101"
    mbg2DataPort= "30201"

    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)
    ### clean 
    print(f"Clean old kinds")
    os.system("make clean-kind-mbg")

    ###Run first Mbg
    printHeader("\n\nStart building MBG1")
    os.system("make run-kind-mbg1")
    waitPod("mbg")
    podMbg1= getPodName("mbg")
    runcmdb(f'kubectl exec -i {podMbg1} -- ./mbg start --id "mbg1" --ip {ipAddr} --cport "30100" --externalDataPortRange {mbg1DataPort}')
    printHeader("Add host cluster to MBG1")
    runcmd(f'kubectl exec -i {podMbg1} -- ./mbg addCluster --id "hostCluster" --ip {ipAddr}:20100')

    ###Run Second Mbg
    printHeader("\n\nStart building MBG2")
    os.system("make run-kind-mbg2")
    waitPod("mbg")
    podMbg2 = getPodName("mbg")
    runcmdb(f'kubectl exec -i {podMbg2} --  ./mbg start --id "mbg2" --ip {ipAddr} --cport "30200" --externalDataPortRange {mbg2DataPort}')
    printHeader("Add MBG1 neighbor to MBG2")
    runcmd(f'kubectl exec -i {podMbg2} -- ./mbg addMbg --id "mbg1" --ip {ipAddr} --cport "30100"')
    printHeader("Send Hello commands")
    runcmd(f'kubectl exec -i {podMbg2} -- ./mbg hello')
    printHeader("Add Destination Cluster to MBG2")
    runcmd(f'kubectl exec -i {podMbg2} -- ./mbg addCluster --id "destCluster" --ip {ipAddr}:20200')

    ###Run host
    printHeader("\n\nStart building cluster-host")
    os.system("make run-kind-host")
    waitPod("cluster-mbg")
    podhost= getPodName("cluster-mbg")
    runcmdb(f'kubectl exec -i {podhost} -- ./cluster start --id "hostCluster"  --ip {ipAddr} --cport 20100 --mbgIP {ipAddr}:30100')

    printHeader("Add iperfIsrael (client) service to host cluster")
    runcmd(f'kubectl exec -i {podhost} -- ./cluster addService --serviceId iperfIsrael --serviceIp :5000')
    
    ###Run dest
    printHeader("\n\nStart building cluster-destination")
    os.system("make run-kind-dest")
    waitPod("cluster-mbg")
    podest= getPodName("cluster-mbg")
    runcmdb(f'kubectl exec -i {podest} -- ./cluster start --id "destCluster"  --ip {ipAddr} --cport 20200 --mbgIP {ipAddr}:30200')
    printHeader("Add iperfIndia (server) service to destination cluster")
    runcmd(f'kubectl exec -i {podest} -- ./cluster addService --serviceId iperfIndia --serviceIp {ipAddr}:{iperf3DestPort}')

    #Expose service
    printHeader("\n\nStart exposing connection")
    runcmd(f'kubectl exec -i {podest} -- ./cluster expose --serviceId iperfIndia')

    #Connect service
    printHeader("\n\nStart Data plan connection iperfIsrael to iperfIndia")
    runcmd(f'kubectl config use-context kind-cluster-host')
    #runcmd(f'kubectl exec -i {podhost} --  cat /root/.clusterApp')
    runcmdb(f'kubectl exec -i {podhost} -- ./cluster connect --serviceId iperfIsrael  --serviceIdDest iperfIndia')
    time.sleep(30)
    
    #Testing
    printHeader("\n\nStart Iperf3 testing")
    podIperf3= getPodName("iperf3-clients")
    
    printHeader("The Iperf3 test connects directly to the destination")
    cmd = f'kubectl exec -i {podIperf3} --  iperf3 -c {ipAddr} -p {iperf3DestPort}'
    iperf3Test(cmd)

    printHeader("The Iperf3 test connects to MBG1")
    cmd = f'kubectl exec -i {podIperf3} --  iperf3 -c {ipAddr} -p {mbg1DataPort}'
    iperf3Test(cmd)
    
    printHeader("fULL Iperf3 test clinet-> MBG1-> MBG2-> dest")
    cmd = f'kubectl exec -i {podIperf3} --  iperf3 -c cluster-iperf3-service -p 5000'
    iperf3Test(cmd)
    
