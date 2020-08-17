#!/usr/bin/env bash
set -e

cd "$(dirname "$(readlink -f "$BASH_SOURCE")")"

# Load the helpers.
. helpers.bash

function execute() {
	>&2 echo "++ $@"
	eval "$@"
}

# Tests to run. Defaults to all.
TESTS=${@:-.}

# N/B: Testing in parallel under automation is discourraged in this instance
#      (so `export JOBS=1`).  It has been observed to cause errors on Ubuntu,
#      and with so few tests here anyway, doesn't save much time (i.e. maybe a
#      few seconds at most)
export JOBS=${JOBS:-$(($(nproc --all) * 4))}

# Run the tests.
execute time bats --jobs "$JOBS" --tap $TESTS
