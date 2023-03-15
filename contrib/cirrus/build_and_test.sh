#!/usr/bin/env bash

set -e

source $(dirname $0)/lib.sh

cd $GOSRC
make install.tools
showrun make local-binary
showrun make local-cross

case $TEST_DRIVER in
    overlay)
        showrun make STORAGE_DRIVER=overlay local-test-integration local-test-unit
        ;;
    overlay-transient)
        showrun make STORAGE_DRIVER=overlay STORAGE_TRANSIENT=1 local-test-integration local-test-unit
        ;;
    fuse-overlay)
        showrun make STORAGE_DRIVER=overlay STORAGE_OPTION=overlay.mount_program=/usr/bin/fuse-overlayfs local-test-integration local-test-unit
        ;;
    fuse-overlay-whiteout)
        showrun make STORAGE_DRIVER=overlay FUSE_OVERLAYFS_DISABLE_OVL_WHITEOUT=1 STORAGE_OPTION=overlay.mount_program=/usr/bin/fuse-overlayfs local-test-integration local-test-unit
        ;;
    vfs)
        showrun make STORAGE_DRIVER=vfs local-test-integration local-test-unit
        ;;
    aufs)
        showrun make STORAGE_DRIVER=aufs local-test-integration local-test-unit
        ;;
    btrfs)
        # Fedora: Needs btrfs-progs, btrfs-progs-devel
        # Debian: Needs btrfs-progs, libbtrfs-dev
        if [[ "$(./hack/btrfs_tag.sh)" =~ exclude_graphdriver_btrfs ]]; then
            die "Built without btrfs, so we can't test it"
        fi
        if ! check_filesystem_supported $TEST_DRIVER ; then
            die "This CI VM does not support $TEST_DRIVER in its kernel"
        fi
        if test -z "$(which mkfs.btrfs 2> /dev/null)" ; then
            die "This CI VM does not have mkfs.btrfs installed"
        fi
        tmpdir=$(mktemp -d)
        if [ -z "$tmpdir" ]; then
            die "Error creating temporary directory"
        fi
        trap "umount -l $tmpdir; rm -f $GOSRC/$TEST_DRIVER.img" EXIT
        truncate -s 0 $GOSRC/$TEST_DRIVER.img
        fallocate -l 1G $GOSRC/$TEST_DRIVER.img
        mkfs.btrfs $GOSRC/$TEST_DRIVER.img
        mount -o loop $GOSRC/$TEST_DRIVER.img $tmpdir
        TMPDIR="$tmpdir" showrun make STORAGE_DRIVER=$TEST_DRIVER local-test-integration local-test-unit
        ;;
    zfs)
        # Debian: Needs zfsutils
        if ! check_filesystem_supported $TEST_DRIVER ; then
            die "This CI VM does not support $TEST_DRIVER in its kernel"
        fi
        if test -z "$(which zpool 2> /dev/null)" ; then
            die "This CI VM does not have zpool installed"
        fi
        if test -z "$(which zfs 2> /dev/null)" ; then
            die "This CI VM does not have zfs installed"
        fi
        tmpfile=$(mktemp -p $GOSRC)
        truncate -s 0 $tmpfile
        fallocate -l 1G $tmpfile
        zpool=$(basename $tmpfile)
        zpool create $zpool $tmpfile
        trap "zfs destroy -Rf $zpool/tmp; zpool destroy -f $zpool; rm -f $tmpfile" EXIT
        zfs create $zpool/tmp
        TMPDIR="/$zpool/tmp" showrun make STORAGE_DRIVER=$TEST_DRIVER local-test-integration local-test-unit
        ;;
    *)
        die "Unknown/Unsupported \$TEST_DRIVER=$TEST_DRIVER (see .cirrus.yml and $(basename $0))"
        ;;
esac
