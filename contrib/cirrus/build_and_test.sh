#!/usr/bin/env bash

set -e

source $(dirname $0)/lib.sh

cd $GOSRC
make install.tools
showrun make local-binary
showrun make local-cross

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
    vfs)
        showrun make STORAGE_DRIVER=vfs local-test-integration
        ;;
    aufs)
        showrun make STORAGE_DRIVER=aufs local-test-integration
        ;;
    *)
        die "Unknown/Unsupported \$TEST_DRIVER=$TEST_DRIVER (see .cirrus.yml and $(basename $0))"
        ;;
esac
