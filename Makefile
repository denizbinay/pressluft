SHELL := /bin/sh
.DEFAULT_GOAL := help

GO ?= go
GO_TEST ?= $(GO) test
NPM ?= pnpm
APP_BINARY ?= bin/pressluft
AGENT_BINARY ?= bin/pressluft-agent
DEV_API_PORT ?= 8081
DEV_UI_PORT ?= 8080
DEV_UI_HOST ?= 0.0.0.0
TMPDIR ?= /tmp
GOCACHE ?= $(TMPDIR)/go-build
ANSIBLE_LOCAL_TEMP ?= $(TMPDIR)/ansible-local
ANSIBLE_REMOTE_TEMP ?= $(TMPDIR)/ansible-remote
NODE_OPTIONS ?= --max-old-space-size=8192

WEB_DIR := web
EMBED_DIST_DIR := internal/server/dist

GO_ENV = env TMPDIR="$(TMPDIR)" GOCACHE="$(GOCACHE)"
ANSIBLE_ENV = env TMPDIR="$(TMPDIR)" ANSIBLE_LOCAL_TEMP="$(ANSIBLE_LOCAL_TEMP)" ANSIBLE_REMOTE_TEMP="$(ANSIBLE_REMOTE_TEMP)"
WEB_ENV = env NODE_OPTIONS="$(NODE_OPTIONS)"

UNIT_TEST_PACKAGES := \
	./cmd \
	./cmd/pressluft-agent \
	./internal/agent \
	./internal/agentauth \
	./internal/agentcommand \
	./internal/auth \
	./internal/contract \
	./internal/database \
	./internal/dispatch \
	./internal/envconfig \
	./internal/platform \
	./internal/registration \
	./internal/security \
	./internal/worker \
	./internal/ws

INTEGRATION_TEST_PACKAGES := \
	./internal/activity \
	./internal/orchestrator \
	./internal/provider/... \
	./internal/runner/ansible \
	./internal/server \
	./internal/server/profiles

ANSIBLE_PLAYBOOKS := \
	ops/ansible/playbooks/hetzner/delete.yml \
	ops/ansible/playbooks/hetzner/firewalls.yml \
	ops/ansible/playbooks/hetzner/provision.yml \
	ops/ansible/playbooks/hetzner/rebuild.yml \
	ops/ansible/playbooks/hetzner/resize.yml \
	ops/ansible/playbooks/hetzner/volume.yml

.PHONY: help prepare-env generate-contract generate-api-contract contract-json frontend-install frontend-generate embed-web build app-build agent agent-dev all dev dev-lab dev-status dev-reset run format fmt-check lint test test-unit test-integration validate-go validate-web validate-profiles validate-profile-schema validate-profile-consistency ansible-syntax ansible-check ansible-validate validate clean

help:
	@printf '%s\n' 'Targets:'
	@printf '  %-28s %s\n' 'help' 'Show supported build and validation targets'
	@printf '  %-28s %s\n' 'prepare-env' 'Create writable cache and temp directories for local tooling'
	@printf '  %-28s %s\n' 'generate-contract' 'Refresh the generated TS runtime and HTTP API contracts from Go'
	@printf '  %-28s %s\n' 'generate-api-contract' 'Refresh the generated TS HTTP API contract from Go'
	@printf '  %-28s %s\n' 'contract-json' 'Print the runtime contract and env config contract as JSON'
	@printf '  %-28s %s\n' 'frontend-generate' 'Build the Nuxt static frontend into web/.output/public'
	@printf '  %-28s %s\n' 'build' 'Generate the frontend, embed it, and build the control-plane binary'
	@printf '  %-28s %s\n' 'agent' 'Build the production agent binary'
	@printf '  %-28s %s\n' 'agent-dev' 'Build the dev agent binary'
	@printf '  %-28s %s\n' 'validate-go' 'Run formatting, vet, and the supported Go test suites'
	@printf '  %-28s %s\n' 'ansible-validate' 'Run playbook syntax checks and profile contract validation'
	@printf '  %-28s %s\n' 'validate-web' 'Generate the frontend static assets and verify the output exists'
	@printf '  %-28s %s\n' 'validate' 'Run the full supported validation suite for this repository'
	@printf '  %-28s %s\n' 'all' 'Build the control plane and agent binaries'
	@printf '  %-28s %s\n' 'dev' 'Run the disposable local dev stack with session-scoped remote connectivity'
	@printf '  %-28s %s\n' 'dev-lab' 'Run the stable local lab workflow for durable remote-agent testing'
	@printf '  %-28s %s\n' 'dev-status' 'Inspect the local Pressluft state bundle and callback durability'
	@printf '  %-28s %s\n' 'dev-reset' 'Reset the local Pressluft state bundle (requires CONFIRM=1)'

prepare-env:
	mkdir -p "$(GOCACHE)" "$(ANSIBLE_LOCAL_TEMP)" "$(ANSIBLE_REMOTE_TEMP)"

generate-contract: prepare-env
	$(GO_ENV) $(GO) run ./cmd/pressluft-contractgen -format ts > "$(WEB_DIR)/app/lib/platform-contract.generated.ts"
	$(GO_ENV) $(GO) run ./cmd/pressluft-contractgen -format api-ts > "$(WEB_DIR)/app/lib/api-contract.ts"

generate-api-contract: prepare-env
	$(GO_ENV) $(GO) run ./cmd/pressluft-contractgen -format api-ts > "$(WEB_DIR)/app/lib/api-contract.ts"

contract-json: prepare-env
	$(GO_ENV) $(GO) run ./cmd/pressluft-contractgen -format json

frontend-install:
	@if [ ! -d "$(WEB_DIR)/node_modules" ]; then $(NPM) --prefix "$(WEB_DIR)" install; fi

frontend-generate: generate-contract frontend-install
	$(WEB_ENV) $(NPM) --prefix "$(WEB_DIR)" run generate
	test -f "$(WEB_DIR)/.output/public/index.html"

embed-web: frontend-generate
	rm -rf "$(EMBED_DIST_DIR)"
	mkdir -p "$(EMBED_DIST_DIR)"
	touch "$(EMBED_DIST_DIR)/.gitkeep"
	cp -R "$(WEB_DIR)/.output/public/." "$(EMBED_DIST_DIR)/"

app-build: prepare-env
	mkdir -p "$(dir $(APP_BINARY))"
	$(GO_ENV) $(GO) build -o "$(APP_BINARY)" ./cmd

build: embed-web app-build

agent: prepare-env
	CGO_ENABLED=0 $(GO_ENV) $(GO) build -o "$(AGENT_BINARY)" ./cmd/pressluft-agent

agent-dev: prepare-env
	CGO_ENABLED=0 $(GO_ENV) $(GO) build -tags dev -o "$(AGENT_BINARY)" ./cmd/pressluft-agent

all: build agent

dev: agent-dev frontend-install
	TMPDIR="$(TMPDIR)" GOCACHE="$(GOCACHE)" ANSIBLE_LOCAL_TEMP="$(ANSIBLE_LOCAL_TEMP)" ANSIBLE_REMOTE_TEMP="$(ANSIBLE_REMOTE_TEMP)" DEV_WORKFLOW="dev" DEV_API_PORT="$(DEV_API_PORT)" DEV_UI_PORT="$(DEV_UI_PORT)" DEV_UI_HOST="$(DEV_UI_HOST)" WEB_DIR="$(WEB_DIR)" NPM="$(NPM)" GO="$(GO)" ./ops/scripts/dev.sh

dev-lab: agent-dev frontend-install
	TMPDIR="$(TMPDIR)" GOCACHE="$(GOCACHE)" ANSIBLE_LOCAL_TEMP="$(ANSIBLE_LOCAL_TEMP)" ANSIBLE_REMOTE_TEMP="$(ANSIBLE_REMOTE_TEMP)" DEV_WORKFLOW="lab" DEV_API_PORT="$(DEV_API_PORT)" DEV_UI_PORT="$(DEV_UI_PORT)" DEV_UI_HOST="$(DEV_UI_HOST)" WEB_DIR="$(WEB_DIR)" NPM="$(NPM)" GO="$(GO)" ./ops/scripts/dev.sh

dev-status: prepare-env
	$(GO_ENV) $(GO) run ./cmd/pressluft-devctl status

dev-reset: prepare-env
	@test "$(CONFIRM)" = "1" || { printf '%s\n' 'dev-reset is destructive. Re-run with CONFIRM=1.'; exit 1; }
	$(GO_ENV) $(GO) run ./cmd/pressluft-devctl reset --force

run: build
	./$(APP_BINARY)

format:
	$(GO_ENV) $(GO) fmt ./...

fmt-check:
	@unformatted="$$(gofmt -l cmd internal)"; \
	if [ -n "$$unformatted" ]; then \
		printf '%s\n' "$$unformatted"; \
		exit 1; \
	fi

lint: prepare-env
	$(GO_ENV) $(GO) vet ./...

test: test-unit test-integration

test-unit: prepare-env
	$(GO_ENV) $(GO_TEST) $(UNIT_TEST_PACKAGES)

test-integration: prepare-env
	$(GO_ENV) $(GO_TEST) -count=1 $(INTEGRATION_TEST_PACKAGES)

validate-go: fmt-check lint test

validate-profile-schema: prepare-env
	$(GO_ENV) $(GO_TEST) -count=1 ./internal/server/profiles -run TestProfileArtifactsSatisfySchema

validate-profile-consistency: prepare-env
	$(GO_ENV) $(GO_TEST) -count=1 ./internal/server/profiles -run TestRegistryMatchesProfileArtifacts

validate-profiles: validate-profile-schema validate-profile-consistency

ansible-syntax: prepare-env
	@set -e; \
	for playbook in $(ANSIBLE_PLAYBOOKS); do \
		$(ANSIBLE_ENV) ansible-playbook -i localhost, -c local --syntax-check "$$playbook"; \
	done

ansible-check:
	./ops/scripts/ansible_check_configure.sh

ansible-validate: ansible-syntax validate-profiles

validate-web: frontend-generate

validate: validate-go ansible-validate all

clean:
	rm -f "$(APP_BINARY)" "$(AGENT_BINARY)"
