#!/bin/bash

set -e

source $(dirname $0)/lib.sh

cd $GOSRC
make install.tools
showrun make local-binary
showrun make local-cross

# On Ubuntu w/ Bats <= 1.2.0 using more than one job throws errors like:
# cat: /tmp/bats-run-23134/parallel_output/1/stdout: No such file or directory
if [[ "$OS_RELEASE_ID" == "ubuntu" ]]; then
    # See tests/test_runner.bash
    export JOBS=1  # Only ~50 tests @ 1-second each, not so bad to do one at a time.
fi

case $TEST_DRIVER in
    overlay)
        showrun make STORAGE_DRIVER=overlay local-test-integration
        ;;
    fuse-overlay)
        showrun make STORAGE_DRIVER=overlay STORAGE_OPTION=overlay.mount_program=/usr/bin/fuse-overlayfs local-test-integration
        ;;
    fuse-overlay-whiteout)
        showrun make STORAGE_DRIVER=overlay FUSE_OVERLAYFS_DISABLE_OVL_WHITEOUT=1 STORAGE_OPTION=overlay.mount_program=/usr/bin/fuse-overlayfs local-test-integration
        ;;
    devicemapper)
        # Setup by devicemapper_setup in lib.sh
        DM_DEVICE=$(< $DM_REF_FILEPATH)
        echo "WARNING: Performing destructive testing against $DM_DEVICE"
        showrun make STORAGE_DRIVER=devicemapper STORAGE_OPTION=dm.directlvm_device=$DM_DEVICE local-test-integration
        ;;
    vfs)
        showrun make STORAGE_DRIVER=vfs local-test-integration
        ;;
    aufs)
        showrun make STORAGE_DRIVER=aufs local-test-integration
        ;;
    *)
        die 11 "Unknown/Unsupported \$TEST_DRIVER=$TEST_DRIVER (see .cirrus.yml and $(basename $0))"
        ;;
esac
