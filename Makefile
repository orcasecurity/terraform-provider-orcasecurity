PKG_NAME?=orcasecurity

default: install

generate:
	go generate ./...

install:
	go install .

build:
	go build .

test-unit:
	go test -count=1 -parallel=4 ./...

test-ci:
	go test -v ./${PKG_NAME}/...

test-acc:
	TF_ACC=1 go test -count=1 -parallel=4 -timeout 10m -v ${TESTARGS} ./${PKG_NAME}/...

# examples/ is the source tfplugindocs embeds into docs/, so a stale attribute name there
# ships into the published docs. No Go test covers example HCL.
validate-examples:
	./scripts/validate-examples.sh

# .golangci-version is the single source of truth: the CI workflow reads the same file,
# so a locally installed golangci-lint cannot report a different set of issues than the
# PR gate. A newer version enables checks CI does not run yet (and vice versa).
GOLANGCI_LINT_VERSION := $(shell cat .golangci-version)
GOLANGCI_LINT := $(CURDIR)/bin/golangci-lint

# Installs the pinned version into ./bin on first use, and re-installs whenever the
# pinned version changes, so `make lint` always matches CI without touching $(GOPATH)/bin.
$(GOLANGCI_LINT):
	@mkdir -p $(CURDIR)/bin
	@echo "installing golangci-lint $(GOLANGCI_LINT_VERSION) into ./bin"
	@GOBIN=$(CURDIR)/bin go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

.PHONY: lint lint-version
lint: lint-version
	$(GOLANGCI_LINT) run ./...

lint-version:
	@if [ ! -x "$(GOLANGCI_LINT)" ] || ! $(GOLANGCI_LINT) --version 2>/dev/null | grep -q "$(patsubst v%,%,$(GOLANGCI_LINT_VERSION))"; then \
		$(MAKE) --no-print-directory $(GOLANGCI_LINT); \
	fi
	@$(GOLANGCI_LINT) --version
