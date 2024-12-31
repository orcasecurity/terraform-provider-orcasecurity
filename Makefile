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
