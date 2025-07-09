# https://gist.github.com/alexedwards/3b40775846535d0014ab1ff477e4a568

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## tidy: format code and tidy modfile
.PHONY: tidy
tidy:
	go fmt ./...
	go mod tidy -v

## build: build the project
.PHONY: build
build:
	go build ./...

## test: run all tests
.PHONY: test
test:
	go test -race -v ./...

## test/cover: run all tests and display coverage
.PHONY: test/cover
test/cover:
	go test -v -race -buildvcs -coverprofile=/tmp/coverage.out ./...
	go tool cover -html=/tmp/coverage.out

## lint: run golangci-lint
.PHONY: lint
lint:
	golangci-lint run

## bench: run go benchmarks
.PHONY: bench
bench:
	go test -bench=. ./spectator/meter

## install: install golangci-lint and check versions
.PHONY: install
install:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.2.1
	go version
	golangci-lint --version
