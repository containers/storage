

# Library of common, shared utility functions.  This file is intended
# to be sourced by other scripts, not called directly.

# Global details persist here
source /etc/environment  # not always loaded under all circumstances

# Due to differences across platforms and runtime execution environments,
# handling of the (otherwise) default shell setup is non-uniform.  Rather
# than attempt to workaround differences, simply force-load/set required
# items every time this library is utilized.
source /etc/profile
source /etc/environment
USER="$(whoami)"
export HOME="$(getent passwd $USER | cut -d : -f 6)"
[[ -n "$UID" ]] || UID=$(getent passwd $USER | cut -d : -f 3)
GID=$(getent passwd $USER | cut -d : -f 4)

# During VM Image build, the 'containers/automation' installation
# was performed.  The final step of installation sets the library
# location $AUTOMATION_LIB_PATH in /etc/environment or in the
# default shell profile depending on distribution.
if [[ -n "$AUTOMATION_LIB_PATH" ]]; then
    source $AUTOMATION_LIB_PATH/common_lib.sh
else
    (
    echo "WARNING: It does not appear that containers/automation was installed."
    echo "         Functionality of most of this library will be negatively impacted"
    echo "         This ${BASH_SOURCE[0]} was loaded by ${BASH_SOURCE[1]}"
    ) > /dev/stderr
fi

# Essential default paths, many are overridden when executing under Cirrus-CI
# others are duplicated here, to assist in debugging.
export GOPATH="${GOPATH:-/var/tmp/go}"
if type -P go &> /dev/null
then
    # required for go 1.12+
    export GOCACHE="${GOCACHE:-$HOME/.cache/go-build}"
    eval "$(go env)"
    # required by make and other tools
    export $(go env | cut -d '=' -f 1)

    # Ensure compiled tooling is reachable
    export PATH="$PATH:$GOPATH/bin"
fi
CIRRUS_WORKING_DIR="${CIRRUS_WORKING_DIR:-$GOPATH/src/github.com/containers/storage}"
export GOSRC="${GOSRC:-$CIRRUS_WORKING_DIR}"
export PATH="$HOME/bin:$GOPATH/bin:/usr/local/bin:$PATH"
SCRIPT_BASE=${GOSRC}/contrib/cirrus

cd $GOSRC
if type -P git &> /dev/null
then
    CIRRUS_CHANGE_IN_REPO=${CIRRUS_CHANGE_IN_REPO:-$(git show-ref --hash=8 HEAD || date +%s)}
else # pick something unique and obviously not from Cirrus
    CIRRUS_CHANGE_IN_REPO=${CIRRUS_CHANGE_IN_REPO:-no_git_$(date +%s)}
fi

export CI="${CI:-false}"
CIRRUS_CI="${CIRRUS_CI:-false}"
CONTINUOUS_INTEGRATION="${CONTINUOUS_INTEGRATION:-false}"
CIRRUS_REPO_NAME=${CIRRUS_REPO_NAME:-storage}
CIRRUS_BASE_SHA=${CIRRUS_BASE_SHA:-unknown$(date +%s)}  # difficult to reliably discover
CIRRUS_BUILD_ID=${CIRRUS_BUILD_ID:-$RANDOM$(date +%s)}  # must be short and unique

# Unsafe env. vars for display
SECRET_ENV_RE='(IRCID)|(ACCOUNT)|(^GC[EP]..+)|(SSH)'

# GCE image-name compatible string representation of distribution name
OS_RELEASE_ID="$(source /etc/os-release; echo $ID)"
# GCE image-name compatible string representation of distribution _major_ version
OS_RELEASE_VER="$(source /etc/os-release; echo $VERSION_ID | tr -d '.')"
# Combined to ease soe usage
OS_REL_VER="${OS_RELEASE_ID}-${OS_RELEASE_VER}"

# Working with dnf + timeout/retry
export SHORT_DNFY='lilto dnf -y'
export LONG_DNFY='bigto dnf -y'
# Working with apt under Debian/Ubuntu automation is a PITA, make it easy
# Avoid some ways of getting stuck waiting for user input
export DEBIAN_FRONTEND=noninteractive
# Short-cut for frequently used base command
export SUDOAPTGET='sudo -E apt-get -q --yes'
# Short list of packages or quick-running command
SHORT_APTGET="lilto $SUDOAPTGET"
# Long list / long-running command
LONG_APTGET="bigto $SUDOAPTGET"

# Packages in generic VM images that conflict with containers/storage testing
RPMS_CONFLICTING="gcc-go"
DEBS_CONFLICTING=""

bad_os_id_ver() {
    die "Unknown/Unsupported distro. $OS_RELEASE_ID and/or version $OS_RELEASE_VER for $(basename $0)"
}

lilto() { err_retry 8 1000 "" "$@"; }  # just over 4 minutes max
bigto() { err_retry 7 5670 "" "$@"; }  # 12 minutes max

install_fuse_overlayfs_from_git(){
    wd=$(pwd)
    DEST="$GOPATH/src/github.com/containers/fuse-overlayfs"
    rm -rf "$DEST"
    ooe.sh git clone https://github.com/containers/fuse-overlayfs.git "$DEST"
    cd "$DEST"
    ooe.sh git fetch origin --tags
    ooe.sh ./autogen.sh
    ooe.sh ./configure
    ooe.sh make
    sudo make install prefix=/usr
    cd $wd
}

install_bats_from_git(){
    git clone https://github.com/bats-core/bats-core --depth=1
    sudo ./bats-core/install.sh /usr
    rm -rf bats-core
    mkdir -p ~/.parallel
    touch ~/.parallel/will-cite
}

showrun() {
    if [[ "$1" == "--background" ]]
    then
        shift
        # Properly escape any nested spaces, so command can be copy-pasted
        msg '+ '$(printf " %q" "$@")' &'
        "$@" &
        msg -e "${RED}<backgrounded>${NOR}"
    else
        msg '--------------------------------------------------'
        msg '+ '$(printf " %q" "$@") > /dev/stderr
        "$@"
    fi
}
