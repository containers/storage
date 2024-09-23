go 1.22.0

// Warning: Ensure the "go" and "toolchain" versions match exactly to prevent unwanted auto-updates.
// That generally means there should be no toolchain directive present.

module github.com/containers/storage

require (
	github.com/BurntSushi/toml v1.4.0
	github.com/Microsoft/go-winio v0.6.2
	github.com/Microsoft/hcsshim v0.12.6
	github.com/containerd/stargz-snapshotter/estargz v0.15.1
	github.com/cyphar/filepath-securejoin v0.3.2
	github.com/docker/go-units v0.5.0
	github.com/google/go-intervals v0.0.2
	github.com/hashicorp/go-multierror v1.1.1
	github.com/json-iterator/go v1.1.12
	github.com/klauspost/compress v1.17.10
	github.com/klauspost/pgzip v1.2.6
	github.com/mattn/go-shellwords v1.0.12
	github.com/mistifyio/go-zfs/v3 v3.0.1
	github.com/moby/sys/capability v0.2.0
	github.com/moby/sys/mountinfo v0.7.2
	github.com/moby/sys/user v0.3.0
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/runtime-spec v1.2.0
	github.com/opencontainers/selinux v1.11.0
	github.com/sirupsen/logrus v1.9.3
	github.com/stretchr/testify v1.9.0
	github.com/tchap/go-patricia/v2 v2.3.1
	github.com/ulikunitz/xz v0.5.12
	github.com/vbatts/tar-split v0.11.5
	golang.org/x/exp v0.0.0-20240823005443-9b4947da3948
	golang.org/x/sys v0.25.0
	gotest.tools v2.2.0+incompatible
)

require (
	github.com/containerd/cgroups/v3 v3.0.3 // indirect
	github.com/containerd/errdefs v0.1.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.opencensus.io v0.24.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240701130421-f6361c86f094 // indirect
	google.golang.org/grpc v1.65.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
