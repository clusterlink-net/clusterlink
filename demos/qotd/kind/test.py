#!/usr/bin/env python3
# Copyright 2023 The ClusterLink Authors.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

##############################################################################################
# Name: quote of today
# Info: support qotd application with gwctl inside the clusters 
#       In this we create three kind clusters
#       1) cluster1- contain gw, gwctl,webApp and engravingApp microservices (qotd services)
#       2) cluster2- contain gw, gwctl, quoteApp, authorApp, imageApp, dbApp microservices (qotd services)
#       3) cluster3- contain gw, gwctl, pdfApp and ratingApp microservices (qotd services)
##############################################################################################
import os
import sys

projDir = os.path.dirname(os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__)))))
sys.path.insert(0,f'{projDir}')

from demos.utils.common import runcmd, createFabric, printHeader,app, ProjDir
from demos.utils.kind import cluster
from demos.utils.k8s import getPodNameIp

############################### MAIN ##########################
if __name__ == "__main__":
    printHeader("\n\nStart Kind Test\n\n")
    printHeader("Start pre-setting")
    
    #GW parameters 
    gwPort       = "30443"    
    gwNS         = "default"    
    cl1          = cluster(name="peer1")
    webApp       = app(name="qotd-web"      , namespace = "qotd-app-eks", host=""                                          , port=30010)
    engravingApp = app(name="qotd-engraving", namespace = "qotd"        , host=""                                          , port=3006)

    cl2          = cluster(name="peer2")
    quoteApp     = app(name="qotd-quote"    , namespace = "qotd-svc-iks", host="qotd-quote.qotd-svc-iks.svc.cluster.local" , port=3001)
    authorApp    = app(name="qotd-author"   , namespace = "qotd-svc-iks", host="qotd-author.qotd-svc-iks.svc.cluster.local", port=3002)
    imageApp     = app(name="qotd-image"    , namespace = "qotd"        , host="qotd-image.qotd.svc.cluster.local"         , port=3003)
    dbApp        = app(name="qotd-db"       , namespace = "qotd-svc-iks", host="qotd-db.qotd-svc-iks.svc.cluster.local"    , port=3306)

    cl3          = cluster(name="peer3")
    pdfApp       = app(name="qotd-pdf"      , namespace = "qotd-svc-ocp", host="qotd-pdf.qotd-svc-ocp.svc.cluster.local"   , port=3005)
    ratingApp    = app(name="qotd-rating"   , namespace = "qotd-svc-ocp", host="qotd-rating.qotd-svc-ocp.svc.cluster.local", port=3004)

    # Folders
    qotdFol    = f"{ProjDir}/demos/qotd/manifests/"
    allowAllPolicy =f"{ProjDir}/pkg/policyengine/examples/allowAll.json"
    testOutputFolder = f"{ProjDir}/bin/tests/qotd"    

    print(f'Working directory {ProjDir}')
    runcmd(ProjDir)

    ### build docker environment 
    printHeader("Build docker image")
    runcmd("make docker-build")
    
    # Create Kind clusters environment 
    cl1.createCluster(runBg=True)        
    cl2.createCluster(runBg=True)
    cl3.createCluster(runBg=False)  

    # Start Kind clusters environment 
    createFabric(testOutputFolder) 
    cl1.startCluster(testOutputFolder)        
    cl2.startCluster(testOutputFolder)        
    cl3.startCluster(testOutputFolder)        
       
    ###get gw parameters
    cl1.useCluster()
    gwctl1Pod, gwctl1Ip = getPodNameIp("gwctl")
    cl2.useCluster()
    gwctl2Pod, gwctl2Ip = getPodNameIp("gwctl")
    cl3.useCluster()
    gwctl3Pod, gwctl3Ip = getPodNameIp("gwctl")

    # Add Peer
    printHeader("Add cl2, cl3 peer to cl1")
    cl1.useCluster()
    runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl create peer --name {cl2.name} --host {cl2.ip} --port {gwPort}')
    runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl create peer --name {cl3.name} --host {cl3.ip} --port {gwPort}')
    printHeader("Add cl1,cl3 peer to cl2")
    cl2.useCluster()
    runcmd(f'kubectl exec -i {gwctl2Pod} -- gwctl create peer --name {cl1.name} --host {cl1.ip} --port {gwPort}')
    runcmd(f'kubectl exec -i {gwctl2Pod} -- gwctl create peer --name {cl3.name} --host {cl3.ip} --port {gwPort}')
    printHeader("Add cl1,cl2 peer to cl3")
    cl3.useCluster()
    runcmd(f'kubectl exec -i {gwctl3Pod} -- gwctl create peer --name {cl2.name} --host {cl2.ip} --port {gwPort}')
    runcmd(f'kubectl exec -i {gwctl3Pod} -- gwctl create peer --name {cl1.name} --host {cl1.ip} --port {gwPort}')
    
    ###Set cl1 services
    cl1.useCluster()
    cl1.loadService(webApp.name, "registry.gitlab.com/quote-of-the-day/qotd-web/v4.0.0:latest" , 
                f"{qotdFol}/qotd_web.yaml", namespace= webApp.namespace)
    cl1.loadService(engravingApp.name, "registry.gitlab.com/quote-of-the-day/qotd-engraving-service/v4.0.0:latest", 
                f"{qotdFol}/qotd_engraving.yaml", namespace= engravingApp.namespace)
    
    ###Set cl2 service
    cl2.useCluster()
    cl2.loadService(imageApp.name, "registry.gitlab.com/quote-of-the-day/qotd-image-service/v4.0.0:latest", 
                f"{qotdFol}/qotd_image.yaml", namespace= imageApp.namespace)
    cl2.loadService(quoteApp.name, "registry.gitlab.com/quote-of-the-day/quote-service/v4.0.0:latest", 
                f"{qotdFol}/qotd_quote.yaml", namespace= quoteApp.namespace)
    cl2.loadService(authorApp.name, "registry.gitlab.com/quote-of-the-day/qotd-author-service/v4.0.0:latest", 
                f"{qotdFol}/qotd_author.yaml", namespace= authorApp.namespace)
    cl2.loadService(dbApp.name, "registry.gitlab.com/quote-of-the-day/qotd-db/v4.0.0:latest", 
                f"{qotdFol}/qotd_db.yaml", namespace= dbApp.namespace)
    
    runcmd(f'kubectl exec -i {gwctl2Pod} -- gwctl create export --name {quoteApp.name}  --host {quoteApp.host}  --port {quoteApp.port}')
    runcmd(f'kubectl exec -i {gwctl2Pod} -- gwctl create export --name {authorApp.name} --host {authorApp.host} --port {authorApp.port}')
    runcmd(f'kubectl exec -i {gwctl2Pod} -- gwctl create export --name {dbApp.name}     --host {dbApp.host}     --port {dbApp.port}')
    runcmd(f'kubectl exec -i {gwctl2Pod} -- gwctl create export --name {imageApp.name}  --host {imageApp.host}  --port {imageApp.port}')
    

    ###Set gwctl3
    cl3.useCluster()
    cl3.loadService(ratingApp.name, "registry.gitlab.com/quote-of-the-day/qotd-ratings-service/v4.0.0:latest", 
                f"{qotdFol}/qotd_rating.yaml", namespace= ratingApp.namespace)
    cl3.loadService(pdfApp.name, "registry.gitlab.com/quote-of-the-day/qotd-pdf-service/v4.0.0:latest", 
                f"{qotdFol}/qotd_pdf.yaml", namespace= pdfApp.namespace)
    
    runcmd(f'kubectl exec -i {gwctl3Pod} -- gwctl create export --name {pdfApp.name}    --host {pdfApp.host}    --port {pdfApp.port}')
    runcmd(f'kubectl exec -i {gwctl3Pod} -- gwctl create export --name {ratingApp.name} --host {ratingApp.host} --port {ratingApp.port}')

    # Import and binding Services
    cl1.useCluster()
    printHeader(f"\n\nStart import svc {quoteApp.name} to cl1 from cl2 ")
    runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl create import --name {quoteApp.name} --port {quoteApp.port} --peer {cl2.name}')
    printHeader(f"\n\nStart import svc {authorApp.name} to cl1 from cl2")
    runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl create import --name {authorApp.name} --port {authorApp.port} --peer {cl2.name}')
    printHeader(f"\n\nStart import svc {dbApp.name} to cl1 from cl2")
    runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl create import --name {dbApp.name} --port {dbApp.port} --peer {cl2.name}')
    printHeader(f"\n\nStart import svc {imageApp.name} to cl1 from cl2")
    runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl create import --name {imageApp.name} --port {imageApp.port} --peer {cl2.name}')
    
    printHeader(f"\n\nStart import svc {pdfApp.name} to cl1 from cl3")
    runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl create import --name {pdfApp.name} --port {pdfApp.port} --peer {cl3.name}')
    printHeader(f"\n\nStart import svc {ratingApp.name} to cl1 from cl3")
    runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl create import --name {ratingApp.name} --port {ratingApp.port} --peer {cl3.name}')

    cl3.useCluster()
    printHeader(f"\n\nStart import svc {quoteApp.name} in cl3")
    runcmd(f'kubectl exec -i {gwctl3Pod} -- gwctl create import --name {quoteApp.name} --port {quoteApp.port} --peer {cl2.name}')
    
    # Set policies
    printHeader(f"\n\nApplying policy file {allowAllPolicy}")
    policyFile ="/tmp/allowAll.json"
    cl1.useCluster()
    runcmd(f'kubectl cp {allowAllPolicy} gwctl:{policyFile}')
    runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl create policy --type access --policyFile {policyFile}')
    cl2.useCluster()
    runcmd(f'kubectl cp {allowAllPolicy} gwctl:{policyFile}')
    runcmd(f'kubectl exec -i {gwctl2Pod} -- gwctl create policy --type access --policyFile {policyFile}')
    cl3.useCluster()
    runcmd(f'kubectl cp {allowAllPolicy} gwctl:{policyFile}')
    runcmd(f'kubectl exec -i {gwctl3Pod} -- gwctl create policy --type access --policyFile {policyFile}')
    
    # Get service and policies
    cl1.useCluster()
    printHeader("\n\nStart get import, binding and policy")
    runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl get import')
    runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl get binding --import {quoteApp.name}')
    runcmd(f'kubectl exec -i {gwctl1Pod} -- gwctl get policy')

    webApp.host=cl1.ip
    print(f"Application url: http://{webApp.host}:{webApp.port}")
    


