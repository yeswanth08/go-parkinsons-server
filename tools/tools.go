//go:build tools

package tools

import (
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
	_ "google.golang.org/grpc/cmd/protoc-gen-go-grpc"
)


// forcing go to include this in the mod file