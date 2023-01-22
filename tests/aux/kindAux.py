import os,time,json
import subprocess as sp
import netifaces as ni
from colorama import Fore
from colorama import Style

proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
folMfst=f"{proj_dir}/manifests"

def waitPod(name):
    time.sleep(2) #Initial start
    podStatus=""
    while(podStatus != "Running"):
        #cmd=f"kubectl get pods -l app={name} -o jsonpath" + "=\'{.items[0].status.containerStatuses[0].ready}\'"
        cmd=f"kubectl get pods -l app={name} "+ '--no-headers -o custom-columns=":status.phase"'
        podStatus =sp.getoutput(cmd)
        if (podStatus != "Running"):
            print (f"Waiting for pod {name} to start current status: {podStatus}")
            time.sleep(7)
        else:
            time.sleep(5)
            break

def getPodName(prefix):
    podName=sp.getoutput(f'kubectl get pods -o name | fgrep {prefix}| cut -d\'/\' -f2')
    return podName

def getPodIp(name):
    name=getPodName(name)
    podIp=sp.getoutput(f"kubectl get pod {name}"+" --template '{{.status.podIP}}'")
    return podIp

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
    clJson=json.loads(sp.getoutput(f' kubectl get nodes -o json'))
    ip = clJson["items"][0]["status"]["addresses"][0]["address"]
    print(f"Kind Cluster {name} ip address:{ip}")
    return ip

def printHeader(msg):
    print(f'{Fore.BLUE}{msg} {Style.RESET_ALL}')
    #print(msg)

def getMbgPorts(podMbg, destSvc):
    mbgJson=json.loads(sp.getoutput(f' kubectl exec -i {podMbg} -- cat ./root/.mbgApp'))
    localPort =(mbgJson["Connections"][destSvc]["Local"]).split(":")[1]
    externalPort =(mbgJson["Connections"][destSvc]["External"]).split(":")[1]
    print(f"Service nodeport will use local Port: {localPort} and externalPort:{externalPort}")
    return localPort, externalPort

def buildMbg(name,cfg):
    runcmd(f"kind create cluster --config {cfg} --name={name}")
    runcmd(f"kind load docker-image mbg --name={name}")
    runcmd(f"kind load docker-image tcp-split --name={name}")
    runcmd(f"kubectl create -f {folMfst}/mbg/mbg.yaml")
    runcmd(f"kubectl create -f {folMfst}/mbg/mbg-client-svc.yaml")
    runcmd(f"kubectl create -f {folMfst}/tcp-split/tcp-split.yaml")
    runcmd(f"kubectl create -f {folMfst}/tcp-split/tcp-split-svc.yaml")
    waitPod("mbg")
    podMbg= getPodName("mbg")
    mbgIp=getKindIp(name)
    return podMbg, mbgIp

def buildMbgctl(name, mbgMode):
    runcmd(f"kubectl create -f {folMfst}/mbgctl/mbgctl.yaml")
    runcmd(f"kubectl create -f {folMfst}/mbgctl/mbgctl-svc.yaml")
    podName= getPodName("mbgctl")
    waitPod("mbgctl")
    ip= getPodIp("mbgctl") if mbgMode=="inside" else getKindIp(name)
    return podName, ip 

def useKindCluster(name):
    runcmd(f'kubectl config use-context kind-{name}')