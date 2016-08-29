.PHONY: all binary build build-binary build-gccgo bundles cross default docs gccgo test test-integration-cli test-unit validate help win tgz

# set the graph driver as the current graphdriver if not set
DRIVER := $(if $(STORAGE_DRIVER),$(STORAGE_DRIVER),$(if $(DOCKER_GRAPHDRIVER),DOCKER_GRAPHDRIVER),$(shell docker info 2>&1 | grep "Storage Driver" | sed 's/.*: //'))

GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null)
GIT_BRANCH_CLEAN := $(shell echo $(GIT_BRANCH) | sed -e "s/[^[:alnum:]]/-/g")

RUNINVM := vagrant/runinvm.sh

default all: build ## validate all checks, build linux binaries, run all tests\ncross build non-linux binaries and generate archives\nusing VMs
	$(RUNINVM) hack/make.sh

build build-binary: bundles ## build using go on the host
	hack/make.sh binary

build-gccgo: bundles ## build using gccgo on the host
	hack/make.sh gccgo

binary: bundles
	$(RUNINVM) hack/make.sh binary

bundles:
	mkdir -p bundles

cross: build ## cross build the binaries for darwin, freebsd and windows\nusing VMs
	$(RUNINVM) hack/make.sh binary cross

win: build ## cross build the binary for windows using VMs
	$(RUNINVM) hack/make.sh win

tgz: build ## build the archives (.zip on windows and .tgz otherwise)\ncontaining the binaries on the host
	hack/make.sh binary cross tgz

docs: ## build the docs on the host
	$(MAKE) -C docs docs

gccgo: build-gccgo ## build the gcc-go linux binaries using VMs
	$(RUNINVM) hack/make.sh gccgo

test: build ## run the unit and integration tests using VMs
	$(RUNINVM) hack/make.sh binary cross test-unit test-integration-cli

test-integration-cli: build ## run the integration tests using VMs
	$(RUNINVM) hack/make.sh binary test-integration-cli

test-unit: build ## run the unit tests using VMs
	$(RUNINVM) hack/make.sh test-unit

validate: build ## validate DCO, Seccomp profile generation, gofmt,\n./pkg/ isolation, golint, tests, tomls, go vet and vendor\nusing VMs
	$(RUNINVM) hack/make.sh validate-dco validate-gofmt validate-pkg validate-lint validate-test validate-toml validate-vet validate-vendor

help: ## this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-z A-Z_-]+:.*?## / {gsub(" ",",",$$1);gsub("\\\\n",sprintf("\n%22c"," "), $$2);printf "\033[36m%-21s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

