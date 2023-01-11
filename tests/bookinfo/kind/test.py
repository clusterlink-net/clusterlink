import os,time
import subprocess as sp
import sys
import argparse
proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))

sys.path.insert(0,f'{proj_dir}/tests/')
print(f"{proj_dir}/tests/")
from aux.kindAux import runcmd, runcmdb, printHeader, waitPod, getPodName, getKindIp, getMbgPorts,buildMbg,buildMbgctl



def connectSvc(srcSvc,destSvc,policy):
    printHeader(f"\n\nStart Data plan connection {srcSvc} to {destSvc}")
    runcmd(f'kubectl config use-context kind-product-cluster')
    podhost= getPodName("mbgctl")
    runcmdb(f'kubectl exec -i {podhost} -- ./mbgctl connect --serviceId {srcSvc}  --serviceIp {srcSvcIp} --serviceIdDest {destSvc}')

    runcmd(f'kubectl config use-context kind-mbg-agent1')
    podMbg1= getPodName("mbg")
    mbg2LocalPort, mbg1ExternalPort = getMbgPorts(podMbg1, srcSvc, destSvc)
    svcName=f"svc{destSvc}"
    runcmd(f"kubectl create service nodeport {svcName} --tcp={mbg2LocalPort}:{mbg2LocalPort} --node-port={mbg1ExternalPort}")
    runcmd(f"kubectl patch service {svcName} -p "+  "\'{\"spec\":{\"selector\":{\"app\": \"mbg\"}}}\'") #replacing app name

############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-c','--command', help='Script command: test/connect/disconnect', required=False, default="test")
    parser.add_argument('-d','--dataplane', help='choose which dataplane to use mtls/tcp', required=False, default="tcp")
    parser.add_argument('-src','--src', help='Source service name', required=False)
    parser.add_argument('-dst','--dest', help='Destination service name', required=False)
    args = vars(parser.parse_args())

    printHeader("\n\nStart Kind Test\n\n")
    printHeader("Start pre-setting")

    review2DestPort="30001"
    review2svc="review-v2"
    review3DestPort="30002"
    review3svc="review-v3"

    mbg1DataPort= "30001"
    mbg1MtlsPort= "30443"
    mbg1MtlsPortLocal= "8443"

    mbg2DataPort= "30001"
    mbg2MtlsPort= "30443"
    mbg2MtlsPortLocal= "8443"

    srcSvc="review"
    srcSvcIp=":9080"
    srcDefaultGW="10.244.0.1"
    svcpolicy ="Forward"

    dataplane = args["dataplane"]
    mbg1crtFlags =f"--rootCa ./mtls/ca.crt  --certificate ./mtls/mbg1.crt --key ./mtls/mbg1.key"  if dataplane =="mtls" else ""
    mbg2crtFlags =f"--rootCa ./mtls/ca.crt  --certificate ./mtls/mbg2.crt --key ./mtls/mbg2.key"  if dataplane =="mtls" else ""
    

    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)
    if args["command"] == "disconnect":
        runcmd(f'kubectl config use-context kind-product-cluster')
        podhost= getPodName("mbgctl")
        printHeader("\n\nClose Iperf3 connection")
        runcmd(f'kubectl exec -i {podhost} -- ./mbgctl disconnect --serviceId {args["src"]} --serviceIdDest {args["dest"]}')
    elif args["command"] == "connect":
        connectSvc(args["src"],args["dest"],svcpolicy)
    else:
        ### clean 
        print(f"Clean old kinds")
        os.system("make clean-kind-bookinfo")
        
        ### build docker environment 
        printHeader(f"Build docker image")
        os.system("make docker-build")
        ###Run first Mbg
        printHeader("\n\nStart building MBG1")
        podMbg1, mbg1Ip= buildMbg("mbg-agent1",f"{proj_dir}/manifests/kind/mbg-config1.yaml")
        runcmdb(f'kubectl exec -i {podMbg1} -- ./mbg start --id "MBG1" --ip {mbg1Ip} --cport "30000" --externalDataPortRange {mbg1DataPort} \
            --dataplane {args["dataplane"]} --mtlsport {mbg1MtlsPort} --mtlsportLocal {mbg1MtlsPortLocal} {mbg1crtFlags}')
        if dataplane =="mtls" :
            runcmd(f"kubectl create service nodeport mbg --tcp={mbg1MtlsPortLocal}:{mbg1MtlsPortLocal} --node-port={mbg1MtlsPort}")

        ###Run Second Mbg
        printHeader("\n\nStart building MBG2")
        podMbg2, mbg2Ip= buildMbg("mbg-agent2",f"{proj_dir}/manifests/kind/mbg-config2.yaml")
        runcmdb(f'kubectl exec -i {podMbg2} --  ./mbg start --id "MBG2" --ip {mbg2Ip} --cport "30000" --externalDataPortRange {mbg2DataPort}\
        --dataplane {args["dataplane"]}  --mtlsport {mbg2MtlsPort} --mtlsportLocal {mbg2MtlsPortLocal}  {mbg2crtFlags}')
        if dataplane =="mtls":
            runcmd(f"kubectl create service nodeport mbg --tcp={mbg2MtlsPortLocal}:{mbg2MtlsPortLocal} --node-port={mbg2MtlsPort}")
    
        ###Run host
        printHeader("\n\nStart building product-cluster")
        folpdct=f"{proj_dir}/tests/bookinfo/manifests/product/"
        runcmd(f"kind create cluster --config {folpdct}/kind-config.yaml --name=product-cluster")
        runcmd(f"kind load docker-image mbg --name=product-cluster")
        runcmd(f"kind load docker-image maistra/examples-bookinfo-productpage-v1 --name=product-cluster")
        runcmd(f"kind load docker-image maistra/examples-bookinfo-details-v1:0.12.0 --name=product-cluster")
        
        runcmd(f"kubectl create -f {folpdct}/product.yaml")
        runcmd(f"kubectl create -f {folpdct}/details.yaml")
        runcmd(f"kubectl create -f {folpdct}/review-svc.yaml")
        podhost, hostIp= buildMbgctl("product Cluster")
        runcmdb(f'kubectl exec -i {podhost} -- ./mbgctl start --id "productCluster"  --ip {hostIp} --cport 30000 --mbgIP {mbg1Ip}:30000')
        printHeader(f"Add {srcSvc} (client) service to host cluster")
        runcmd(f'kubectl exec -i {podhost} -- ./mbgctl addService --serviceId {srcSvc} --serviceIp {srcDefaultGW}')
        
        # Add MBG Peer
        printHeader("Add MBG2 peer to MBG1")
        runcmd(f'kubectl exec -i {podhost} -- ./mbgctl addPeer --id "MBG2" --ip {mbg2Ip} --cport "30000"')
    
        # Send Hello
        printHeader("Send Hello commands")
        runcmd(f'kubectl exec -i {podhost} -- ./mbgctl hello')
        
        ###Run dest
        printHeader("\n\nStart building review-cluster")
        folReview=f"{proj_dir}/tests/bookinfo/manifests/review"
        runcmd(f"kind create cluster --config {folReview}/kind-config.yaml --name=review-cluster")
        runcmd(f"kind load docker-image mbg --name=review-cluster")
        runcmd(f"kind load docker-image maistra/examples-bookinfo-reviews-v2 --name=review-cluster")
        runcmd(f"kind load docker-image maistra/examples-bookinfo-reviews-v3 --name=review-cluster")
        runcmd(f"kind load docker-image maistra/examples-bookinfo-ratings-v1:0.12.0 --name=review-cluster")
        runcmd(f"kubectl create -f {folReview}/review-v3.yaml")
        runcmd(f"kubectl create service nodeport reviews-v3 --tcp=9080:9080 --node-port={review3DestPort}")
        runcmd(f"kubectl create -f {folReview}/review-v2.yaml")
        runcmd(f"kubectl create service nodeport reviews-v2 --tcp=9080:9080 --node-port={review2DestPort}")
        runcmd(f"kubectl create -f {folReview}/rating.yaml")
        podest, destIp= buildMbgctl("dest Cluster")   
        runcmdb(f'kubectl exec -i {podest} -- ./mbgctl start --id "reviewCluster"  --ip {destIp} --cport 30000 --mbgIP {mbg2Ip}:30000')
        printHeader(f"Add {review2svc} (server) service to destination cluster")
        runcmd(f'kubectl exec -i {podest} -- ./mbgctl addService --serviceId {review2svc} --serviceIp {destIp}:{review2DestPort}')
        runcmd(f'kubectl exec -i {podest} -- ./mbgctl addService --serviceId {review3svc} --serviceIp {destIp}:{review3DestPort}')

        
        
        #Add host cluster to MBG1
        runcmd(f'kubectl config use-context kind-mbg-agent1')
        printHeader("Add host cluster to MBG1")
        runcmd(f'kubectl exec -i {podMbg1} -- ./mbg addMbgctl --id "productCluster" --ip {hostIp}:30000')

        #Add dest cluster to MBG2
        runcmd(f'kubectl config use-context kind-mbg-agent2')
        printHeader("Add dest cluster to MBG2")
        runcmd(f'kubectl exec -i {podMbg2} -- ./mbg addMbgctl --id "reviewCluster" --ip {destIp}:30000')
        
        #Expose service
        runcmd(f'kubectl config use-context kind-review-cluster')
        printHeader("\n\nStart exposing connection")
        runcmd(f'kubectl exec -i {podest} -- ./mbgctl expose --serviceId {review2svc}')
        runcmd(f'kubectl exec -i {podest} -- ./mbgctl expose --serviceId {review3svc}')

        #Get services
        runcmd(f'kubectl config use-context kind-product-cluster')
        printHeader("\n\nStart get service")
        runcmdb(f'kubectl exec -i {podhost} -- ./mbgctl getService')
    
        #connect
        connectSvc(srcSvc,review2svc,svcpolicy)
