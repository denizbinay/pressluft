SHELL := /bin/sh
.DEFAULT_GOAL := dev

dev:
	go run ./cmd/pressluft dev

build:
	go run ./cmd/pressluft build

test:
	go run ./cmd/pressluft test

validate:
	go run ./cmd/pressluft validate

.PHONY: dev build test validate
