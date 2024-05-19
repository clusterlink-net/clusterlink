#!/usr/bin/env python3
# Copyright (c) The ClusterLink Authors.
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

from demos.utils.common import runcmd, printHeader,app, ProjDir
from demos.utils.kind import Cluster
############################### MAIN ##########################
if __name__ == "__main__":
    printHeader("\n\nStart Kind Test\n\n")
    printHeader("Start pre-setting")

    #GW parameters
    gwPort       = "30443"
    gwNS         = "default"
    cl1          = Cluster(name="peer1")
    webApp       = app(name="qotd-web"      , namespace = "qotd-app-eks", host=""                                          , port=30010)
    engravingApp = app(name="qotd-engraving", namespace = "qotd"        , host=""                                          , port=3006)

    cl2          = Cluster(name="peer2")
    quoteApp     = app(name="qotd-quote"    , namespace = "qotd-svc-iks", host="qotd-quote.qotd-svc-iks.svc.cluster.local" , port=3001)
    authorApp    = app(name="qotd-author"   , namespace = "qotd-svc-iks", host="qotd-author.qotd-svc-iks.svc.cluster.local", port=3002)
    imageApp     = app(name="qotd-image"    , namespace = "qotd"        , host="qotd-image.qotd.svc.cluster.local"         , port=3003)
    dbApp        = app(name="qotd-db"       , namespace = "qotd-svc-iks", host="qotd-db.qotd-svc-iks.svc.cluster.local"    , port=3306)

    cl3          = Cluster(name="peer3")
    pdfApp       = app(name="qotd-pdf"      , namespace = "qotd-svc-ocp", host="qotd-pdf.qotd-svc-ocp.svc.cluster.local"   , port=3005)
    ratingApp    = app(name="qotd-rating"   , namespace = "qotd-svc-ocp", host="qotd-rating.qotd-svc-ocp.svc.cluster.local", port=3004)

    # Folders
    qotdFol    = f"{ProjDir}/demos/qotd/manifests/"
    testOutputFolder = f"{ProjDir}/bin/tests/qotd"

    print(f'Working directory {ProjDir}')
    runcmd(f"cd {ProjDir}")

    ### build docker environment
    printHeader("Build docker image")
    runcmd("make docker-build")

    # Create Kind clusters environment
    cl1.createCluster(runBg=True)
    cl2.createCluster(runBg=True)
    cl3.createCluster(runBg=False)

    # Start Kind clusters environment
    cl1.create_fabric(testOutputFolder)
    cl1.startCluster(testOutputFolder)
    cl2.startCluster(testOutputFolder)
    cl3.startCluster(testOutputFolder)

    # Add Peer
    printHeader("Add cl2, cl3 peer to cl1")
    cl1.peers.create(cl2.name, cl2.ip, cl2.port)
    cl1.peers.create(cl3.name, cl3.ip, cl3.port)
    printHeader("Add cl1,cl3 peer to cl2")
    cl2.peers.create(cl1.name, cl1.ip, cl1.port)
    cl2.peers.create(cl3.name, cl3.ip, cl3.port)
    printHeader("Add cl1,cl2 peer to cl3")
    cl3.peers.create(cl1.name, cl1.ip, cl1.port)
    cl3.peers.create(cl2.name, cl2.ip, cl2.port)

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

    cl2.exports.create(quoteApp.name,  quoteApp.namespace,  quoteApp.port)
    cl2.exports.create(authorApp.name, authorApp.namespace, authorApp.port)
    cl2.exports.create(dbApp.name,     dbApp.namespace,     dbApp.port)
    cl2.exports.create(imageApp.name,  imageApp.namespace,  imageApp.port)

    ###Set gwctl3
    cl3.useCluster()
    cl3.loadService(ratingApp.name, "registry.gitlab.com/quote-of-the-day/qotd-ratings-service/v4.0.0:latest",
                f"{qotdFol}/qotd_rating.yaml", namespace= ratingApp.namespace)
    cl3.loadService(pdfApp.name, "registry.gitlab.com/quote-of-the-day/qotd-pdf-service/v4.0.0:latest",
                f"{qotdFol}/qotd_pdf.yaml", namespace= pdfApp.namespace)

    cl3.exports.create(pdfApp.name,    pdfApp.namespace,    pdfApp.port)
    cl3.exports.create(ratingApp.name, ratingApp.namespace, ratingApp.port)

    # Import and binding Services
    cl1.useCluster()
    printHeader(f"\n\nStart import svc {quoteApp.name} to cl1 from cl2 ")
    cl1.imports.create(quoteApp.name,  webApp.namespace, quoteApp.port,  cl2.name, quoteApp.name,  quoteApp.namespace)
    printHeader(f"\n\nStart import svc {authorApp.name} to cl1 from cl2")
    cl1.imports.create(authorApp.name, webApp.namespace, authorApp.port, cl2.name, authorApp.name, authorApp.namespace)
    printHeader(f"\n\nStart import svc {dbApp.name} to cl1 from cl2")
    cl1.imports.create(dbApp.name,     webApp.namespace, dbApp.port,     cl2.name, dbApp.name,     dbApp.namespace)
    printHeader(f"\n\nStart import svc {imageApp.name} to cl1 from cl2")
    cl1.imports.create(imageApp.name,  webApp.namespace, imageApp.port,  cl2.name, imageApp.name,  imageApp.namespace)
    printHeader(f"\n\nStart import svc {pdfApp.name} to cl1 from cl3")
    cl1.imports.create(pdfApp.name,    webApp.namespace, pdfApp.port,    cl3.name, pdfApp.name,    pdfApp.namespace)
    printHeader(f"\n\nStart import svc {ratingApp.name} to cl1 from cl3")
    cl1.imports.create(ratingApp.name, webApp.namespace, ratingApp.port, cl3.name, ratingApp.name, ratingApp.namespace)

    printHeader(f"\n\nStart import svc {quoteApp.name} to cl3 from cl2")
    cl3.imports.create(quoteApp.name,  pdfApp.namespace, quoteApp.port,  cl2.name, quoteApp.name,  quoteApp.namespace)

    # Set privileged_policies
    printHeader(f"\n\nApplying allow-all policy")
    cl1.policies.create(name="allow-all",namespace= webApp.namespace, action="allow", from_attribute=[{"workloadSelector": {}}],to_attribute=[{"workloadSelector": {}}])
    cl2.policies.create(name="allow-all",namespace= quoteApp.namespace, action="allow", from_attribute=[{"workloadSelector": {}}],to_attribute=[{"workloadSelector": {}}])
    cl2.policies.create(name="allow-all",namespace= imageApp.namespace, action="allow", from_attribute=[{"workloadSelector": {}}],to_attribute=[{"workloadSelector": {}}])
    cl3.policies.create(name="allow-all",namespace= pdfApp.namespace, action="allow", from_attribute=[{"workloadSelector": {}}],to_attribute=[{"workloadSelector": {}}])

    # Get service and policies
    cl1.useCluster()
    printHeader("\n\nStart get import, binding and policy")
    runcmd(f'kubectl get imports.clusterlink.net')
    runcmd(f'kubectl get accesspolicies.clusterlink.net')

    webApp.host=cl1.ip
    print(f"Application url: http://{webApp.host}:{webApp.port}")



