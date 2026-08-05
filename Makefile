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

# Validate shift-left examples (embedded into docs/ by tfplugindocs).
validate-examples:
	./scripts/validate-examples.sh

# Pin golangci-lint to .golangci-version (same file CI reads).
GOLANGCI_LINT_VERSION := $(shell cat .golangci-version)
GOLANGCI_LINT := $(CURDIR)/bin/golangci-lint

# Install/reinstall the pinned binary into ./bin so `make lint` matches CI.
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
