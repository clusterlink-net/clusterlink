#!/usr/bin/env bash
set -ex

TEST_DIR=$(mktemp -d)
CLADM=$GOPATH/src/github.com/clusterlink-org/clusterlink/bin/cl-adm

function clean_up {
  # delete containers
  docker rm -f peer1-controlplane || true
  docker rm -f peer1-dataplane || true
  docker rm -f peer1-gwctl || true

  # delete network
  docker network rm peer1 || true

  cd -
}

function test_docker {
  # create fabric with a single peer (peer1)
  $CLADM create fabric
  $CLADM create peer --name peer1

  # start containers
  ./peer1/docker-run.sh

  # connect all containers via network
  docker network create peer1
  docker network connect peer1 peer1-controlplane
  docker network connect peer1 peer1-dataplane
  docker network connect peer1 peer1-gwctl

  # install iperf3 and jq
  docker exec -it peer1-gwctl apk add iperf3 jq

  # start iperf3 server
  docker exec -itd peer1-gwctl iperf3 -s -p 1234

  # wait for API server to come up
  docker exec -it peer1-gwctl timeout 30 sh -c 'until gwctl get peer; do sleep 0.1; done > /dev/null 2>&1'

  # export iperf server
  docker exec -it peer1-gwctl gwctl create export --name foo --host peer1-gwctl --port 1234

  # import
  docker exec -it peer1-gwctl gwctl create peer --host peer1-dataplane --port 443 --name peer1
  docker exec -it peer1-gwctl gwctl create import --name foo --host bla --port 9999
  docker exec -it peer1-gwctl gwctl create binding --import foo --peer peer1

  # get imported service port
  PORT=$(docker exec -it peer1-gwctl /bin/bash -c "gwctl get import --name foo | jq '.Status.Listener.Port' | tr -d '\n'")

  # wait for imported service socket to come up
  docker exec -it peer1-gwctl timeout 30 sh -c 'until nc -z $0 $1; do sleep 0.1; done' peer1-dataplane $PORT
  # wait for iperf server to come up
  docker exec -it peer1-gwctl timeout 30 sh -c 'until nc -z $0 $1; do sleep 0.1; done' peer1-gwctl 1234

  # run iperf client
  docker exec -it peer1-gwctl iperf3 -c peer1-dataplane -p $PORT -t 1
}

cd $TEST_DIR
clean_up

trap clean_up INT TERM EXIT

cd $TEST_DIR
test_docker

echo OK
