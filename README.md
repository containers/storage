`storage` is a Go library which aims to provide methods for storing filesystem
layers, container images, and containers.  An `oci-storage` (name not yet
final) CLI wrapper is also included for manual and scripting use.

To build the CLI wrapper, use 'make build-binary', optionally passing
'AUTO_GOPATH=1' as an additional argument to avoid having to set $GOPATH
manually.  For information on other recognized targets, run 'make help'.

Operations which use VMs expect to launch them using 'vagrant', defaulting to
using its 'libvirt' provider.  The boxes used are also available for the
'virtualbox' provider, and can be selected by setting $VAGRANT_PROVIDER to
'virtualbox' before kicking off the build.
