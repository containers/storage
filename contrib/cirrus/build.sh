#!/bin/bash

set -e

source $(dirname $0)/lib.sh

cd $GOSRC
make install.tools
echo "Build Binary"
make local-binary
echo "local-test-integration Overlay"
make STORAGE_DRIVER=overlay local-test-integration
echo "local-test-integration Fuse-overlay"
make STORAGE_DRIVER=overlay STORAGE_OPTION=overlay.mount_program=/usr/bin/fuse-overlayfs local-test-integration

case "$OS_REL_VER" in
    ubuntu-19)
	echo "local-test-integration Aufs"
	make STORAGE_DRIVER=aufs local-test-integration
	echo "local-test-unit"
	make local-test-unit
        ;;
    fedora-30)
	echo "local-test-unit"
	make local-test-unit
        ;;
esac
#make STORAGE_DRIVER=vfs local-test-integration
#make STORAGE_DRIVER=overlay FUSE_OVERLAYFS_DISABLE_OVL_WHITEOUT=1 STORAGE_OPTION=overlay.mount_program=/usr/bin/fuse-overlayfs local-test-integration
#make STORAGE_DRIVER=devicemapper STORAGE_OPTION=dm.directlvm_device=/dev/abc local-test-integration
