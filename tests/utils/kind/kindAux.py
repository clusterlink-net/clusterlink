import json,os
import subprocess as sp

from tests.utils.mbgAux import runcmd, runcmdb, printHeader, waitPod, getPodName, getMbgPorts,buildMbg,buildMbgctl,getPodIp

def BuildKindCluster(name, cfg=""):
    if cfg != "": 
        runcmd(f"kind create cluster --config {cfg} --name={name}")
    else:
        runcmd(f"kind create cluster --name={name}")
    runcmd(f"kind load docker-image mbg --name={name}")
    mbgIp=getKindIp(name)
    return mbgIp

def useKindCluster(name):
    os.system(f'kubectl config use-context kind-{name}')

def getKindIp(name):
    clJson=json.loads(sp.getoutput(f' kubectl get nodes -o json'))
    ip = clJson["items"][0]["status"]["addresses"][0]["address"]
    return ip


def startKindClusterMbg(mbgName, mbgctlName, mbgcPortLocal, mbgcPort, mbgDataPort,dataplane ,mbgcrtFlags, runInfg=False):
    os.system(f"kind delete cluster --name={mbgName}")
    ###first Mbg
    printHeader(f"\n\nStart building {mbgName}")
    mbgKindIp           = BuildKindCluster(mbgName)
    podMbg, podMbgIp    = buildMbg(mbgName)
    mbgctlPod, mbgctlIp = buildMbgctl(mbgctlName)
    destMbgIp          = f"{podMbgIp}:{mbgcPortLocal}"
    runcmd(f"kubectl create service nodeport mbg --tcp={mbgcPortLocal}:{mbgcPortLocal} --node-port={mbgcPort}")
    runcmdb(f'kubectl exec -i {mbgctlPod} -- ./mbgctl start --id {mbgctlName}  --ip {mbgctlIp} --mbgIP {destMbgIp}  --dataplane {dataplane} {mbgcrtFlags} ')
    runcmd(f'kubectl exec -i {mbgctlPod} -- ./mbgctl addPolicyEngine --target {podMbgIp}:9990')
    
    printHeader(f"\n\nStart {mbgName} (along with PolicyEngine)")
    startcmd= f'{podMbg} -- ./mbg start --id "{mbgName}" --ip {mbgKindIp} --cport {mbgcPort} --cportLocal {mbgcPortLocal}  --externalDataPortRange {mbgDataPort}\
    --dataplane {dataplane} {mbgcrtFlags} --startPolicyEngine {True}'
    
    if runInfg:
        runcmd("kubectl exec -it " + startcmd)
    else:
        runcmdb("kubectl exec -i " + startcmd)
