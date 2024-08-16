GO := go
BUILDDIR := build

all: $(BUILDDIR)

.PHONY: vendor
vendor:
	$(GO) mod tidy
	$(GO) mod vendor
	$(GO) mod verify

define go-build
	$(shell cd `pwd` && $(GO) build -mod=vendor -o $(BUILDDIR)/$(shell basename $(1)) $(1))
	@echo > /dev/null
endef

.PHONY: clean
clean:
	rm -rf $(BUILDDIR)

$(BUILDDIR): \
	$(BUILDDIR)/git-validation \
	$(BUILDDIR)/go-md2man \
	$(BUILDDIR)/golangci-lint

$(BUILDDIR)/git-validation:
	$(call go-build,./vendor/github.com/vbatts/git-validation)

$(BUILDDIR)/go-md2man:
	$(call go-build,./vendor/github.com/cpuguy83/go-md2man)

$(BUILDDIR)/golangci-lint:
	@[ "${GOLANGCI_LINT_VERSION}" ] || ( echo "GOLANGCI_LINT_VERSION is not set"; exit 1 )
	curl -fsSL https://raw.githubusercontent.com/golangci/golangci-lint/v$(GOLANGCI_LINT_VERSION)/install.sh | sh -s -- -b ./$(BUILDDIR) v$(GOLANGCI_LINT_VERSION)
