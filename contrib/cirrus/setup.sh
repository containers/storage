#!/bin/bash

set -e

source $(dirname $0)/lib.sh

show_env_vars

cd $GOSRC

case "$OS_REL_VER" in
    fedora-30)
        echo "Setting up $OS_RELEASE_ID $OS_RELEASE_VER"  # STUB: Add VM setup instructions here
        ;;
    fedora-29)
        echo "Setting up $OS_RELEASE_ID $OS_RELEASE_VER"  # STUB: Add VM setup instructions here
        ;;
    ubuntu-19)
        echo "Setting up $OS_RELEASE_ID $OS_RELEASE_VER"  # STUB: Add VM setup instructions here
        ;;
    ubuntu-18)
        echo "Setting up $OS_RELEASE_ID $OS_RELEASE_VER"  # STUB: Add VM setup instructions here
        ;;
    *)
        bad_os_id_ver
        ;;
esac

echo "Installing common tooling"
#make install.tools
