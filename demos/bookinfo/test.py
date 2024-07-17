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
# Name: Bookinfo
# Info: support bookinfo application inside the clusters
#       In this we create three kind clusters
#       1) cluster1- contain gw, product and details microservices (bookinfo services)
#       2) cluster2- contain gw, review-v2 and rating microservices (bookinfo services)
#       3) cluster3- contain gw, review-v3 and rating microservices (bookinfo services)
##############################################################################################

import os
import sys
projDir = os.path.dirname(os.path.dirname(os.path.dirname( os.path.abspath(__file__))))
sys.path.insert(0,f'{projDir}')

from demos.utils.common import runcmd, printHeader
from demos.utils.kind import Cluster
from demos.utils.k8s import getPodIp
from demos.utils.clusterlink import CLUSTELINK_OPERATOR_NS


folpdct   = f"{projDir}/demos/bookinfo/manifests/product/"
folReview = f"{projDir}/demos/bookinfo/manifests/review"

reviewSvc     = "reviews"
srcSvc1       = "productpage"
srcSvc2       = "productpage2"
srcK8sSvcPort = 9080
review2pod    = "reviews-v2"
review3pod    = "reviews-v3"
namespace     = "default"

# bookInfoDemo runs the bookinfo demo.
def bookInfoDemo(cl1:Cluster, cl2:Cluster, cl3:Cluster, testOutputFolder,logLevel="info" ,dataplane="envoy"):
    print(f'Working directory {projDir}')
    os.chdir(projDir)
    ### build docker environment
    printHeader("Build docker image")
    runcmd("make docker-build")

    # Create Kind clusters environment
    cl1.createCluster(runBg=True)
    cl2.createCluster(runBg=True)
    cl3.createCluster(runBg=False)

    # Start Kind clusters environment
    cl1.create_fabric(testOutputFolder)
    cl1.startCluster(testOutputFolder, logLevel, dataplane)
    cl2.startCluster(testOutputFolder, logLevel, dataplane)
    cl3.startCluster(testOutputFolder, logLevel, dataplane)

    # Set GW services
    cl1.useCluster()
    cl1.loadService(srcSvc1, "istio/examples-bookinfo-productpage-v1",f"{folpdct}/product.yaml")
    cl1.loadService(srcSvc2, "istio/examples-bookinfo-productpage-v1",f"{folpdct}/product2.yaml")
    cl1.loadService(srcSvc1, "istio/examples-bookinfo-details-v1",f"{folpdct}/details.yaml")
    cl2.useCluster()
    cl2.loadService(reviewSvc, "istio/examples-bookinfo-reviews-v2",f"{folReview}/review-v2.yaml")
    cl2.loadService("ratings", "istio/examples-bookinfo-ratings-v1",f"{folReview}/rating.yaml")
    cl3.useCluster()
    cl3.loadService(reviewSvc, "istio/examples-bookinfo-reviews-v3",f"{folReview}/review-v3.yaml")
    cl3.loadService("ratings", "istio/examples-bookinfo-ratings-v1",f"{folReview}/rating.yaml")

    # Add GW Peers
    printHeader("Add cl2, cl3 peer to cl1")
    cl1.useCluster()
    cl1.peers.create(cl2.name, cl2.ip, cl2.port)
    cl1.peers.create(cl3.name, cl3.ip, cl3.port)
    printHeader("Add cl1 peer to cl2")
    cl2.useCluster()
    cl2.peers.create(cl1.name, cl1.ip, cl1.port)
    cl3.useCluster()
    printHeader("Add cl3 peer to cl1")
    cl3.peers.create(cl1.name, cl1.ip, cl1.port)

    # Set exports
    cl1.useCluster()
    printHeader(f"create exports {srcSvc1} {srcSvc2}")
    cl1.exports.create(srcSvc1, namespace,  srcK8sSvcPort)
    cl1.exports.create(srcSvc2, namespace,  srcK8sSvcPort)
    cl2.useCluster()
    review2Ip = f"{getPodIp(reviewSvc)}"
    review2Port = srcK8sSvcPort
    cl2.exports.create(reviewSvc, namespace, host=review2Ip, port=review2Port)

    cl3.useCluster()
    review3Ip = f"{getPodIp(reviewSvc)}"
    review3Port = srcK8sSvcPort
    cl3.exports.create(reviewSvc, namespace, host=review3Ip, port=review3Port)

    # Import service
    cl1.useCluster()
    printHeader(f"\n\nStart import svc {reviewSvc}")
    cl1.imports.create(reviewSvc,  namespace, srcK8sSvcPort,  [cl2.name,cl3.name], [reviewSvc,reviewSvc], [namespace,namespace])

    # Get services
    cl1.useCluster()
    printHeader("\n\nStart get service")
    runcmd(f'kubectl get imports.clusterlink.net')
    runcmd(f'kubectl get accesspolicies.clusterlink.net')

    # Set policies
    printHeader(f"\n\nApplying allow-all policy file")
    cl1.policies.create(name="allow-all",namespace= namespace, action="allow", from_attribute=[{"workloadSelector": {}}],to_attribute=[{"workloadSelector": {}}])
    cl2.policies.create(name="allow-all",namespace= namespace, action="allow", from_attribute=[{"workloadSelector": {}}],to_attribute=[{"workloadSelector": {}}])
    cl3.policies.create(name="allow-all",namespace= namespace, action="allow", from_attribute=[{"workloadSelector": {}}],to_attribute=[{"workloadSelector": {}}])

# applyPolicy apply policy for the bookInfo demo
def applyPolicy(cl:Cluster, type):
    cl.useCluster()
    if type == "random":
        printHeader("Set random poilicy")
        cl.imports.update(reviewSvc,  namespace, srcK8sSvcPort,  ["peer2","peer3"], [reviewSvc,reviewSvc], [namespace,namespace],"random")
    if type == "round-robin":
        printHeader("Set round-robin poilicy")
        cl.imports.update(reviewSvc,  namespace, srcK8sSvcPort,  ["peer2","peer3"], [reviewSvc,reviewSvc], [namespace,namespace],"round-robin")
    elif type == "same":
        printHeader("Set same policy to all services")
        cl.imports.update(reviewSvc,  namespace, srcK8sSvcPort,  ["peer2","peer3"], [reviewSvc,reviewSvc], [namespace,namespace],"static")
    elif type == "diff":
        cl.policies.delete(name="allow-all",namespace= namespace)
        cl.policies.create(name="src1topeer2",namespace= namespace, action="allow", from_attribute=[{"workloadSelector": {"matchLabels": {"client.clusterlink.net/labels.app": srcSvc1}}}],to_attribute=[{"workloadSelector": {"matchLabels": {"export.clusterlink.net/name": reviewSvc, "peer.clusterlink.net/name": "peer2"}}}])
        cl.policies.create(name="src2topeer3",namespace= namespace, action="allow", from_attribute=[{"workloadSelector": {"matchLabels": {"client.clusterlink.net/labels.app": srcSvc2}}}],to_attribute=[{"workloadSelector": {"matchLabels": {"export.clusterlink.net/name": reviewSvc, "peer.clusterlink.net/name": "peer3"}}}])
    elif type == "show":
        runcmd(f'kubectl get imports.clusterlink.net')
    elif type == "clean":
        cl.policies.delete(name="src1topeer2",namespace= namespace)
        cl.policies.delete(name="src2topeer3",namespace= namespace)
        cl.policies.create(name="allow-all",namespace= namespace, action="allow", from_attribute=[{"workloadSelector": {}}],to_attribute=[{"workloadSelector": {}}])
        cl.imports.update(reviewSvc,  namespace, srcK8sSvcPort,  ["peer2","peer3"], [reviewSvc,reviewSvc], [namespace,namespace],"round-robin")

def apply_failover(cl:Cluster, type, testOutputFolder,container_reg="", ingress_type="", ingress_port=0):
    cl.useCluster()
    if type == "fail":
        printHeader(f"Failing {cl.name} dataplane")
        runcmd(f"kubectl delete instances.clusterlink.net cl-instance -n {CLUSTELINK_OPERATOR_NS}")
    elif type == "start":
        printHeader(f"Restoring {cl.name} dataplane")
        cl.deploy_peer(cl.name, testOutputFolder, container_reg, ingress_type, ingress_port)

