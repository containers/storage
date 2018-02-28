#!/bin/bash

set -e

# This script can be run manually - assuming password-less sudo access
# and a docker-daemon running.

FQIN="${FQIN:-docker.io/cevich/travis_ubuntu:latest}"
SPCCMD="${SPCCMD:-./.spc_ci_test.sh}"

REPO_HOST='github.com'
REPO_OWNER='containers'

REPO_NAME=$(basename $(git rev-parse --show-toplevel))
REPO_VOL_DEST="/root/go/src/$REPO_HOST/$REPO_OWNER/$REPO_NAME"

# Volume-mounting the repo into the SPC makes a giant mess of permissions
# on the host.  This really sucks for developers, so make a copy for use
# in the SPC separate from the host, throw it away when this script exits.
echo
echo "Making temporary copy of $PWD that will appear in SPC as $REPO_VOL_DEST"
TMP_SPC_REPO_COPY=$(mktemp -p '' -d ${REPO_NAME}_XXXXXX)
trap "sudo rm -rf $TMP_SPC_REPO_COPY" EXIT
/usr/bin/rsync --recursive --links --delete-after --quiet \
               --delay-updates --whole-file --safe-links \
               --perms --times --checksum "${PWD}/" "${TMP_SPC_REPO_COPY}/" &

SPC_ARGS="--interactive --rm --privileged --ipc=host --pid=host --net=host"

# In Travis $PWD == $TRAVIS_BUILD_DIR == a subdir of $HOME == /home/travis/
VOL_ARGS="-v $TMP_SPC_REPO_COPY:$REPO_VOL_DEST
          -v /run:/run -v /etc/localtime:/etc/localtime
          -v /var/log:/var/log -v /sys/fs/cgroup:/sys/fs/cgroup
          -v /var/run/docker.sock:/var/run/docker.sock
          --workdir $REPO_VOL_DEST"

ENV_ARGS="-e GO_VERSION=${GO_VERSION:-stable} -e HOME=/root -e SHELL=/bin/bash"

echo
echo "Preparing to run $SPCCMD in a $FQIN SPC."
echo "Override either, for a different experience (e.g. SPCCMD=bash)"
set -x
sudo docker pull $FQIN
wait  # for rsync if not finished
sudo docker run -t $SPC_ARGS $VOL_ARGS $ENV_ARGS $TRAVIS_ENV $FQIN $SPCCMD
