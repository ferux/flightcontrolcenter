GO=go

PKG=$(shell $(GO) list | head -1 | sed -e 's/.*///')
PKG_PATH=$(shell $(GO) list | head -1)

BRANCH?=$(shell git symbolic-ref --short HEAD)
REVISION?=$(shell git rev-parse --short HEAD)
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
	@go build -ldflags '-X $(PKG_PATH).Revision=$(REVISION) -X $(PKG_PATH).Branch=$(BRANCH)' -o $(OUT) ./internal/cmd/main.go

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

.PHONY: build_remote
build_remote: check
	@git diff --quiet
	@ssh $(SSH_USER)@$(SSH_HOST) /opt/fcc/deploy.sh

.PHONY: check
check:
	@echo ">"Inspecting code...
	@golangci-lint run && echo ">>"Everything is okay! || echo !!Oopsie

.PHONY: prepare
prepare: install_tools
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

.PHONY: install_tools
install_tools:
	@echo ">"Updating go-bindata
	@GOBIN="$$PWD/bin" $(GO) get -u github.com/go-bindata/go-bindata/...@v3.1.2
	@echo ">"Updating quicktemplates
	@GOBIN="$$PWD/bin" $(GO) get -u github.com/valyala/quicktemplate/qtc@v1.1.1

.PHONY: test
test:
	go test -race -timeout 60s ./internal/...
