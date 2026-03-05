SHELL := /bin/sh

GO ?= go
NPM ?= npm
APP_BINARY ?= bin/pressluft
AGENT_BINARY ?= bin/pressluft-agent
DEV_API_PORT ?= 8081
DEV_UI_PORT ?= 8080
DEV_UI_HOST ?= 0.0.0.0

WEB_DIR := web
EMBED_DIST_DIR := internal/server/dist

.PHONY: build dev run clean format lint test check agent agent-dev all

build:
	@if [ ! -d "$(WEB_DIR)/node_modules" ]; then $(NPM) --prefix "$(WEB_DIR)" install; fi
	$(NPM) --prefix "$(WEB_DIR)" run generate
	test -f "$(WEB_DIR)/.output/public/index.html"
	rm -rf "$(EMBED_DIST_DIR)"
	mkdir -p "$(EMBED_DIST_DIR)"
	touch "$(EMBED_DIST_DIR)/.gitkeep"
	cp -R "$(WEB_DIR)/.output/public/." "$(EMBED_DIST_DIR)/"
	mkdir -p "$(dir $(APP_BINARY))"
	$(GO) build -o "$(APP_BINARY)" ./cmd

agent:
	CGO_ENABLED=0 $(GO) build -o "$(AGENT_BINARY)" ./cmd/pressluft-agent

agent-dev:
	CGO_ENABLED=0 $(GO) build -tags dev -o "$(AGENT_BINARY)" ./cmd/pressluft-agent

all: build agent

dev: agent-dev
	@if [ ! -d "$(WEB_DIR)/node_modules" ]; then $(NPM) --prefix "$(WEB_DIR)" install; fi
	DEV_API_PORT="$(DEV_API_PORT)" DEV_UI_PORT="$(DEV_UI_PORT)" DEV_UI_HOST="$(DEV_UI_HOST)" WEB_DIR="$(WEB_DIR)" NPM="$(NPM)" GO="$(GO)" ./ops/scripts/dev.sh

run: build
	./$(APP_BINARY)

format:
	$(GO) fmt ./...

lint:
	$(GO) vet ./...

test:
	$(GO) test ./...

check: format lint test build

clean:
	rm -f "$(APP_BINARY)" "$(AGENT_BINARY)"
