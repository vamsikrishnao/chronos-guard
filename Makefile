# Declare all recipes as PHONY targets to prevent timestamp tracking conflicts
.PHONY: proto build run client clean

# Zero-Dependency Proto Target: Looks up plugin binaries directly from Go's environment path parameters
proto:
	@echo "Compiling Protocol Buffer interfaces via direct binary injection..."
	protoc \
		--plugin=protoc-gen-go=$(shell go env GOPATH)/bin/protoc-gen-go \
		--plugin=protoc-gen-go-grpc=$(shell go env GOPATH)/bin/protoc-gen-go-grpc \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/chronos/v1/guard.proto

build: proto
	@echo "Compiling system binaries..."
	go build -o bin/sidecar cmd/sidecar/main.go
	go build -o bin/client_mock cmd/client_mock/main.go

run: build
	./bin/sidecar

client:
	./bin/client_mock

clean:
	rm -rf bin/
	find proto/ -name "*.pb.go" -delete