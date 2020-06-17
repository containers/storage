module github.com/containers/storage

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/Microsoft/go-winio v0.4.15-0.20190919025122-fc70bd9a86b5
	github.com/Microsoft/hcsshim v0.8.9
	github.com/containerd/cgroups v0.0.0-20200609174450-80c669f4bad0 // indirect
	github.com/containerd/containerd v1.4.0-beta.1.0.20200604173407-38cb1c1a54e3
	github.com/docker/go-events v0.0.0-20190806004212-e31b211e4f1c // indirect
	github.com/docker/go-units v0.4.0
	github.com/gogo/googleapis v1.4.0 // indirect
	github.com/google/uuid v1.1.1 // indirect
	github.com/hashicorp/go-multierror v1.1.0
	github.com/imdario/mergo v0.3.9 // indirect
	github.com/klauspost/compress v1.10.10
	github.com/klauspost/pgzip v1.2.4
	github.com/mattn/go-shellwords v1.0.10
	github.com/mistifyio/go-zfs v2.1.1+incompatible
	github.com/opencontainers/go-digest v1.0.0
	github.com/opencontainers/image-spec v1.0.1
	github.com/opencontainers/runc v1.0.0-rc90
	github.com/opencontainers/runtime-spec v1.0.2
	github.com/opencontainers/selinux v1.5.2
	github.com/pkg/errors v0.9.1
	github.com/pquerna/ffjson v0.0.0-20181028064349-e517b90714f7
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.6.1
	github.com/syndtr/gocapability v0.0.0-20180916011248-d98352740cb2
	github.com/tchap/go-patricia v2.3.0+incompatible
	github.com/vbatts/tar-split v0.11.1
	go.etcd.io/bbolt v1.3.4 // indirect
	golang.org/x/net v0.0.0-20191004110552-13f9640d40b9
	golang.org/x/sys v0.0.0-20200202164722-d101bd2416d5
	google.golang.org/grpc v1.29.1 // indirect
	gotest.tools v2.2.0+incompatible
	gotest.tools/v3 v3.0.2 // indirect
)

go 1.13
