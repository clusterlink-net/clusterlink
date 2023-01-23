############################################################
# Name: Bookinfo 2 clusters test
# Info: support bookinfo application with mbgctl inside or outside the application 
#       In this test microservices: product and details locates in MBG1 
#       and review and rating in MBG2
#########################################################
import os,time
import subprocess as sp
import sys
import argparse
proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))

sys.path.insert(0,f'{proj_dir}/tests/')
print(f"{proj_dir}/tests/")
from aux.kindAux import runcmd, runcmdb, printHeader, getPodName, waitPod,getMbgPorts,buildMbg,buildMbgctl,useKindCluster,getPodIp



def connectSvc(srcSvc,destSvc,srcK8sName,policy):
    printHeader(f"\n\nStart connect {srcSvc} to {destSvc}")
    useKindCluster(mbg1ClusterName)    
    podMbg1= getPodName("mbg-deployment")        
    mbg1LocalPort, mbg1ExternalPort = getMbgPorts(podMbg1, destSvc)
    if mbgMode !="inside": #Set forwarder
        printHeader(f"\n\nStart Data plan connection {srcSvc} to {destSvc}")
        useKindCluster(productClusterName)
        podhost= getPodName("mbgctl")
        runcmdb(f'kubectl exec -i {podhost} -- ./mbgctl connect --serviceId {srcSvc}  --serviceIp {srcK8sSvcIp} --serviceIdDest {destSvc}')

        useKindCluster(mbg1ClusterName)    
        svcName=f"svc{destSvc}"
        runcmd(f"kubectl create service nodeport {svcName} --tcp={mbg1LocalPort}:{mbg1LocalPort} --node-port={mbg1ExternalPort}")
        runcmd(f"kubectl patch service {svcName} -p "+  "\'{\"spec\":{\"selector\":{\"app\": \"mbg\"}}}\'") #replacing app name
    else:
        runcmd(f"kubectl delete service {srcK8sName}")
        runcmd(f"kubectl create service clusterip {srcK8sName} --tcp={srcK8sSvcPort}:{mbg1LocalPort}")
        runcmd(f"kubectl patch service {srcK8sName} -p "+  "\'{\"spec\":{\"selector\":{\"app\": \"mbg\"}}}\'") #replacing app name

############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-c','--command', help='Script command: test/connect/disconnect', required=False, default="test")
    parser.add_argument('-d','--dataplane', help='choose which dataplane to use mtls/tcp', required=False, default="tcp")
    parser.add_argument('-m','--mbgmode', help='mbg mode inside or outside the cluste', required=False, default="inside")

    parser.add_argument('-src','--src', help='Source service name', required=False)
    parser.add_argument('-dst','--dest', help='Destination service name', required=False)
    args = vars(parser.parse_args())

    printHeader("\n\nStart Kind Test\n\n")
    printHeader("Start pre-setting")
    
    folpdct   = f"{proj_dir}/tests/bookinfo/manifests/product/"
    folReview = f"{proj_dir}/tests/bookinfo/manifests/review"
    dataplane = args["dataplane"]
    mbgMode   = args["mbgmode"]
    #MBG1 parameters 
    mbg1DataPort    = "30001"
    mbg1cPort       = "30443"
    mbg1cPortLocal  = "8443"
    mbg1ClusterName = "mbg-agent1"
    mbg1crtFlags    = f"--rootCa ./mtls/ca.crt --certificate ./mtls/mbg1.crt --key ./mtls/mbg1.key"  if dataplane =="mtls" else ""
    
    #MBG2 parameters 
    mbg2DataPort    = "30001"
    mbg2cPort       = "30443"
    mbg2cPortLocal  = "8443"
    mbg2crtFlags    = f"--rootCa ./mtls/ca.crt --certificate ./mtls/mbg2.crt --key ./mtls/mbg2.key"  if dataplane =="mtls" else ""
    mbg2ClusterName = "mbg-agent2"
    #Product cluster
    srcSvc             = "productpage"
    srcK8sName         = "reviews"
    srcK8sSvcPort      = "9080"
    srcK8sSvcIp        = ":"+srcK8sSvcPort
    srcDefaultGW       = "10.244.0.1"
    svcpolicy          = "Forward"
    productCrtFlags    = f"--rootCa ./mtls/ca.crt --certificate ./mtls/mbg1.crt --key ./mtls/mbg1.key"  if dataplane =="mtls" else ""
    productClusterName = "mbg-agent1" if mbgMode =="inside" else "product-cluster"

    #Review cluster
    review2DestPort   = "30001"
    review2svc        = "reviews-v2"
    review3DestPort   = "30002"
    review3svc        = "reviews-v3"
    reviewClusterName = "mbg-agent2" if mbgMode =="inside" else"review-cluster"
    reviewCrtFlags    = f"--rootCa ./mtls/ca.crt --certificate ./mtls/mbg2.crt --key ./mtls/mbg2.key"  if dataplane =="mtls" else ""
    
    

    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)
    if args["command"] == "disconnect":
        useKindCluster(productClusterName)
        podhost= getPodName("mbgctl")
        printHeader("\n\nClose Iperf3 connection")
        runcmd(f'kubectl exec -i {podhost} -- ./mbgctl disconnect --serviceId {args["src"]} --serviceIdDest {args["dest"]}')
    elif args["command"] == "connect":
        connectSvc(args["src"],args["dest"],srcK8sName,svcpolicy)
    else:
        ### clean 
        print(f"Clean old kinds")
        os.system("make clean-kind-bookinfo")
        
        ### build docker environment 
        printHeader(f"Build docker image")
        os.system("make docker-build")
        ## build Kind clusters environment 
        ###first Mbg
        printHeader("\n\nStart building MBG1")
        podMbg1, mbg1Ip= buildMbg("mbg-agent1",f"{proj_dir}/manifests/kind/mbg-config1.yaml")
        ###Second Mbg
        printHeader("\n\nStart building MBG2")
        podMbg2, mbg2Ip= buildMbg("mbg-agent2",f"{proj_dir}/manifests/kind/mbg-config2.yaml")
        if mbgMode !="inside":
            ###Run host
            printHeader("\n\nStart building product-cluster")
            runcmd(f"kind create cluster --config {folpdct}/kind-config.yaml --name={productClusterName}")
            runcmd(f"kind load docker-image mbg --name={productClusterName}")
            ###Run dest
            printHeader("\n\nStart building {reviewClusterName}")
            runcmd(f"kind create cluster --config {folReview}/kind-config.yaml --name={reviewClusterName}")
            runcmd(f"kind load docker-image mbg --name={reviewClusterName}")
        
        #Set First MBG
        useKindCluster(mbg1ClusterName)
        runcmdb(f'kubectl exec -i {podMbg1} -- ./mbg start --id "MBG1" --ip {mbg1Ip} --cport {mbg1cPort} --cportLocal {mbg1cPortLocal} --externalDataPortRange {mbg1DataPort} \
            --dataplane {args["dataplane"]}  {mbg1crtFlags}')
        runcmd(f"kubectl create service nodeport mbg --tcp={mbg1cPortLocal}:{mbg1cPortLocal} --node-port={mbg1cPort}")
        #Set Second MBG
        useKindCluster(mbg2ClusterName)
        runcmdb(f'kubectl exec -i {podMbg2} --  ./mbg start --id "MBG2" --ip {mbg2Ip} --cport {mbg2cPort} --cportLocal {mbg2cPortLocal} --externalDataPortRange {mbg2DataPort}\
        --dataplane {args["dataplane"]}  {mbg2crtFlags}')
        runcmd(f"kubectl create service nodeport mbg --tcp={mbg2cPortLocal}:{mbg2cPortLocal} --node-port={mbg2cPort}")
    
        ###Set product
        useKindCluster(productClusterName)
        runcmd(f"kind load docker-image maistra/examples-bookinfo-productpage-v1 --name={productClusterName}")
        runcmd(f"kind load docker-image maistra/examples-bookinfo-details-v1:0.12.0 --name={productClusterName}")
        runcmd(f"kubectl create -f {folpdct}/product.yaml")
        runcmd(f"kubectl create -f {folpdct}/details.yaml")
        podhost, hostIp= buildMbgctl("product Cluster", mbgMode)
        productMbgIp = f"{getPodIp(podMbg1)}:{mbg1cPortLocal}" if mbgMode =="inside" else f"{mbg1Ip}:{mbg1cPort}"
        runcmdb(f'kubectl exec -i {podhost} -- ./mbgctl start --id "productCluster"  --ip {hostIp} --mbgIP  {productMbgIp} --dataplane {args["dataplane"]} {productCrtFlags}')
        printHeader(f"Add {srcSvc} (client) service to host cluster")
        srcSvcIp =getPodIp(srcSvc)  if mbgMode =="inside" else srcDefaultGW
        runcmd(f'kubectl exec -i {podhost} -- ./mbgctl addService --serviceId {srcSvc} --serviceIp {srcSvcIp}')
        if mbgMode !="inside":
            runcmd(f"kubectl create -f {folpdct}/review-svc.yaml")
        # Add MBG Peer
        printHeader("Add MBG2 peer to MBG1")
        runcmd(f'kubectl exec -i {podhost} -- ./mbgctl addPeer --id "MBG2" --ip {mbg2Ip} --cport {mbg2cPort}')
    
        # Send Hello
        printHeader("Send Hello commands")
        runcmd(f'kubectl exec -i {podhost} -- ./mbgctl hello')
        
        ###Set dest
        useKindCluster(reviewClusterName)
        runcmd(f"kind load docker-image maistra/examples-bookinfo-reviews-v2 --name={reviewClusterName}")
        runcmd(f"kind load docker-image maistra/examples-bookinfo-reviews-v3 --name={reviewClusterName}")
        runcmd(f"kind load docker-image maistra/examples-bookinfo-ratings-v1:0.12.0 --name={reviewClusterName}")
        runcmd(f"kubectl create -f {folReview}/review-v3.yaml")
        runcmd(f"kubectl create service nodeport {review3svc} --tcp={srcK8sSvcPort}:{srcK8sSvcPort} --node-port={review3DestPort}")
        runcmd(f"kubectl create -f {folReview}/review-v2.yaml")
        runcmd(f"kubectl create service nodeport {review2svc} --tcp={srcK8sSvcPort}:{srcK8sSvcPort} --node-port={review2DestPort}")
        runcmd(f"kubectl create -f {folReview}/rating.yaml")
        podest, destIp= buildMbgctl("dest Cluster", mbgMode)   
        destMbgIp = f"{getPodIp(podMbg2)}:{mbg2cPortLocal}" if mbgMode =="inside" else f"{mbg2Ip}:{mbg2cPort}"
        runcmdb(f'kubectl exec -i {podest} -- ./mbgctl start --id "reviewCluster"  --ip {destIp} --mbgIP {destMbgIp} --dataplane {args["dataplane"]} {reviewCrtFlags}')
        printHeader(f"Add {review2svc} (server) service to destination cluster")
        waitPod(review2svc)
        waitPod(review3svc)
        destSvcReview2Ip = f"{getPodIp(review2svc)}:{srcK8sSvcPort}" if mbgMode =="inside" else f"{destIp}:{review2DestPort}"
        destSvcReview3Ip = f"{getPodIp(review3svc)}:{srcK8sSvcPort}" if mbgMode =="inside" else f"{destIp}:{review3DestPort}"
        runcmd(f'kubectl exec -i {podest} -- ./mbgctl addService --serviceId {review2svc} --serviceIp {destSvcReview2Ip}')
        runcmd(f'kubectl exec -i {podest} -- ./mbgctl addService --serviceId {review3svc} --serviceIp {destSvcReview3Ip}')

        #Add host cluster to MBG1
        useKindCluster(mbg1ClusterName)
        printHeader("Add host cluster to MBG1")
        runcmd(f'kubectl exec -i {podMbg1} -- ./mbg addMbgctl --id "productCluster" --ip {hostIp}')

        #Add dest cluster to MBG2
        useKindCluster(mbg2ClusterName)
        printHeader("Add dest cluster to MBG2")
        runcmd(f'kubectl exec -i {podMbg2} -- ./mbg addMbgctl --id "reviewCluster" --ip {destIp}')
        
        #Expose service
        useKindCluster(reviewClusterName)
        printHeader("\n\nStart exposing connection")
        runcmd(f'kubectl exec -i {podest} -- ./mbgctl expose --serviceId {review2svc}')
        runcmd(f'kubectl exec -i {podest} -- ./mbgctl expose --serviceId {review3svc}')

        #Get services
        useKindCluster(productClusterName)
        printHeader("\n\nStart get service")
        runcmdb(f'kubectl exec -i {podhost} -- ./mbgctl getService')
    
        #connect
        connectSvc(srcSvc, review2svc, srcK8sName, svcpolicy)
