default: install

generate:
	go generate ./...

install:
	go install .

test:
	go test -count=1 -parallel=4 ./...

test-ci:
	go test -v -cover ./internal/* -cover ./orcasecurity/

testacc:
	TF_ACC=1 go test -count=1 -parallel=4 -timeout 10m -v ./...
