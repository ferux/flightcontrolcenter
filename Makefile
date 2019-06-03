GO=go

PKG=$(shell $(GO) list | head -1 | sed -e 's/.*///')
PKG_PATH=$(shell $(GO) list | head -1)

BRANCH=$(shell git symbolic-ref --short HEAD)
REVISION=$(shell git rev-parse --short HEAD)
OUT?=bin/fcc

GOOS?=linux
GOARCH?=amd64

SSH_HOST?=localhost
SSH_USER?=root

default: build

.PHONY: run
run: build
	@echo ">"Running...
	@./bin/fcc

.PHONY: build
build:
	@echo ">"Building...
	@go build -ldflags '-X $(PKG_PATH).Revision=$(REVISION) -X $(PKG_PATH).Branch=$(BRANCH)' -o $(OUT) ./internal/cmd/main.go

.PHONY: build_linux
build_linux: export GOOS=linux
build_linux: export GOARCH=amd64
build_linux: export OUT = bin/fcc_linux
build_linux: build
	@echo ">"Built for linux!

.PHONY: build_remote
build_remote: check
	@git diff --quiet
	@ssh $(SSH_USER)@$(SSH_HOST) /opt/fcc/deploy.sh

.PHONY: check
check:
	@echo ">"Inspecting code...
	@golangci-lint run && echo ">>"Everything is okay! || echo !!Oopsie

.PHONY: prepare
prepare:
	@echo ">"Installing linter
	@GO111MODULE=off go get -u github.com/golangci/golangci-lint/cmd/golangci-lint

.PHONY: deploy
deploy: build_linux
	@echo ">"Stoping service
	@ssh $(SSH_USER)@$(SSH_HOST) systemctl stop fcc
	@echo ">"Copying binary file
	@scp bin/fcc_linux $(SSH_USER)@$(SSH_HOST):/opt/fcc/fcc
	@echo ">"Starting service
	@ssh $(SSH_USER)@$(SSH_HOST) systemctl start fcc
