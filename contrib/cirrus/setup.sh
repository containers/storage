#!/usr/bin/env bash

set -e

source $(dirname $0)/lib.sh

req_env_vars GOSRC OS_RELEASE_ID OS_RELEASE_VER SHORT_APTGET TEST_DRIVER

show_env_vars

cd $GOSRC
msg "Setting up $OS_RELEASE_ID $OS_RELEASE_VER"
case "$OS_RELEASE_ID" in
    fedora)
        $LONG_DNFY update  # install latest packages
        [[ -z "$RPMS_REQUIRED" ]] || \
            $SHORT_DNFY install $RPMS_REQUIRED
        [[ -z "$RPMS_CONFLICTING" ]] || \
            $SHORT_DNFY erase $RPMS_CONFLICTING
        # Only works on Fedora VM images
        bash "$SCRIPT_BASE/add_second_partition.sh"
        if [[ "$TEST_DRIVER" == "devicemapper" ]]; then
            $SHORT_DNFY install lvm2
            devicemapper_setup
        fi
        ;;
    ubuntu)
        $SHORT_APTGET update  # Fetch latest package metadata
        [[ -z "$DEBS_HOLD" ]] || \
            apt-mark hold $DEBS_HOLD
        $LONG_APTGET upgrade # install latest packages
        [[ -z "$DEBS_REQUIRED" ]] || \
            $SHORT_APTGET -q install $DEBS_REQUIRED
        [[ -z "$DEBS_CONFLICTING" ]] || \
            $SHORT_APTGET -q remove $DEBS_CONFLICTING
        ;;
    *)
        bad_os_id_ver
        ;;
esac

install_fuse_overlayfs_from_git
install_bats_from_git
