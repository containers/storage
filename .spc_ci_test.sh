#!/bin/bash

# N/B: This script is intended to be executed in an SPC by .run_ci_tests.sh
#      Using it otherwise may result in unplesent side-effects

# Additional packages needed that aren't already in the base image
DEBS="btrfs-tools libdevmapper-dev"

# Don't want to see this spam...unless it breaks
TMPFILE=$(mktemp)
set +e
echo "Updating/Installing: $DEBS"
(
    set -x
    apt-get -qq update && apt-get install -qq $DEBS
) &> $TMPFILE
if [[ "$?" -gt "0" ]]
then
    cat $TMPFILE
    exit 1
fi

set -e

export HOME="/repo_copy"
echo "Copying everything to $HOME so volume isn't cluttered up"
mkdir -p "$HOME"
rsync --recursive --links --delay-updates --whole-file \
      --safe-links --perms --times --checksum ./ "${HOME}/"
cd "$HOME"

# FIXME: All a total guess, someone intelligent should fix this
export GOPATH="$HOME/"
export PATH="${PATH}:${GOPATH//://bin:}/bin"

echo "Setting up for go $GO_VERSION - export GO_VERSION=whatever for something different"
# Ref: https://github.com/travis-ci/gimme/blob/master/README.md
mkdir -p "$HOME/bin"
curl -sL -o $HOME/bin/gimme https://raw.githubusercontent.com/travis-ci/gimme/master/gimme
chmod +x $HOME/bin/gimme
GIMME_OUTPUT="$($HOME/bin/gimme $GO_VERSION | tee -a $HOME/.bashrc)" && eval "$GIMME_OUTPUT"

echo "Build Environment:"
env

echo "Building/Running tests"

make install.tools
make local-binary docs local-cross local-validate
make local-test-unit local-test-integration
