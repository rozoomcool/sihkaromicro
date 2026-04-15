BINARY_NAME=main

gen-proto-auth:
	protoc --go_out=gen --go_opt=paths=source_relative --go-grpc_out=gen --go-grpc_opt=paths=source_relative proto/sso/sso.proto
# 	protoc --go_out=generated --go_opt=paths=source_relative --go-grpc_out=generated --go-grpc_opt=paths=source_relative api/v1/auth.proto

build:
	go build -o bin/$(BINARY_NAME) cmd/sso/main.go

run: build
	./bin/$(BINARY_NAME)

.PHONY: protogen build