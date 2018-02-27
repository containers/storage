#!/bin/bash

# This script can be run manually (assuming docker-daemon access) as-is.
# Or, export SPCCMD=bash to get your hands less clean.

set -ex

SPCCMD="${SPCCMD:-./.spc_ci_test.sh}"
FQIN="docker.io/cevich/travis_ubuntu:latest"
SPC_ARGS="--interactive --rm --privileged --ipc=host --pid=host --net=host"
VOL_ARGS="-v ${REPO_VOL:-$PWD}:${REPO_VOL:-$PWD}:z
          -v /run:/run -v /etc/localtime:/etc/localtime
          -v /var/log:/var/log -v /sys/fs/cgroup:/sys/fs/cgroup
          -v /var/run/docker.sock:/var/run/docker.sock
          --workdir ${TRAVIS_BUILD_DIR:-$PWD}"
ENV_ARGS="-e HOME=${REPO_VOL:-$PWD} -e GO_VERSION=${GO_VERSION:-stable}"
sudo docker pull $FQIN
sudo docker run -t $SPC_ARGS $VOL_ARGS $ENV_ARGS $TRAVIS_ENV $FQIN $SPCCMD
