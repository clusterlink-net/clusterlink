import json,os
import subprocess as sp
from demos.utils.manifests.kind.flannel.create_cni_bridge import createCniBridge,createKindCfgForflunnel
from demos.utils.mbgAux import runcmd, runcmdb, printHeader, waitPod, getPodNameIp, getMbgPorts,buildMbg,buildMbgctl,getPodIp,proj_dir

def BuildKindCluster(name, cni="default", cfg="" ):
    #Set config file
    cfgFlag = f" --config {cfg}" if cfg != "" else  ""
    cfgFlag = f" --config {proj_dir}/demos/utils/manifests/kind/calico/calico-config.yaml" if (cfg == "" and cni== "calico")  else cfgFlag
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

def startGwctl(gwctlName, mbgIP, mbgcPort, dataplane, gwctlcrt):
    runcmd(f'gwctl init --id {gwctlName} --gwIP {mbgIP} --gwPort {mbgcPort}  --dataplane {dataplane} {gwctlcrt} ')

def startKindClusterMbg(mbgName, gwctlName, mbgcPortLocal, mbgcPort, mbgDataPort,dataplane, mbgcrtFlags, gwctlLocal=True, runInfg=False, cni="default", logFile=True,zeroTrust=False):
    os.system(f"kind delete cluster --name={mbgName}")
    ###first Mbg
    printHeader(f"\n\nStart building {mbgName}")
    mbgKindIp           = BuildKindCluster(mbgName,cni)
    podMbg, podMbgIp    = buildMbg(mbgName)
    podDataPlane, _= getPodNameIp("dataplane")

    runcmd(f"kubectl create service nodeport dataplane --tcp={mbgcPortLocal}:{mbgcPortLocal} --node-port={mbgcPort}")
    
    printHeader(f"\n\nStart {mbgName} (along with PolicyEngine)")
    startcmd= f'{podMbg} -- ./controlplane start --id "{mbgName}" --ip {mbgKindIp} --cport {mbgcPort} --cportLocal {mbgcPortLocal}  --externalDataPortRange {mbgDataPort}\
    --dataplane {dataplane} {mbgcrtFlags} --startPolicyEngine={True} --observe={True} --logFile={logFile} --zeroTrust={zeroTrust}'

    if runInfg:
        runcmd("kubectl exec -it " + startcmd)
        runcmd(f"kubectl exec -it {podDataPlane} -- ./dataplane --id {mbgName} --dataplane {dataplane} {mbgcrtFlags}")
    else:
        runcmdb("kubectl exec -i " + startcmd)
        runcmdb(f"kubectl exec -i {podDataPlane} -- ./dataplane --id {mbgName} --dataplane {dataplane} {mbgcrtFlags}")

    if gwctlLocal:
        gwctlPod, gwctlIp = buildMbgctl(gwctlName)
        runcmdb(f'kubectl exec -i {gwctlPod} -- ./gwctl init --id {gwctlName} --gwIP {podMbgIp} --gwPort {mbgcPortLocal} --dataplane {dataplane} {mbgcrtFlags} ')

