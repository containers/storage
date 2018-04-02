

# N/B: This script is only to be used by run_ci_tests.sh for setting up podman to run
# in in the travis VM (host).  All other usage may result in severe halitosis.

set -e

if [[ "$CONTAINER" != "podman" ]] || [[ "$TRAVIS" != "true" ]]
then
    exit 42
fi

# FIXME: Add steps needed to install / setup podman on Ubuntu Trusty
