MIGRATE=migrate -path migrations -database $(DB_DSN)
BINARY_NAME=main

gen-proto:
	protoc --go_out=gen --go_opt=paths=source_relative --go-grpc_out=gen --go-grpc_opt=paths=source_relative proto/sources/sources.proto
test:
	go test ./internal/service/mock

build:
	go build -o bin/$(BINARY_NAME) cmd/sources/main.go

run: build
	./bin/$(BINARY_NAME)

migrate-up:
	$(MIGRATE) up

migrate-down:
	$(MIGRATE) down 1

migrate-create:
	migrate create -ext sql -dir migrations -seq $(name)

.PHONY: gen-proto-projects protogen build test  migrate-up migrate-down migrate-create