.PHONY: all build build-gccgo cross default docs docs-build dynbinary gccgo test test-integration-cli test-unit validate help

# set the graph driver as the current graphdriver if not set
DRIVER := $(if $(DOCKER_GRAPHDRIVER),$(DOCKER_GRAPHDRIVER),$(shell docker info 2>&1 | grep "Storage Driver" | sed 's/.*: //'))

GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null)
GIT_BRANCH_CLEAN := $(shell echo $(GIT_BRANCH) | sed -e "s/[^[:alnum:]]/-/g")

default: all

all: build ## validate all checks, build linux binaries, run all tests\ncross build non-linux binaries and generate archives
	hack/make.sh

build: bundles
	hack/make.sh

build-gccgo: bundles
	hack/make.sh

dynbinary: bundles
	hack/make.sh dynbinary

bundles:
	mkdir bundles

cross: build ## cross build the binaries for darwin, freebsd and\nwindows
	hack/make.sh dynbinary cross

win: build ## cross build the binary for windows
	hack/make.sh win

tgz: build ## build the archives (.zip on windows and .tgz\notherwise) containing the binaries
	hack/make.sh dynbinary cross tgz

deb: build  ## build the deb packages
	hack/make.sh dynbinary build-deb

docs: ## build the docs
	$(MAKE) -C docs docs

gccgo: build-gccgo ## build the gcc-go linux binaries
	hack/make.sh gccgo

install: ## install the linux binaries
	KEEPBUNDLE=1 hack/make.sh install-binary

rpm: build ## build the rpm packages
	hack/make.sh dynbinary build-rpm

run: build ## run the docker daemon in a container
	sh -c "KEEPBUNDLE=1 hack/make.sh install-binary run"

test: build ## run the unit, integration and docker-py tests
	hack/make.sh dynbinary cross test-unit test-integration-cli test-docker-py

test-docker-py: build ## run the docker-py tests
	hack/make.sh dynbinary test-docker-py

test-integration-cli: build ## run the integration tests
	hack/make.sh dynbinary test-integration-cli

test-unit: build ## run the unit tests
	hack/make.sh test-unit

validate: build ## validate DCO, Seccomp profile generation, gofmt,\n./pkg/ isolation, golint, tests, tomls, go vet and vendor 
	hack/make.sh validate-dco validate-default-seccomp validate-gofmt validate-pkg validate-lint validate-test validate-toml validate-vet validate-vendor

help: ## this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {sub("\\\\n",sprintf("\n%22c"," "), $$2);printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

