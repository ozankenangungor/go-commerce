SHELL := /bin/bash

COMPOSE_FILE := deployments/docker-compose.yaml
ENV_FILE ?= .env

.PHONY: help fmt lint test compose-up compose-down compose-logs compose-ps buf-lint buf-generate tools

help: ## Show available targets
	@awk 'BEGIN {FS = ":.*##"; printf "Available targets:\n"} /^[a-zA-Z0-9_-]+:.*##/ {printf "  %-14s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

fmt: ## Format Go files
	go fmt ./...

lint: ## Run linters
	golangci-lint run ./...

test: ## Run tests
	go test -race ./...

compose-up: ## Start local infrastructure
	docker compose -f $(COMPOSE_FILE) --env-file $(ENV_FILE) up -d

compose-down: ## Stop local infrastructure
	docker compose -f $(COMPOSE_FILE) --env-file $(ENV_FILE) down

compose-logs: ## Tail local infrastructure logs
	docker compose -f $(COMPOSE_FILE) --env-file $(ENV_FILE) logs -f

compose-ps: ## Show local infrastructure container status
	docker compose -f $(COMPOSE_FILE) --env-file $(ENV_FILE) ps

buf-lint: ## Lint protobuf sources
	buf lint

buf-generate: ## Generate protobuf Go code
	buf generate

tools: ## Print tool versions
	@set -euo pipefail; \
	go version; \
	docker --version; \
	docker compose version; \
	buf --version; \
	golangci-lint --version
