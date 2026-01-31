// Package proto contains generated protobuf code for the cali calendar service.
//
// The protobuf definitions are in calendar.proto and are compiled to Go code
// using buf generate. To regenerate the code, run:
//
//	go generate ./...
//
// This will invoke buf generate to create:
//   - calendar.pb.go: Protocol buffer message definitions
//   - calendar_grpc.pb.go: gRPC service stubs
//   - calendar_cli.pb.go: proto-cli command-line interface code
//
// Generated files are automatically formatted with golangci-lint fmt and gofumpt.
package proto

//go:generate go run github.com/bufbuild/buf/cmd/buf generate
//go:generate go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint fmt .
//go:generate go run mvdan.cc/gofumpt -w .
