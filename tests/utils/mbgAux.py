import os,time,json
import subprocess as sp
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

def getPodNameIp(app):
    podName = getPodNameApp(app)
    podIp   =  getPodIp(podName)  
    return podName, podIp

def getPodNameApp(app):
    cmd=f"kubectl get pods -l app={app} "+'-o jsonpath="{.items[0].metadata.name}"'
    podName=sp.getoutput(cmd)
    return podName

def getPodName(prefix):
    podName=sp.getoutput(f'kubectl get pods -o name | fgrep {prefix}| cut -d\'/\' -f2')
    return podName

def getPodIp(name):
    name=getPodName(name)
    podIp=sp.getoutput(f"kubectl get pod {name}"+" --template '{{.status.podIP}}'")
    return podIp

def runcmd(cmd):
    print(f'{Fore.YELLOW}{cmd} {Style.RESET_ALL}')
    #sp.Popen(cmd,shell=True)
    os.system(cmd)
    
def runcmdb(cmd):
    print(f'{Fore.YELLOW}{cmd} {Style.RESET_ALL}')
    #sp.Popen(cmd,shell=True)
    os.system(cmd + ' &')
    time.sleep(7)

def printHeader(msg):
    print(f'{Fore.GREEN}{msg} {Style.RESET_ALL}')
    #print(msg)

def getMbgPorts(podMbg, destSvc):
    mbgJson =sp.getoutput(f' kubectl exec -i {podMbg} -- cat ./root/.mbg/mbgApp')
    mbgJson=json.loads(mbgJson)
    localPort =(mbgJson["Connections"][destSvc]["Local"]).split(":")[1]
    externalPort =(mbgJson["Connections"][destSvc]["External"]).split(":")[1]
    print(f"Service nodeport will use local Port: {localPort} and externalPort:{externalPort}")
    return localPort, externalPort

def buildMbg(name):
    runcmd(f"kubectl apply -f {folMfst}/mbg/mbg-role.yaml")
    runcmd(f"kubectl create -f {folMfst}/mbg/mbg.yaml")
    waitPod("mbg")
    podMbg, mbgIp= getPodNameIp("mbg")
    return podMbg, mbgIp

def buildMbgctl(name):
    runcmd(f"kubectl create -f {folMfst}/mbgctl/mbgctl.yaml")
    waitPod("mbgctl")
    name,ip= getPodNameIp("mbgctl")
    return name, ip 

def clean_cluster():
    runcmd(f'kubectl delete --all deployments')
    runcmd(f'kubectl delete --all svc')