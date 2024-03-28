#!/usr/bin/env bash
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

set -ex

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
TEST_DIR=$(mktemp -d)
CLI=$SCRIPT_DIR/../bin/clusterlink
DATAPLANE_TYPE="${1:-envoy}"

function clean_up {
  kind delete cluster --name peer1

  cd -
}

function clean_up_with_logs {
  # export logs
  kind export logs /tmp/clusterlink-k8s-e2e-$DATAPLANE_TYPE --name peer1

  clean_up
}

function test_k8s {
  # create fabric with a single peer (peer1)
  $CLI create fabric
  $CLI create peer-cert --name peer1 --dataplane-type $DATAPLANE_TYPE

  # create kind cluster
  kind create cluster --name peer1

  # load images to cluster
  kind load docker-image cl-controlplane --name peer1
  kind load docker-image cl-dataplane --name peer1
  kind load docker-image cl-go-dataplane --name peer1
  kind load docker-image gwctl --name peer1

  # configure kubectl
  kubectl config use-context kind-peer1

  # wait for service account to be created
  timeout 30 sh -c 'until kubectl -n default get serviceaccount default -o name; do sleep 0.1; done > /dev/null 2>&1'

  # create clusterlink objects
  kubectl create -f ./peer1/k8s.yaml

  # start iperf3 server
  kubectl run iperf-server --image=networkstatic/iperf3 -- iperf3 -s -p 1234

  # wait for gwctl pod to run
  kubectl wait --for=condition=ready pod/gwctl

  # install iperf3 and jq
  kubectl exec -i gwctl -- timeout 30 sh -c 'until apk add iperf3 jq; do sleep 0.1; done > /dev/null 2>&1'

  # expose iperf3 server
  kubectl expose pod iperf-server --name=foo --port=80 --target-port=1234

  # wait for API server to come up
  kubectl exec -i gwctl -- timeout 30 sh -c 'until gwctl get peer; do sleep 0.1; done > /dev/null 2>&1'

  # export iperf server
  kubectl exec -i gwctl -- gwctl create export --name foo --host foo --port 80

  # import
  kubectl exec -i gwctl -- gwctl create peer --host cl-dataplane --port 443 --name peer1
  kubectl exec -i gwctl -- gwctl create import --name bla --port 9999 --peer peer1
  kubectl cp $SCRIPT_DIR/../pkg/policyengine/examples/allowAll.json gwctl:/tmp/allowAll.json
  kubectl exec -i gwctl -- gwctl create policy --type access --policyFile /tmp/allowAll.json

  # get imported service port
  PORT=$(kubectl exec -i gwctl -- /bin/bash -c "gwctl get import --name foo | jq '.Status.Listener.Port' | tr -d '\n'")

  # wait for imported service socket to come up
  kubectl exec -i gwctl -- timeout 30 sh -c 'until nc -z $0 $1; do sleep 0.1; done' bla 9999
  # wait for iperf server to come up
  kubectl wait --for=condition=ready pod/iperf-server

  # run iperf client
  kubectl exec -i gwctl -- timeout 30 sh -c 'until iperf3 -c bla -p 9999 -t 1; do sleep 0.1; done > /dev/null 2>&1'
}

cd $TEST_DIR
clean_up

trap clean_up_with_logs INT TERM EXIT

cd $TEST_DIR
test_k8s

echo OK
