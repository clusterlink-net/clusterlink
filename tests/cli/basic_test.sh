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
CLI=$SCRIPT_DIR/../../bin/clusterlink

function clean_up {
  kind delete cluster --name peer1

  cd -
}

function clean_up_with_logs {
  # export logs
  kind export logs /tmp/clusterlink-cli --name peer1

  clean_up
}

function test_k8s {
  # create fabric with a single peer (peer1)
  $CLI create fabric
  $CLI create peer-cert --name peer1

  # create kind cluster
  kind create cluster --name peer1

  # load images to cluster
  kind load docker-image cl-controlplane --name peer1
  kind load docker-image cl-dataplane --name peer1
  kind load docker-image cl-operator --name peer1

  # configure kubectl
  kubectl config use-context kind-peer1

  # wait for service account to be created
  timeout 30 sh -c 'until kubectl -n default get serviceaccount default -o name; do sleep 0.1; done > /dev/null 2>&1'

  # create clusterlink objects
  $CLI deploy peer --name peer1 --container-registry=docker.io/library --ingress=NodePort --ingress-port=30443

  # wait for cl-controlplane and cl-dataplane to be created
    if ! timeout 30 sh -c 'until kubectl rollout status deployment cl-controlplane -n clusterlink-system; do sleep 0.1; done > /dev/null 2>&1'; then
        echo "Error: Timeout occurred while waiting for cl-controlplane deployment"
        exit 1
    fi

    if ! timeout 30 sh -c 'until kubectl rollout status deployment cl-dataplane -n clusterlink-system; do sleep 0.1; done > /dev/null 2>&1'; then
        echo "Error: Timeout occurred while waiting for cl-dataplane deployment"
        exit 1
    fi

  # Selete clusterlink objects
  $CLI delete peer --name peer1

}

cd $TEST_DIR
clean_up

trap clean_up_with_logs INT TERM EXIT

cd $TEST_DIR
test_k8s

echo OK