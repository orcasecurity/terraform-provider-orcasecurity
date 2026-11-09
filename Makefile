PKG_NAME?=orcasecurity
GORELEASER_VERSION?=v1.25.1
GORELEASER?=$(CURDIR)/bin/goreleaser

.PHONY: default changelog generate install build test-unit test-ci test-acc

default: install

# GoReleaser v1 needs Go <=1.23 (old x/tools) and the external linker so the
# binary has LC_UUID, which current macOS requires before it will run it.
$(GORELEASER):
	GOTOOLCHAIN=go1.22.12 CGO_ENABLED=1 GOBIN=$(CURDIR)/bin go install -ldflags=-linkmode=external github.com/goreleaser/goreleaser@$(GORELEASER_VERSION)

changelog: $(GORELEASER)
	GORELEASER=$(GORELEASER) scripts/changelog.sh update

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
