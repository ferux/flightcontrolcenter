export GO111MODULE=on
export GOFLAGS=-mod=vendor
export GOBIN=$(PWD)/bin/$(shell go env GOHOSTOS)-$(shell go env GOHOSTARCH)

PKG=$(shell go list | head -1 | sed -e 's/.*///')
PKG_PATH=$(shell go list | head -1)

BRANCH?=$(shell git symbolic-ref --short HEAD)
REVISION?=$(shell git rev-parse --short HEAD)

ENV?=production
OUT?=bin/fcc

GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)


default: build

.PHONY: run
run: build
	$(info >Running binary $(OUT))
	@$(OUT)

.PHONY: build
build: build_static
	$(info >Building for $(GOOS)-$(GOARCH))
	@go build -ldflags '-X main.revision=$(REVISION) -X main.branch=$(BRANCH) -X main.env=$(ENV)' -o $(OUT) ./internal/cmd/main.go

.PHONY: build_static
build_static: 
	$(info >Appending swagger)
	@$(GOBIN)/go-bindata -fs -prefix "assets/swagger" -pkg static -o internal/static/assets.go assets/swagger
	$(info >Generating quick template)
	@$(GOBIN)/qtc -dir=./internal/templates

.PHONY: build_linux
build_linux: export GOOS=linux
build_linux: export GOARCH=amd64
build_linux: export OUT = bin/fcc_linux
build_linux: build

.PHONY: check
check:
	$(info >Organizing imports)
	@$(GOBIN)/gci -local $(cat go.mod | grep module | sed 's/module //g') ./internal/

	$(info >Inspecting code)
	@$(GOBIN)/golangci-lint run

.PHONY: test
test:
	go test -count=1 -timeout 60s ./internal/...

.PHONY: ssh_deploy
ssh_deploy: build_linux
	$(info >Deploying via ssh)
	@sh scripts/ssh_deploy.sh

install_tools:
	$(info >Installing tools)
	@cat tools.go | grep _ | sed -e 's/.*_ "//g' | sed -e 's/"//g' | xargs -tI % go install %

proto_gen: export PATH := ${PATH}:${PWD}/bin
proto_gen: export GOPATH := $(shell go env GOPATH)
proto_gen:
	$(info >Generating .proto files)
	protoc \
	-I internal/keeper/talk \
	-I ${GOPATH}/src \
	-I vendor/github.com/gogo/protobuf/proto/ \
	-I vendor/github.com/gogo/protobuf/plugin/ \
	-I vendor/github.com/mwitkow/go-proto-validators/ \
	-I vendor/ \
	--gofast_out=plugins=grpc\
	,Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types\
	:internal/keeper/talk/ \
	--govalidators_out=gogoimport=true\
	,Mgoogle/protobuf/timestamp.proto=github.com/gogo/protobuf/types\
	:internal/keeper/talk/ \
	internal/keeper/talk/*.proto

DOCKER_IMAGE?=fcc
DOCKER_FLAGS?=--rm\
	-v "$$PWD":/go/src/flightcontrolcenter\
	-w /go/src/flightcontrolcenter

# assume image is ffc
build_docker:
	docker run $(DOCKER_FLAGS) $(DOCKER_IMAGE) make install_tools proto_gen check test build

make_image:
	docker build -t $(DOCKER_IMAGE) .
