#!/bin/bash

set -e

# This script can be run manually - assuming password-less sudo access
# and a docker-daemon running.  To play in the SPC, just
# `export SPCCMD=bash` beforehand.

SPCCMD="${SPCCMD:-./hack/spc_ci_test.sh}"
DISTRO="${DISTRO:-fedora}"
FQIN="docker.io/cevich/travis_${DISTRO}:latest"

# This can take a while, start it going as early as possible
sudo docker info
sudo docker pull $FQIN &

REPO_DIR=$(realpath "$(dirname $0)/../")  # assume parent directory of 'hack'
REPO_HOST=${REPO_HOST:-github.com}  # required for go building/testing
# Otherwise undiscoverable, default to parent dir of $REPO_DIR outside of travis
REPO_OWNER=$(echo "$TRAVIS_REPO_SLUG" | cut -d / -f 1)  # may be empty
REPO_OWNER=${REPO_OWNER:-$(basename $(realpath $(dirname $REPO_DIR)))}
REPO_NAME=$(basename $(git rev-parse --show-toplevel))
# In Travis $PWD == $TRAVIS_BUILD_DIR == $HOME/$REPO_OWNER/$REPO_NAME
TRAVIS_BUILD_DIR="/root/$REPO_OWNER/$REPO_NAME"
WORKDIR="/root/go/src/$REPO_HOST/$REPO_OWNER/$REPO_NAME"

# Volume-mounting the repo into the SPC makes a giant mess of permissions
# on the host.  This really sucks for developers, so make a copy for use
# in the SPC separate from the host, throw it away when this script exits.
echo
echo "Making temporary copy of $REPO_DIR that"
echo "will appear in SPC as $WORKDIR"
TMP_SPC_REPO_COPY=$(mktemp -p '' -d ${REPO_NAME}_XXXXXX)
trap "sudo rm -rf $TMP_SPC_REPO_COPY" EXIT
/usr/bin/rsync --recursive --links --delete-after --quiet \
               --delay-updates --whole-file --safe-links \
               --perms --times "${REPO_DIR}/" "${TMP_SPC_REPO_COPY}/" &

SPC_ARGS="--interactive --rm --privileged --ipc=host --pid=host --net=host"

VOL_ARGS="-v $TMP_SPC_REPO_COPY:$WORKDIR
          -v /run:/run -v /etc/localtime:/etc/localtime
          -v /var/log:/var/log -v /sys/fs/cgroup:/sys/fs/cgroup
          -v /var/run/docker.sock:/var/run/docker.sock
          -v /dev:/dev
          --tmpfs /tmp:rw,nosuid,nodev,exec,relatime,mode=1777,size=50%
          --workdir $WORKDIR"

ENV_ARGS="-e GO_VERSION=${GO_VERSION:-stable} -e HOME=/root -e SHELL=/bin/bash
          -e SPC=true -e DISTRO=$DISTRO -e REPO=$REPO_HOST/$REPO_OWNER/$REPO_NAME"

echo
echo "Preparing to run \$SPCMD=$SPCCMD in a \$DISTRO=$DISTRO SPC."
echo "Override either for a different experience."
wait  # for backgrounded processes to finish
echo
set -x
sudo docker run -t $SPC_ARGS $VOL_ARGS $ENV_ARGS $TRAVIS_ENV $FQIN $SPCCMD
