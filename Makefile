SHELL := /bin/sh
.DEFAULT_GOAL := cli

cli:
	go build -o bin/pressluft ./cmd/pressluft

dev:
	go run ./cmd/pressluft dev

clean:
	rm -rf bin/

.PHONY: cli dev clean
