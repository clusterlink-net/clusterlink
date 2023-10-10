#!/usr/bin/env bash
set -ex

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
TEST_DIR=$(mktemp -d)
CLADM=$SCRIPT_DIR/../bin/cl-adm

function clean_up {
  kind delete cluster --name peer1

  cd -
}

function test_k8s {
  # create fabric with a single peer (peer1)
  $CLADM create fabric
  $CLADM create peer --name peer1

  # create kind cluster
  kind create cluster --name peer1

  # load images to cluster
  kind load docker-image cl-controlplane cl-dataplane gwctl --name peer1

  # configure kubectl
  kubectl config use-context kind-peer1

  # create clusterlink objects
  kubectl create -f ./peer1/k8s.yaml

  # wait for gwctl pod to run
  kubectl wait --for=condition=ready pod/gwctl

  # install iperf3 and jq
  kubectl exec -i gwctl -- apk add iperf3 jq

  # start iperf3 server
  kubectl exec -i gwctl -- iperf3 -s -D -p 1234

  # expose iperf3 server
  kubectl expose pod gwctl --name=foo --port=80 --target-port=1234

  # wait for API server to come up
  kubectl exec -i gwctl -- timeout 30 sh -c 'until gwctl get peer; do sleep 0.1; done > /dev/null 2>&1'

  # export iperf server
  kubectl exec -i gwctl -- gwctl create export --name foo --host foo --port 80

  # import
  kubectl exec -i gwctl -- gwctl create peer --host cl-dataplane --port 443 --name peer1
  kubectl exec -i gwctl -- gwctl create import --name foo --host bla --port 9999
  kubectl exec -i gwctl -- gwctl create binding --import foo --peer peer1
  kubectl cp $SCRIPT_DIR/../pkg/policyengine/policytypes/examples/allowAll.json gwctl:/tmp/allowAll.json
  kubectl exec -i gwctl -- gwctl create policy --type access --policyFile /tmp/allowAll.json

  # get imported service port
  PORT=$(kubectl exec -i gwctl -- /bin/bash -c "gwctl get import --name foo | jq '.Status.Listener.Port' | tr -d '\n'")

  # expose imported service (TODO: remove this when controlplane automatically creates a service)
  kubectl expose deployment cl-dataplane --name=bla --port=9999 --target-port=$PORT

  # wait for imported service socket to come up
  kubectl exec -i gwctl -- timeout 30 sh -c 'until nc -z $0 $1; do sleep 0.1; done' bla 9999
  # wait for iperf server to come up
  kubectl exec -i gwctl -- timeout 30 sh -c 'until nc -z $0 $1; do sleep 0.1; done' gwctl 1234

  # run iperf client
  kubectl exec -i gwctl -- iperf3 -c bla -p 9999 -t 1
}

cd $TEST_DIR
clean_up

trap clean_up INT TERM EXIT

cd $TEST_DIR
test_k8s

echo OK
