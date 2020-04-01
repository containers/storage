#!/bin/bash
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

export JOBS=${JOBS:-$(($(nproc --all) * 4))}

# Run the tests.
execute time bats --jobs "$JOBS" --tap $TESTS
