PROJECT_NAME=flightcontrolcenter
PROJECT_PATH=github.com/ferux/$(PROJECT_NAME)

BRANCH=$(shell git symbolic-ref --short HEAD)
REVISION=$(shell git rev-parse --short HEAD)

default: run

.PHONY: run
run: build
	@echo ">"Running...
	@./bin/fcc

.PHONY: build
build: check
	@echo ">"Building...
	@go build -ldflags '-X $(PROJECT_PATH).Revision=$(REVISION) -X $(PROJECT_PATH).Branch=$(BRANCH)' -o ./bin/fcc ./internal/cmd/main.go

.PHONY: check
check:
	@echo ">"Inspecting code...
	@golangci-lint run