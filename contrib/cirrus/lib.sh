

# Library of common, shared utility functions.  This file is intended
# to be sourced by other scripts, not called directly.

# BEGIN Global export of all variables
set -a

# Due to differences across platforms and runtime execution environments,
# handling of the (otherwise) default shell setup is non-uniform.  Rather
# than attempt to workaround differences, simply force-load/set required
# items every time this library is utilized.
USER="$(whoami)"
HOME="$(getent passwd $USER | cut -d : -f 6)"
# Some platforms set and make this read-only
[[ -n "$UID" ]] || \
    UID=$(getent passwd $USER | cut -d : -f 3)

# Automation library installed at image-build time,
# defining $AUTOMATION_LIB_PATH in this file.
if [[ -r "/etc/automation_environment" ]]; then
    source /etc/automation_environment
fi
# shellcheck disable=SC2154
if [[ -n "$AUTOMATION_LIB_PATH" ]]; then
        # shellcheck source=/usr/share/automation/lib/common_lib.sh
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
GOPATH="${GOPATH:-/var/tmp/go}"
GOCACHE="${GOCACHE:-$GOPATH/cache/go-build}"
# called processes like `make` and other tools need these vars.
eval "$(go env)"
CIRRUS_WORKING_DIR="${CIRRUS_WORKING_DIR:-$GOPATH/src/github.com/containers/storage}"
GOSRC="${GOSRC:-$CIRRUS_WORKING_DIR}"
PATH="$HOME/bin:$GOPATH/bin:/usr/local/bin:$PATH"
SCRIPT_BASE=${GOSRC}/contrib/cirrus

CI="${CI:-false}"
CIRRUS_CI="${CIRRUS_CI:-false}"
DEST_BRANCH="${DEST_BRANCH:-main}"
CONTINUOUS_INTEGRATION="${CONTINUOUS_INTEGRATION:-false}"
CIRRUS_REPO_NAME=${CIRRUS_REPO_NAME:-storage}
# Cirrus only sets $CIRRUS_BASE_SHA properly for PRs, but $EPOCH_TEST_COMMIT
# needs to be set from this value in order for `make validate` to run properly.
# When running get_ci_vm.sh, most $CIRRUS_xyz variables are empty. Attempt
# to accomidate both branch and get_ci_vm.sh testing by discovering the base
# branch SHA value.
if [[ -z "$CIRRUS_BASE_SHA" ]] && [[ -z "$CIRRUS_TAG" ]]
then  # Operating on a branch, or under `get_ci_vm.sh`
    CIRRUS_BASE_SHA=$(git rev-parse ${UPSTREAM_REMOTE:-origin}/$DEST_BRANCH)
elif [[ -z "$CIRRUS_BASE_SHA" ]]
then  # Operating on a tag
    CIRRUS_BASE_SHA=$(git rev-parse HEAD)
fi
# The starting place for linting and code validation
EPOCH_TEST_COMMIT="$CIRRUS_BASE_SHA"

# Unsafe env. vars for display
SECRET_ENV_RE='(IRCID)|(ACCOUNT)|(^GC[EP]..+)|(SSH)'

# Working with dnf + timeout/retry
SHORT_DNFY='lilto dnf -y'
LONG_DNFY='bigto dnf -y'
# Working with apt under Debian/Ubuntu automation is a PITA, make it easy
# Avoid some ways of getting stuck waiting for user input
DEBIAN_FRONTEND=noninteractive
# Short-cut for frequently used base command
SUDOAPTGET='sudo -E apt-get -q --yes'
# Short list of packages or quick-running command
SHORT_APTGET="lilto $SUDOAPTGET"
# Long list / long-running command
LONG_APTGET="bigto $SUDOAPTGET"

# Packages in generic VM images that conflict with containers/storage testing
RPMS_CONFLICTING="gcc-go"
DEBS_CONFLICTING=""

# END Global export of all variables
set +a

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

check_filesystem_supported(){
    if ! grep -q "	$1\$" /proc/filesystems ; then
        modprobe $1 > /dev/null 2> /dev/null || :en
        if ! grep -q "	$1\$" /proc/filesystems ; then
            echo "This CI VM does not support $TEST_DRIVER in its kernel"
	    false
        fi
    fi
    true
}
