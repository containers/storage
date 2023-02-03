#!/usr/bin/env bash

set -e

source $(dirname $0)/lib.sh

req_env_vars GOSRC OS_RELEASE_ID OS_RELEASE_VER SHORT_APTGET TEST_DRIVER

show_env_vars

cd $GOSRC
msg "Setting up $OS_RELEASE_ID $OS_RELEASE_VER"
case "$OS_RELEASE_ID" in
    fedora)
        [[ -z "$RPMS_CONFLICTING" ]] || \
            $SHORT_DNFY erase $RPMS_CONFLICTING
        $SHORT_DNFY install zstd
        ;;
    ubuntu)
        [[ -z "$DEBS_CONFLICTING" ]] || \
            $SHORT_APTGET -q remove $DEBS_CONFLICTING
        $SHORT_APTGET -q update
        $SHORT_APTGET -q install zstd zfsutils-linux
        ;;
    *)
        bad_os_id_ver
        ;;
esac

install_fuse_overlayfs_from_git
install_bats_from_git
