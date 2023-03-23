import json,os
import subprocess as sp
from tests.utils.manifests.kind.flannel.create_cni_bridge import createCniBridge,createKindCfgForflunnel
from tests.utils.mbgAux import runcmd, runcmdb, printHeader, waitPod, getPodName, getMbgPorts,buildMbg,buildMbgctl,getPodIp,proj_dir

def BuildKindCluster(name, cni="default", cfg="" ):
    #Set config file
    cfgFlag = f" --config {cfg}" if cfg != "" else  ""
    cfgFlag = f" --config {proj_dir}/tests/utils/manifests/kind/calico/calico-config.yaml" if (cfg == "" and cni== "calico")  else cfgFlag
    if  cni == "flannel" and cfg =="":
        cfgFlag = f" --config {proj_dir}/bin/plugins/flannel-config.yaml"
        createCniBridge()
        createKindCfgForflunnel()

    runcmd(f"kind create cluster  --name={name} {cfgFlag}")
    if  cni == "flannel":
        runcmd("kubectl apply -f https://raw.githubusercontent.com/flannel-io/flannel/master/Documentation/kube-flannel.yml")
    if  cni == "calico":
        runcmd("kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.25.0/manifests/tigera-operator.yaml")
        runcmd("kubectl create -f https://raw.githubusercontent.com/projectcalico/calico/v3.25.0/manifests/custom-resources.yaml")

    runcmd(f"kind load docker-image mbg --name={name}")
    mbgIp=getKindIp(name)
    return mbgIp

def useKindCluster(name):
    os.system(f'kubectl config use-context kind-{name}')

def getKindIp(name):
    clJson=json.loads(sp.getoutput(f' kubectl get nodes -o json'))
    ip = clJson["items"][0]["status"]["addresses"][0]["address"]
    return ip


def startKindClusterMbgOnly(mbgName, mbgctlName, mbgcPortLocal, mbgcPort, mbgDataPort,dataplane ,mbgcrtFlags, runInfg=False,cni="default"):
    os.system(f"kind delete cluster --name={mbgName}")
    ###first Mbg
    printHeader(f"\n\nStart building {mbgName}")
    mbgKindIp           = BuildKindCluster(mbgName,cni)
    podMbg, podMbgIp    = buildMbg(mbgName)
    runcmd(f"kubectl create service nodeport mbg --tcp={mbgcPortLocal}:{mbgcPortLocal} --node-port={mbgcPort}")
    runcmd(f"kubectl create service nodeport policy --tcp={9990}:{9990} --node-port={30444}")

    printHeader(f"\n\nStart {mbgName} (along with PolicyEngine)")
    startcmd= f'{podMbg} -- ./mbg start --id "{mbgName}" --ip {mbgKindIp} --cport {mbgcPort} --cportLocal {mbgcPortLocal}  --externalDataPortRange {mbgDataPort}\
    --dataplane {dataplane} {mbgcrtFlags} --startPolicyEngine {True} --policyEngineIp {podMbgIp}:{mbgcPortLocal}'
    
    if runInfg:
        runcmd("kubectl exec -it " + startcmd)
    else:
        runcmdb("kubectl exec -i " + startcmd)

def startMbgctl(mbgctlName, mbgIP, mbgcPort, dataplane, mbgctlcrt):
    runcmd(f'mbgctl create --id {mbgctlName} --mbgIP {mbgIP}:{mbgcPort}  --dataplane {dataplane} {mbgctlcrt} ')
    runcmd(f'mbgctl add policyengine --myid {mbgctlName} --target {mbgIP}:{mbgcPort}')

def startKindClusterMbg(mbgName, mbgctlName, mbgcPortLocal, mbgcPort, mbgDataPort,dataplane ,mbgcrtFlags, runInfg=False,cni="default"):
    os.system(f"kind delete cluster --name={mbgName}")
    ###first Mbg
    printHeader(f"\n\nStart building {mbgName}")
    mbgKindIp           = BuildKindCluster(mbgName,cni)
    podMbg, podMbgIp    = buildMbg(mbgName)
    mbgctlPod, mbgctlIp = buildMbgctl(mbgctlName)
    destMbgIp          = f"{podMbgIp}:{mbgcPortLocal}"
    runcmd(f"kubectl create service nodeport mbg --tcp={mbgcPortLocal}:{mbgcPortLocal} --node-port={mbgcPort}")
    runcmdb(f'kubectl exec -i {mbgctlPod} -- ./mbgctl create --id {mbgctlName} --mbgIP {destMbgIp}  --dataplane {dataplane} {mbgcrtFlags} ')
    
    printHeader(f"\n\nStart {mbgName} (along with PolicyEngine)")
    startcmd= f'{podMbg} -- ./mbg start --id "{mbgName}" --ip {mbgKindIp} --cport {mbgcPort} --cportLocal {mbgcPortLocal}  --externalDataPortRange {mbgDataPort}\
    --dataplane {dataplane} {mbgcrtFlags} --startPolicyEngine {True} --logFile {logFile}'
    
    
    if runInfg:
        runcmd("kubectl exec -it " + startcmd)
    else:
        runcmdb("kubectl exec -i " + startcmd)
