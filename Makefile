export GO111MODULE=on
export GOFLAGS=-tags=netgo
export GOBIN=$(PWD)/bin

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
	@$(GOBIN)/go-bindata -fs -prefix "assets/swagger" -pkg static -o internal/static/assets.go assets/swagger
	@echo ">"Building templates
	@$(GOBIN)/qtc -dir=./internal/templates

.PHONY: build_linux
build_linux: export GOOS=linux
build_linux: export GOARCH=amd64
build_linux: export OUT = bin/fcc_linux
build_linux: build
	@echo ">"Built for linux!

.PHONY: check
check:
	@echo ">"Inspecting code...
	@$(GOBIN)/golangci-lint run && echo ">>"Everything is okay! || echo !!Oopsie

.PHONY: test
test:
	go test -timeout 60s ./internal/...

.PHONY: vendor
vendor:
	go mod tidy
	go mod vendor

.PHONY: ssh_deploy
ssh_deploy: build_linux
	$(info >Deploying via ssh)
	@sh scripts/ssh_deploy.sh

download:
	$(info downloading modules)
	@$(GO) mod download

install_tools:
	$(info installing tools)
	@cat tools.go | grep _ | sed -e 's/.*_ "//g' | sed -e 's/"//g' | xargs -tI % go install %

proto_gen:
	protoc -I internal/keeper/talk \
	-I bin/protoc-gen-gofast \
	--gofast_out=plugins=grpc:internal/keeper/talk/ \
	internal/keeper/talk/*.proto
