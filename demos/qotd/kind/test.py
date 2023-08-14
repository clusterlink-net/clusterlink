#!/usr/bin/env python3
##############################################################################################
# Name: quote of today
# Info: support qotd application with gwctl inside the clusters 
#       In this we create three kind clusters
#       1) MBG1- contain mbg, gwctl,webApp and engravingApp microservices (qotd services)
#       2) MBG2- contain mbg, gwctl, quoteApp, authorApp, imageApp, dbApp microservices (qotd services)
#       3) MBG3- contain mbg, gwctl, pdfApp and ratingApp microservices (qotd services)
##############################################################################################

import os,time
import subprocess as sp
import sys
import argparse
proj_dir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{proj_dir}')

from demos.utils.mbgAux import printHeader, waitPod,getPodNameIp,runcmd,createMbgK8sService, app
from demos.utils.kind.kindAux import useKindCluster,startKindClusterMbg,getKindIp


############################### MAIN ##########################
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description='Description of your program')
    parser.add_argument('-d','--dataplane', help='choose which dataplane to use mtls/tcp', required=False, default="mtls")

    parser.add_argument('-src','--src', help='Source service name', required=False)
    parser.add_argument('-dst','--dest', help='Destination service name', required=False)
    args = vars(parser.parse_args())

    printHeader("\n\nStart Kind Test\n\n")
    printHeader("Start pre-setting")
    
    qotdFol   = f"{proj_dir}/demos/qotd/manifests/"
    
    dataplane = args["dataplane"]
    
    #MBG1 parameters 
    mbg1DataPort    = "30001"
    mbg1cPort       = "30443"
    mbg1cPortLocal  = "443"
    mbg1Name        = "mbg1"
    mbg1crtFlags    = f"--certca ./mtls/ca.crt --cert ./mtls/mbg1.crt --key ./mtls/mbg1.key"  if dataplane =="mtls" else ""
    gwctl1Name     = "gwctl1"   
    webApp       = app(name="qotd-web"      , namespace = "qotd-app-eks", host=""                                          , port=30010)
    engravingApp = app(name="qotd-engraving", namespace = "qotd"        , host=""                                          , port=3006)
    

    #MBG2 parameters 
    mbg2DataPort    = "30001"
    mbg2cPort       = "30443"
    mbg2cPortLocal  = "443"
    mbg2crtFlags    = f"--certca ./mtls/ca.crt --cert ./mtls/mbg2.crt --key ./mtls/mbg2.key"  if dataplane =="mtls" else ""
    mbg2Name        = "mbg2"
    gwctl2Name     = "gwctl2"
    quoteApp     = app(name="qotd-quote"    , namespace = "qotd-svc-iks", host="qotd-quote.qotd-svc-iks.svc.cluster.local" , port=3001)
    authorApp    = app(name="qotd-author"   , namespace = "qotd-svc-iks", host="qotd-author.qotd-svc-iks.svc.cluster.local", port=3002)
    imageApp     = app(name="qotd-image"    , namespace = "qotd"        , host="qotd-image.qotd.svc.cluster.local"         , port=3003)
    dbApp        = app(name="qotd-db"       , namespace = "qotd-svc-iks", host="qotd-db.qotd-svc-iks.svc.cluster.local"    , port=3306)

    #MBG3 parameters 
    mbg3DataPort    = "30001"
    mbg3cPort       = "30443"
    mbg3cPortLocal  = "443"
    mbg3crtFlags    = f"--certca ./mtls/ca.crt --cert ./mtls/mbg3.crt --key ./mtls/mbg3.key"  if dataplane =="mtls" else ""
    mbg3Name        = "mbg3"
    gwctl3Name     = "gwctl3"
    pdfApp       = app(name="qotd-pdf"      , namespace = "qotd-svc-ocp", host="qotd-pdf.qotd-svc-ocp.svc.cluster.local"   , port=3005)
    ratingApp    = app(name="qotd-rating"   , namespace = "qotd-svc-ocp", host="qotd-rating.qotd-svc-ocp.svc.cluster.local", port=3004)

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
    startKindClusterMbg(mbg1Name, gwctl1Name, mbg1cPortLocal, mbg1cPort, mbg1DataPort, dataplane ,mbg1crtFlags)        
    startKindClusterMbg(mbg2Name, gwctl2Name, mbg2cPortLocal, mbg2cPort, mbg2DataPort,dataplane ,mbg2crtFlags)        
    startKindClusterMbg(mbg3Name, gwctl3Name, mbg3cPortLocal, mbg3cPort, mbg3DataPort,dataplane ,mbg3crtFlags)        
    
    ###get mbg parameters
    useKindCluster(mbg1Name)
    mbg1Pod, _           = getPodNameIp("mbg")
    mbg1Ip               = getKindIp("mbg1")
    gwctl1Pod, gwctl1Ip= getPodNameIp("gwctl")
    useKindCluster(mbg2Name)
    mbg2Pod, _            = getPodNameIp("mbg")
    gwctl2Pod, gwctl2Ip = getPodNameIp("gwctl")
    mbg2Ip                =getKindIp(mbg2Name)
    useKindCluster(mbg3Name)
    mbg3Pod, _            = getPodNameIp("mbg")
    mbg3Ip                = getKindIp("mbg3")
    gwctl3Pod, gwctl3Ip = getPodNameIp("gwctl")

    # Add MBG Peer
    printHeader("Add GW2, GW3 peer to GW1")
    useKindCluster(mbg1Name)
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl create peer --name {mbg2Name} --host {mbg2Ip} --port {mbg2cPort}')
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl create peer --name {mbg3Name} --host {mbg3Ip} --port {mbg3cPort}')
    printHeader("Add GW1,GW3 peer to GW2")
    useKindCluster(mbg2Name)
    runcmd(f'kubectl exec -i {gwctl2Pod} -- ./gwctl create peer --name {mbg1Name} --host {mbg1Ip} --port {mbg1cPort}')
    runcmd(f'kubectl exec -i {gwctl2Pod} -- ./gwctl create peer --name {mbg3Name} --host {mbg3Ip} --port {mbg3cPort}')
    printHeader("Add GW1,GW2 peer to GW3")
    useKindCluster(mbg3Name)
    runcmd(f'kubectl exec -i {gwctl3Pod} -- ./gwctl create peer --name {mbg2Name} --host {mbg2Ip} --port {mbg2cPort}')
    runcmd(f'kubectl exec -i {gwctl3Pod} -- ./gwctl create peer --name {mbg1Name} --host {mbg1Ip} --port {mbg1cPort}')

    time.sleep(1)
    
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
    runcmd(f'kubectl exec -i {gwctl2Pod} -- ./gwctl create export --name {quoteApp.name}  --host {quoteApp.host}  --port {quoteApp.port}')
    runcmd(f'kubectl exec -i {gwctl2Pod} -- ./gwctl create export --name {authorApp.name} --host {authorApp.host} --port {authorApp.port}')
    runcmd(f'kubectl exec -i {gwctl2Pod} -- ./gwctl create export --name {dbApp.name}     --host {dbApp.host}     --port {dbApp.port}')
    runcmd(f'kubectl exec -i {gwctl2Pod} -- ./gwctl create export --name {imageApp.name}  --host {imageApp.host}  --port {imageApp.port}')
    

    ###Set gwctl3
    useKindCluster(mbg3Name)
    runcmd(f"kind load docker-image registry.gitlab.com/quote-of-the-day/qotd-ratings-service/v4.0.0:latest --name={mbg3Name}")
    runcmd(f"kind load docker-image registry.gitlab.com/quote-of-the-day/qotd-pdf-service/v4.0.0:latest --name={mbg3Name}")
    runcmd(f"kubectl create -f {qotdFol}/qotd_rating.yaml")
    runcmd(f"kubectl create -f {qotdFol}/qotd_pdf.yaml")
    printHeader(f"Add {ratingApp.name}, {pdfApp.name}, service to destination cluster")
    waitPod(pdfApp.name   , pdfApp.namespace)
    waitPod(ratingApp.name, ratingApp.namespace)
    runcmd(f'kubectl exec -i {gwctl3Pod} -- ./gwctl create export --name {pdfApp.name}    --host {pdfApp.host}    --port {pdfApp.port}')
    runcmd(f'kubectl exec -i {gwctl3Pod} -- ./gwctl create export --name {ratingApp.name} --host {ratingApp.host} --port {ratingApp.port}')

    # Import and binding Services
    useKindCluster(mbg1Name)
    printHeader(f"\n\nStart import and binding svc {quoteApp.name} to GW1 from GW2 ")
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl create import --name {quoteApp.name} --host {quoteApp.name} --port {quoteApp.port}')
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl create binding --import {quoteApp.name}  --peer {mbg2Name}')
    printHeader(f"\n\nStart import and binding svc {authorApp.name} to GW1 from GW2")
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl create import --name {authorApp.name} --host {authorApp.name} --port {authorApp.port}')
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl create binding --import {authorApp.name}  --peer {mbg2Name}')
    printHeader(f"\n\nStart import and binding svc {dbApp.name} to GW1 from GW2")
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl create import --name {dbApp.name} --host {dbApp.name} --port {dbApp.port}')
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl create binding --import {dbApp.name}  --peer {mbg2Name}')
    printHeader(f"\n\nStart import and binding svc {imageApp.name} to GW1 from GW2")
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl create import --name {imageApp.name} --host {imageApp.name} --port {imageApp.port}')
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl create binding --import {imageApp.name}  --peer {mbg2Name}')
    
    printHeader(f"\n\nStart import and binding svc {pdfApp.name} to GW1 from GW3")
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl create import --name {pdfApp.name} --host {pdfApp.name} --port {pdfApp.port}')
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl create binding --import {pdfApp.name} --peer {mbg3Name}')
    printHeader(f"\n\nStart import and binding svc {ratingApp.name} to GW1 from GW3")
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl create import --name {ratingApp.name} --host {ratingApp.name} --port {ratingApp.port}')
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl create binding --import {ratingApp.name}  --peer {mbg3Name}')


   
    useKindCluster(mbg3Name)
    printHeader(f"\n\nStart import and binding svc {quoteApp.name} in GW3")
    runcmd(f'kubectl exec -i {gwctl3Pod} -- ./gwctl create import --name {quoteApp.name} --host {quoteApp.name} --port {quoteApp.port}')
    runcmd(f'kubectl exec -i {gwctl3Pod} -- ./gwctl create binding --import {quoteApp.name}  --peer {mbg2Name}')
    
    # Get service and policies
    useKindCluster(mbg1Name)
    printHeader("\n\nStart get import, binding and policy")
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl get import --myid {gwctl1Name}')
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl get binding --myid {gwctl1Name} --import {quoteApp.name}')
    runcmd(f'kubectl exec -i {gwctl1Pod} -- ./gwctl get policy --myid {gwctl1Name}')


    webApp.host=mbg1Ip
    print(f"Application url: http://{webApp.host}:{webApp.port}")
    


