#!/bin/bash
set -e

GO_VERSION=1.7.4

source /etc/os-release

case "${ID_LIKE:-${ID:-unknown}}" in
  debian)
    export DEBIAN_FRONTEND=noninteractive
    apt-get -q update
    apt-get -q -y install linux-headers-`uname -r`
    echo deb http://httpredir.debian.org/debian testing main    >  /etc/apt/sources.list
    echo deb http://httpredir.debian.org/debian testing contrib >> /etc/apt/sources.list
    apt-get -q update
    apt-get -q -y install systemd curl
    apt-get -q -y install apt make git btrfs-progs libdevmapper-dev
    apt-get -q -y install zfs-dkms zfsutils-linux
    curl -sSL https://storage.googleapis.com/golang/go${GO_VERSION}.linux-amd64.tar.gz | tar -xvz -C /usr/local
    modprobe aufs
    modprobe zfs
    ;;
  fedora)
    dnf -y clean all
    dnf -y install make git gcc btrfs-progs-devel device-mapper-devel
    curl -sSL https://storage.googleapis.com/golang/go${GO_VERSION}.linux-amd64.tar.gz | tar -xvz -C /usr/local
    ;;
  unknown)
    echo Unknown box OS, unsure of how to install required packages.
    exit 1
    ;;
esac
mkdir -p /go/src/github.com/containers
rm -f /go/src/github.com/containers/storage
ln -s /vagrant /go/src/github.com/containers/storage
export GOPATH=/go:/go/src/github.com/containers/storage/vendor
export PATH=/usr/local/go/bin:/go/bin:${PATH}
go get github.com/golang/lint
