.PHONY: run build test lint sqlc

run:
	go run ./cmd/server

build:
	CGO_ENABLED=0 go build -v -o ../../bin/api-server ./cmd/server

test:
	go test ./... -race -cover

lint:
	golangci-lint run ./...

sqlc:
	sqlc generate
