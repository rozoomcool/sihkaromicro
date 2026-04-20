BINARY_NAME=main

gen-proto-projects:
	protoc --go_out=gen --go_opt=paths=source_relative --go-grpc_out=gen --go-grpc_opt=paths=source_relative proto/projects/projects.proto
# 	protoc --go_out=generated --go_opt=paths=source_relative --go-grpc_out=generated --go-grpc_opt=paths=source_relative api/v1/auth.proto

build:
	go build -o bin/$(BINARY_NAME) cmd/projects/main.go

run: build
	./bin/$(BINARY_NAME)

gettoken:
	./bin/gettoken.sh

.PHONY: gen-proto-projects protogen build