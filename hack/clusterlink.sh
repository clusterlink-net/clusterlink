#!/bin/sh
set -e

# Detrmine the OS.
OS=$(uname)
if [ "${OS}" = "Darwin" ] ; then
  CL_OS="os"
else
  CL_OS="linux"
fi


# Detrmine the OS architecture.
OS_ARCH=$(uname -m)
case "${OS_ARCH}" in
  x86_64|amd64)
    CL_ARCH=amd64
    ;;
  armv8*|aarch64*|arm64)
    ARCH=arm64
    ;;
  *)
    echo "This ${OS_ARCH} architecture isn't supported"
    exit 1
    ;;
esac

filename="clusterlink-${CL_OS}-${CL_ARCH}.tar.gz"
url="https://github.com/clusterlinl-net/clusterlink/releases/download/${VERSION}/${filename}"

# Set version to latest if not define and update the url.
if [ "${VERSION}" = "" ] ; then
  VERSION="latest"
  url="https://github.com/clusterlinl-net/clusterlink/releases/${VERSION}/download/${filename}"
fi

printf "\n Downloading %s from %s ...\n" "$filename" "$url"

if ! curl -o /dev/null -sIf "$url"; then
    printf "\n%s This version of clusterlonk wasn't found\n" "$url"
    exit 1
fi

# Open the tar file.
curl -fsLO ${url}
tar -xzf "${filename}"
rm "${filename}"

current_path=$(pwd)/clusterlink

install_path=${HOME}/.local/bin
mv $current_path/* $install_path
rm -rf $current_path

# Installation summary.
printf "\n"
printf "\e[1;34m.----.  .----. .-. . .-..-. .-..-.    .----.   .--.  .----.     .---.  .----. .-.   .-..----. .-.   .----..---. .----.\n"
printf "| {}  \\/  {}  \| |/ \| ||  \`| || |   /  {}  \\ / {} \\ | {}  \   /  ___}/  {}  \|  \`.'  || {}  }| |   | {_ {_   _}| {_  \n"
printf "|     /\\      /|  .'.  || |\  || \`--.\      //  /\\  \|     /   \\     }\      /| |\ /| || .--' | \`--.| {__  | |  | {__ \n"
printf "\`----'  \`----' \`-'   \`-'\`-' \`-'\`----' \`----' \`-'  \`-'\`----'     \`---'  \`----' \`-' \` \`-'\`-'    \`----'\`----' \`-'  \`----'\e[0m\n"
printf "\n\n"

printf "%s has been successfully downloaded.\n" "$filename"
printf "\n"
printf "ClusterLink CLI (gwctl and cl-adm) added to your environment path:\n"
printf "\n"
printf "\t\e[1;33m%s\n\e[0m" "$install_path"
printf "\n"
printf "If the ClusterLink command is not in your path, please add it using the following command:\n"
printf "\n"
printf "\t\e[1;33mexport PATH=\"\$PATH:%s\"\n\e[0m" "$install_path"
printf "\n"
printf "For more information on how to set up ClusterLink in your Kubernetes cluster, please see: \e[4mhttps://cluster-link.net/docs/getting-started\e[0m\n"
printf "\n"