.PHONY: dev build test vet backend-gates

PORT ?= 18080

dev:
	go run ./cmd/pressluft dev --port $(PORT)

build:
	go build -o ./bin/pressluft ./cmd/pressluft

test:
	go test ./internal/... -v

vet:
	go vet ./...

backend-gates: build vet test
