BINARY_NAME=main

gen-proto:
	protoc --go_out=gen --go_opt=paths=source_relative --go-grpc_out=gen --go-grpc_opt=paths=source_relative proto/sources/sources.proto
test:
	go test ./internal/service/mock

build:
	go build -o bin/$(BINARY_NAME) cmd/sources/main.go

run: build
	./bin/$(BINARY_NAME)


.PHONY: gen-proto-projects protogen build test