#!/bin/bash
set -e

#-- docker/Python
# Docker and python can be quite involved to get right (e.g., require sudo,
# setting up permissions, etc.), so we ask the user to install them manully.
# The devcontainer will install using apt-get, so that's probably ok for now.
if [ -z "$(which docker)" ] || [ "$1" = "--force" ]; then
  echo "please install docker manually (https://docs.docker.com/engine/install/)"
fi

#-- python3
if [ -z "$(which python)" ] || [ "$1" = "--force" ]; then
	echo "please install python3 manually (https://docs.python.org/3/using/index.html)"
fi

# Go based executables are much easier, just need to ensure $GOPATH/bin is available in search path ;-)
GOBIN="$(go env GOPATH)/bin"
mkdir -p "$GOBIN"
if [ -z "$(echo $PATH | grep $GOBIN)" ]; then
  export PATH="$PATH:$GOBIN"
fi

#-- kubectl
VERSION=$(curl -L -s https://dl.k8s.io/release/stable.txt)
if [ -z "$(which kubectl)" ] || [ "$1" = "--force" ]; then
  echo installing kubectl "($VERSION)"
  if [[ $(uname -s) == "Linux" ]]; then
    curl -sLO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
  elif [[ $(uname -s) == "Darwin" ]]; then
    echo "Operating System: macOS"
    arch=$(uname -m)
    case $arch in
    "x86_64")
      echo "Architecture: amd64"
      curl -sLO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/darwin/amd64/kubectl"
      ;;
    "arm64")
      echo "Architecture: arm64"
      curl -sLO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/darwin/arm64/kubectl"
      ;;
    *)
      echo "Unknown Architecture: $arch"
      exit 1
      ;;
    esac
  else
    echo "Unknown OS"
    exit 1
  fi
  chmod 755 kubectl
  mv kubectl "$(go env GOPATH)/bin"
fi

#-- kind
VERSION=v0.22.0
if [ -z "$(which kind)" ] || [ "$1" = "--force" ]; then
  echo installing kind "($VERSION)"
  go install sigs.k8s.io/kind@$VERSION
fi

#-- golangci-lint
VERSION=v1.54.2
if [ -z "$(which golangci-lint)" ] || [ "$1" = "--force" ]; then
  echo installing golangci-lint "($VERSION)"
  go install "github.com/golangci/golangci-lint/cmd/golangci-lint@$VERSION"
fi

#-- goimports
VERSION=latest
if [ -z "$(which tparse)" ] || [ "$1" = "--force" ]; then
  echo installing goimports "($VERSION)"
  go install "golang.org/x/tools/cmd/goimports@$VERSION"
fi

#-- gofumpt
VERSION="v0.6.0"
if [ -z "$(which gofumpt)" ] || [ "$1" = "--force" ]; then
  echo installing gofumpt "($VERSION)"
  go install "mvdan.cc/gofumpt@$VERSION"
fi

#-- tparse
VERSION=latest
if [ -z "$(which tparse)" ] || [ "$1" = "--force" ]; then
  echo installing tparse "($VERSION)"
  go install "github.com/mfridman/tparse@$VERSION"
fi

#--  setup-envtest
VERSION=latest
if [ -z "$(which setup-envtest)" ] || [ "$1" = "--force" ]; then
  echo installing setup-envtest "($VERSION)"
  go install "sigs.k8s.io/controller-runtime/tools/setup-envtest@$VERSION"
fi

#-- muffet (link checker)
VERSION=v2.9.3
if [ -z "$(which nuffet)" ] || [ "$1" = "--force" ]; then
  echo installing "muffet ($VERSION)"
  go install "github.com/raviqqe/muffet/v2@$VERSION"
fi
