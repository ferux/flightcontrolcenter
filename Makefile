export GO111MODULE=on
export GOFLAGS=-tags=netgo

GO=go

PKG=$(shell $(GO) list | head -1 | sed -e 's/.*///')
PKG_PATH=$(shell $(GO) list | head -1)

BRANCH?=$(shell git symbolic-ref --short HEAD)
REVISION?=$(shell git rev-parse --short HEAD)
ENV?=production
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
build: build_static
	@echo ">"Building...
	@$(GO) build -mod=vendor -ldflags '-X main.revision=$(REVISION) -X main.branch=$(BRANCH) -X main.env=$(ENV)' -o $(OUT) ./internal/cmd/main.go

.PHONY: build_static
build_static: 
	@echo ">"Embedding static files...
	@bin/go-bindata -fs -prefix "assets/swagger" -pkg static -o internal/static/assets.go assets/swagger
	@echo ">"Building templates
	@bin/qtc -dir=./internal/templates

.PHONY: build_linux
build_linux: export GOOS=linux
build_linux: export GOARCH=amd64
build_linux: export OUT = bin/fcc_linux
build_linux: build
	@echo ">"Built for linux!

.PHONY: check
check:
	@echo ">"Inspecting code...
	@golangci-lint run && echo ">>"Everything is okay! || echo !!Oopsie

.PHONY: prepare
prepare: install_tools
	@echo ">"Installing linter
	@GO111MODULE=off go get -u github.com/golangci/golangci-lint/cmd/golangci-lint


.PHONY: install_tools
install_tools:
	@echo ">"Updating go-bindata
	@GOBIN="$$PWD/bin" $(GO) get github.com/go-bindata/go-bindata@v3.1.2
	@echo ">"Updating go-bindata binaries
	@GOBIN="$$PWD/bin" $(GO) get github.com/go-bindata/go-bindata/...@v3.1.2
	@echo ">"Updating quicktemplates
	@GOBIN="$$PWD/bin" $(GO) get github.com/valyala/quicktemplate/qtc@v1.1.1

.PHONY: test
test:
	go test -race -timeout 60s ./internal/...

.PHONY: vendor
vendor:
	go mod tidy
	go mod vendor

.PHONY: ssh_deploy
ssh_deploy: build_linux
	$(info >Deploying via ssh)
	@sh scripts/ssh_deploy.sh
