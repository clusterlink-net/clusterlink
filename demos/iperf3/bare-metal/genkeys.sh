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

#
# Generate a set of TLS credentials that can be used to run development mode.
#
# Based on script by Ash Wilson (@smashwilson)
# https://github.com/cloudpipe/cloudpipe/pull/45/files#diff-15
#
# usage: sh ./genkeys.sh NAME HOSTNAME IP

set -o errexit

USAGE="usage: sh ./genkeys.sh NAME HOSTNAME IP"
ROOT="$(pwd)"
PASSFILE="${ROOT}/dev.password"
PASSOPT="file:${ROOT}/dev.password"
CAFILE="${ROOT}/ca.crt"
CAKEY="${ROOT}/ca.key"

# Randomly create a password file, if you haven't supplied one already.
# For development mode, we'll just use the same (random) password for everything.
if [ ! -f "${PASSFILE}" ]; then
  echo ">> creating a random password in ${PASSFILE}."
  touch ${PASSFILE}
  chmod 600 ${PASSFILE}
  # "If the same pathname argument is supplied to -passin and -passout arguments then the first
  # line will be used for the input password and the next line for the output password."
  cat /dev/random | head -c 128 | base64 | sed -n '{p;p;}' >> ${PASSFILE}
  echo "<< random password created"
fi

# Generate the certificate authority that we'll use as the root for all the things.
if [ ! -f "${CAFILE}" ]; then
  echo ">> generating a certificate authority"
  openssl genrsa -des3 \
    -passout ${PASSOPT} \
    -out ${CAKEY} 2048
  openssl req -new -x509 -days 365 \
    -batch \
    -passin ${PASSOPT} \
    -key ${CAKEY} \
    -passout ${PASSOPT} \
    -out ${CAFILE}
  echo "<< certificate authority generated."
fi

# Generate a named keypair
keypair() {
  local NAME=$1
  local HOSTNAME=$2
  local IP=$3

  local SERIALOPT=""
  if [ ! -f "${ROOT}/ca.srl" ]; then
    echo ">> creating serial"
    SERIALOPT="-CAcreateserial"
  else
    SERIALOPT="-CAserial ${ROOT}/ca.srl"
  fi

  echo ">> generating a keypair for: ${NAME}"

  echo ".. key"
  openssl genrsa -des3 \
    -passout ${PASSOPT} \
    -out ${ROOT}/${NAME}-key.pem 2048

  cp ${ROOT}/openssl.cnf ${ROOT}/openssl-${NAME}.cnf
  echo "IP = ${IP}" >> ${ROOT}/openssl-${NAME}.cnf

  echo ".. request"
  openssl req -subj "/CN=${HOSTNAME}" -new \
    -batch \
    -passin ${PASSOPT} \
    -key ${ROOT}/${NAME}-key.pem \
    -passout ${PASSOPT} \
    -out ${ROOT}/${NAME}-req.csr \
    -config ${ROOT}/openssl-${NAME}.cnf

  echo ".. certificate"
  openssl x509 -req -days 365 \
    -passin ${PASSOPT} \
    -in ${ROOT}/${NAME}-req.csr \
    -CA ${CAFILE} \
    -CAkey ${CAKEY} \
    ${SERIALOPT} \
    -extensions v3_req \
    -extfile ${ROOT}/openssl-${NAME}.cnf \
    -out ${ROOT}/${NAME}-cert.pem \

  echo ".. removing key password"
  openssl rsa \
    -passin ${PASSOPT} \
    -in ${ROOT}/${NAME}-key.pem \
    -out ${ROOT}/${NAME}-key.pem

  echo "<< ${NAME} keypair generated."
}

# call with arguments name, hostname, and ip address
if [ -z "$1" ]; then
  echo "${USAGE}"
  exit 1
fi
if [ -z "$2" ]; then
  echo "${USAGE}"
  exit 1
fi
if [ -z "$3" ]; then
  echo "${USAGE}"
  exit 1
fi

keypair "$1" "$2" "$3"
