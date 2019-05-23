GO=go

PKG=$(shell $(GO) list | head -1 | sed -e 's/.*///')
PKG_PATH=$(shell $(GO) list | head -1)

BRANCH=$(shell git symbolic-ref --short HEAD)
REVISION=$(shell git rev-parse --short HEAD)

GOOS?=linux
GOARCH?=amd64

default: build

.PHONY: run
run: build
	@echo ">"Running...
	@./bin/fcc

.PHONY: build
build: check
	@echo ">"Building...
	@go build -ldflags '-X $(PKG_PATH).Revision=$(REVISION) -X $(PKG_PATH).Branch=$(BRANCH)' -o ./bin/fcc ./internal/cmd/main.go

.PHONY: check
check:
	@echo ">"Inspecting code...
	@golangci-lint run && echo ">>"Everything is okay! || echo !!Oopsie

.PHONY: prepare
prepare:
	@echo ">"Installing linter@v1.16.0
	@GO111MODULE=on go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.16.0