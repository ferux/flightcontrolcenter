// +build tools

package tools

import (
	_ "github.com/daixiang0/gci"
	_ "github.com/go-bindata/go-bindata/go-bindata"
	_ "github.com/gogo/protobuf/protoc-gen-gofast"
	_ "github.com/gogo/protobuf/protoc-gen-gogo"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/mwitkow/go-proto-validators/protoc-gen-govalidators"
	_ "github.com/valyala/quicktemplate/qtc"
)
