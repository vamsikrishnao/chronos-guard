.PHONY: proto build run client clean

proto:
	@echo "Compiling Protocol Buffer interfaces..."
	protoc \
		--plugin=protoc-gen-go=$(shell go env GOPATH)/bin/protoc-gen-go \
		--plugin=protoc-gen-go-grpc=$(shell go env GOPATH)/bin/protoc-gen-go-grpc \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/chronos/v1/guard.proto

build: proto
	@echo "Compiling system binaries..."
	go build -o bin/sidecar cmd/server/main.go
	go build -o bin/client_mock cmd/client_mock/main.go

run: build
	./bin/sidecar

client:
	./bin/client_mock

clean:
	rm -rf bin/
	find proto/ -name "*.pb.go" -delete