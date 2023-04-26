#!/usr/bin/env python3
##############################################################################################
# Name: quote of today
# Info: support qotd application with mbgctl inside the clusters 
#       In this we create three kind clusters
#       1) MBG1- contain mbg, mbgctl,webApp and engravingApp microservices (qotd services)
#       2) MBG2- contain mbg, mbgctl, quoteApp, authorApp, imageApp, dbApp microservices (qotd services)
#       3) MBG3- contain mbg, mbgctl, pdfApp and ratingApp microservices (qotd services)
##############################################################################################

import os,time
import subprocess as sp
import sys
import argparse
proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

from tests.utils.mbgAux import printHeader, waitPod,getPodNameIp,app
from tests.utils.kind.kindAux import useKindCluster,startKindClusterMbg,getKindIp


############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-d','--dataplane', help='choose which dataplane to use mtls/tcp', required=False, default="mtls")

    parser.add_argument('-src','--src', help='Source service name', required=False)
    parser.add_argument('-dst','--dest', help='Destination service name', required=False)
    args = vars(parser.parse_args())

    printHeader("\n\nStart Kind Test\n\n")
    printHeader("Start pre-setting")
    
    qotdFol   = f"{proj_dir}/tests/qotd/manifests/"
    
    dataplane = args["dataplane"]
    
    #MBG1 parameters 
    mbg1DataPort    = "30001"
    mbg1cPort       = "30443"
    mbg1cPortLocal  = "8443"
    mbg1Name        = "mbg1"
    mbg1crtFlags    = f"--rootCa ./mtls/ca.crt --certificate ./mtls/mbg1.crt --key ./mtls/mbg1.key"  if dataplane =="mtls" else ""
    mbgctl1Name     = "mbgctl1"   
    webApp       = app(name="qotd-web"      , namespace = "qotd-app-eks", target=""                                          , port=30010)
    engravingApp = app(name="qotd-engraving", namespace = "qotd"        , target=""                                          , port=3006)
    

    #MBG2 parameters 
    mbg2DataPort    = "30001"
    mbg2cPort       = "30443"
    mbg2cPortLocal  = "8443"
    mbg2crtFlags    = f"--rootCa ./mtls/ca.crt --certificate ./mtls/mbg2.crt --key ./mtls/mbg2.key"  if dataplane =="mtls" else ""
    mbg2Name        = "mbg2"
    mbgctl2Name     = "mbgctl2"
    quoteApp     = app(name="qotd-quote"    , namespace = "qotd-svc-iks", target="qotd-quote.qotd-svc-iks.svc.cluster.local" , port=3001)
    authorApp    = app(name="qotd-author"   , namespace = "qotd-svc-iks", target="qotd-author.qotd-svc-iks.svc.cluster.local", port=3002)
    imageApp     = app(name="qotd-image"    , namespace = "qotd"        , target="qotd-image.qotd.svc.cluster.local"         , port=3003)
    dbApp        = app(name="qotd-db"       , namespace = "qotd-svc-iks", target="qotd-db.qotd-svc-iks.svc.cluster.local"    , port=3306)

    #MBG3 parameters 
    mbg3DataPort    = "30001"
    mbg3cPort       = "30443"
    mbg3cPortLocal  = "8443"
    mbg3crtFlags    = f"--rootCa ./mtls/ca.crt --certificate ./mtls/mbg3.crt --key ./mtls/mbg3.key"  if dataplane =="mtls" else ""
    mbg3Name        = "mbg3"
    mbgctl3Name     = "mbgctl3"
    pdfApp       = app(name="qotd-pdf"      , namespace = "qotd-svc-ocp", target="qotd-pdf.qotd-svc-ocp.svc.cluster.local"   , port=3005)
    ratingApp    = app(name="qotd-rating"   , namespace = "qotd-svc-ocp", target="qotd-rating.qotd-svc-ocp.svc.cluster.local", port=3004)

    mbgNS="default"    

    print(f'Working directory {proj_dir}')
    os.chdir(proj_dir)

    ### clean 
    print(f"Clean old kinds")
    os.system("make clean-kind")
    
    ### build docker environment 
    printHeader(f"Build docker image")
    os.system("make docker-build")
    
    ## build Kind clusters environment 
    startKindClusterMbg(mbg1Name, mbgctl1Name, mbg1cPortLocal, mbg1cPort, mbg1DataPort, dataplane ,mbg1crtFlags)        
    startKindClusterMbg(mbg2Name, mbgctl2Name, mbg2cPortLocal, mbg2cPort, mbg2DataPort,dataplane ,mbg2crtFlags)        
    startKindClusterMbg(mbg3Name, mbgctl3Name, mbg3cPortLocal, mbg3cPort, mbg3DataPort,dataplane ,mbg3crtFlags)        
    
    ###get mbg parameters
    useKindCluster(mbg1Name)
    mbg1Pod, _           = getPodNameIp("mbg")
    mbg1Ip               = getKindIp("mbg1")
    mbgctl1Pod, mbgctl1Ip= getPodNameIp("mbgctl")
    useKindCluster(mbg2Name)
    mbg2Pod, _            = getPodNameIp("mbg")
    mbgctl2Pod, mbgctl2Ip = getPodNameIp("mbgctl")
    mbg2Ip                =getKindIp(mbg2Name)
    useKindCluster(mbg3Name)
    mbg3Pod, _            = getPodNameIp("mbg")
    mbg3Ip                = getKindIp("mbg3")
    mbgctl3Pod, mbgctl3Ip = getPodNameIp("mbgctl")

    # Add MBG Peer
    printHeader("Add MBG2, MBG3 peer to MBG1")
    useKindCluster(mbg1Name)
    runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl add peer --id {mbg2Name} --target {mbg2Ip} --port {mbg2cPort}')
    runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl add peer --id {mbg3Name} --target {mbg3Ip} --port {mbg3cPort}')
    useKindCluster(mbg2Name)
    runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl add peer --id {mbg3Name} --target {mbg3Ip} --port {mbg3cPort}')
    # Send Hello
    printHeader("Send Hello commands")
    useKindCluster(mbg1Name)
    runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl hello')
    useKindCluster(mbg2Name)
    runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl hello')
    
    ###Set mbg1 services
    useKindCluster(mbg1Name)
    runcmd(f"kind load docker-image registry.gitlab.com/quote-of-the-day/qotd-engraving-service/v4.0.0:latest --name={mbg1Name}")
    runcmd(f"kind load docker-image registry.gitlab.com/quote-of-the-day/qotd-web/v4.0.0:latest --name={mbg1Name}")
    runcmd(f"kubectl create -f {qotdFol}/qotd_engraving.yaml")
    runcmd(f"kubectl create -f {qotdFol}/qotd_web.yaml")
    printHeader(f"Add {webApp.name} {engravingApp}.name  services to host cluster")
    waitPod(webApp.name, webApp.namespace)
    waitPod(engravingApp.name, engravingApp.namespace)
    
    ###Set mbg2 service
    useKindCluster(mbg2Name)
    runcmd(f"kind load docker-image registry.gitlab.com/quote-of-the-day/qotd-image-service/v4.0.0:latest --name={mbg2Name}")
    runcmd(f"kind load docker-image registry.gitlab.com/quote-of-the-day/quote-service/v4.0.0:latest --name={mbg2Name}")
    runcmd(f"kind load docker-image registry.gitlab.com/quote-of-the-day/qotd-author-service/v4.0.0:latest --name={mbg2Name}")
    runcmd(f"kind load docker-image registry.gitlab.com/quote-of-the-day/qotd-db/v4.0.0:latest --name={mbg2Name}")
    runcmd(f"kubectl create -f {qotdFol}/qotd_image.yaml")
    runcmd(f"kubectl create -f {qotdFol}/qotd_quote.yaml")
    runcmd(f"kubectl create -f {qotdFol}/qotd_author.yaml")
    runcmd(f"kubectl create -f {qotdFol}/qotd_db.yaml")
    printHeader(f"Add {imageApp.name}, {quoteApp.name}, {authorApp.name}, {dbApp.name} service to destination cluster")
    waitPod(imageApp.name , imageApp.namespace)
    waitPod(quoteApp.name , quoteApp.namespace)
    waitPod(authorApp.name, authorApp.namespace)
    waitPod(dbApp.name    , dbApp.namespace)
    runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl add service --id {quoteApp.name}  --target {quoteApp.target}  --port {quoteApp.port}')
    runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl add service --id {authorApp.name} --target {authorApp.target} --port {authorApp.port}')
    runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl add service --id {dbApp.name}     --target {dbApp.target}     --port {dbApp.port}')
    runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl add service --id {imageApp.name}  --target {imageApp.target}  --port {imageApp.port}')
    

    ###Set mbgctl3
    useKindCluster(mbg3Name)
    runcmd(f"kind load docker-image registry.gitlab.com/quote-of-the-day/qotd-ratings-service/v4.0.0:latest --name={mbg3Name}")
    runcmd(f"kind load docker-image registry.gitlab.com/quote-of-the-day/qotd-pdf-service/v4.0.0:latest --name={mbg3Name}")
    runcmd(f"kubectl create -f {qotdFol}/qotd_rating.yaml")
    runcmd(f"kubectl create -f {qotdFol}/qotd_pdf.yaml")
    printHeader(f"Add {ratingApp.name}, {pdfApp.name}, service to destination cluster")
    waitPod(pdfApp.name   , pdfApp.namespace)
    waitPod(ratingApp.name, ratingApp.namespace)
    runcmd(f'kubectl exec -i {mbgctl3Pod} -- ./mbgctl add service --id {pdfApp.name}    --target {pdfApp.target}    --port {pdfApp.port}')
    runcmd(f'kubectl exec -i {mbgctl3Pod} -- ./mbgctl add service --id {ratingApp.name} --target {ratingApp.target} --port {ratingApp.port}')

    # Expose service
    useKindCluster(mbg2Name)
    printHeader(f"\n\nStart exposing svc {quoteApp.name}")
    runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl expose --service {quoteApp.name}')
    printHeader(f"\n\nStart exposing svc {authorApp.name}")
    runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl expose --service {authorApp.name} --peer {mbg1Name}')
    printHeader(f"\n\nStart exposing svc {dbApp.name}")
    runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl expose --service {dbApp.name} --peer {mbg1Name}')
    printHeader(f"\n\nStart exposing svc {imageApp.name}")
    runcmd(f'kubectl exec -i {mbgctl2Pod} -- ./mbgctl expose --service {imageApp.name} --peer {mbg1Name}')

    useKindCluster(mbg3Name)
    printHeader(f"\n\nStart exposing svc {pdfApp.name}")
    runcmd(f'kubectl exec -i {mbgctl3Pod} -- ./mbgctl expose --service {pdfApp.name} --peer {mbg1Name}')
    printHeader(f"\n\nStart exposing svc {ratingApp.name}")
    runcmd(f'kubectl exec -i {mbgctl3Pod} -- ./mbgctl expose --service {ratingApp.name} --peer {mbg1Name}')
   
    #Get services
    useKindCluster(mbg1Name)
    printHeader("\n\nStart get service")
    runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl get service --myid {mbgctl1Name}')
    runcmd(f'kubectl exec -i {mbgctl1Pod} -- ./mbgctl get policy --myid {mbgctl1Name}')

    #Create k8s service
    useKindCluster(mbg1Name)    
    creatMbgK8sService(quoteApp.name  , quoteApp.name ,mbgNS, quoteApp.port)
    creatMbgK8sService(authorApp.name , authorApp.name, mbgNS, authorApp.port)
    creatMbgK8sService(dbApp.name     , dbApp.name    , mbgNS, dbApp.port)
    creatMbgK8sService(imageApp.name  , imageApp.name , mbgNS, imageApp.port)
    creatMbgK8sService(pdfApp.name    , pdfApp.name   , mbgNS, pdfApp.port)
    creatMbgK8sService(ratingApp.name , ratingApp.name   , mbgNS, ratingApp.port)

    useKindCluster(mbg3Name)    
    creatMbgK8sService(quoteApp.name  , quoteApp.name , mbgNS, quoteApp.port)

    webApp.target=mbg1Ip
    print(f"Application url: http://{webApp.target}:{webApp.port}")
    


