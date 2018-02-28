#!/bin/bash

# N/B: This script is intended to be executed in an SPC by .run_ci_tests.sh
#      Using it otherwise may result in unplesent side-effects.

# Additional packages needed that aren't already in the base image
DEBS="${DEBS:-btrfs-tools libdevmapper-dev}"

export GO_VERSION="${GO_VERSION:-stable}"

# Don't want to see this spam...unless it breaks
echo
echo "Updating/Installing: $DEBS"
TMPFILE=$(mktemp)
set +e
(
    set -x
    apt-get -qq update && apt-get install -qq $DEBS
) &> $TMPFILE
set -e
if [[ "$?" -gt "0" ]]
then
    cat $TMPFILE
    exit 1
fi

echo
echo "Setting up for go version \"$GO_VERSION\" (export GO_VERSION=something for that instead)"
if [[ ! -d "$HOME/.gimme" ]]
then
    # Ref: https://github.com/travis-ci/gimme/blob/master/README.md
    mkdir -p "$HOME/bin"
    curl -sL -o $HOME/bin/gimme https://raw.githubusercontent.com/travis-ci/gimme/master/gimme
    chmod +x $HOME/bin/gimme
    # Set env. vars here and for any future bash sessions
    X=$(echo 'export GOPATH="$HOME/go"' | tee -a $HOME/.bashrc) && eval "$X"
    X=$(echo 'export PATH="${PATH}:$HOME/bin:${GOPATH//://bin:}/bin"' | tee -a $HOME/.bashrc) && eval "$X"
    X="$($HOME/bin/gimme $GO_VERSION | tee -a $HOME/.bashrc)" && eval "$X"
    unset X
fi
source "$HOME/.bashrc"

echo
echo "Build Environment:"
go env
echo "PATH=$PATH"
echo "PWD=$PWD"

echo
echo "Building/Running tests"
make install.tools
make local-binary docs local-cross local-validate
make local-test-unit local-test-integration
