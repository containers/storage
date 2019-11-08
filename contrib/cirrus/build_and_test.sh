#!/bin/bash

set -e

source $(dirname $0)/lib.sh

cd $GOSRC
make install.tools
showrun make local-binary
showrun make local-cross

showrun make local-test-unit

# TODO: Some integration tests fail on Fedora
if [[ "$OS_RELEASE_ID" != "fedora" ]]; then
    showrun make STORAGE_DRIVER=overlay local-test-integration
fi

showrun make STORAGE_DRIVER=overlay STORAGE_OPTION=overlay.mount_program=/usr/bin/fuse-overlayfs local-test-integration

showrun make STORAGE_DRIVER=overlay FUSE_OVERLAYFS_DISABLE_OVL_WHITEOUT=1 STORAGE_OPTION=overlay.mount_program=/usr/bin/fuse-overlayfs local-test-integration

showrun make STORAGE_DRIVER=vfs local-test-integration

if [[ "$OS_RELEASE_ID" == "ubuntu" ]]; then
    showrun make STORAGE_DRIVER=aufs local-test-integration
fi

# TODO: Requires partitioning of $(cat /root/second_partition_ready) device after running
# https://github.com/containers/libpod/blob/v1.6.2/contrib/cirrus/add_second_partition.sh
#
#showrun make STORAGE_DRIVER=devicemapper STORAGE_OPTION=dm.directlvm_device=/dev/abc local-test-integration
