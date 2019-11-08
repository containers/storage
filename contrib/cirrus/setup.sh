#!/bin/bash

set -e

source $(dirname $0)/lib.sh

req_env_var GOSRC OS_RELEASE_ID OS_RELEASE_VER SHORT_APTGET

install_ooe

show_env_vars

cd $GOSRC

echo "Setting up $OS_RELEASE_ID $OS_RELEASE_VER"
case "$OS_RELEASE_ID" in
    fedora)
        $LONG_DNFY update  # install latest packages
        [[ -z "$RPMS_REQUIRED" ]] || \
            $SHORT_DNFY install $RPMS_REQUIRED
        [[ -z "$RPMS_CONFLICTING" ]] || \
            $SHORT_DNFY erase $RPMS_CONFLICTING
        ;;
    ubuntu)
        $SHORT_APTGET update  # Fetch latest package metadata
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

echo "Installing common tooling"
#make install.tools
