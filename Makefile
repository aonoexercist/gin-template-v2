SHELL := /bin/bash
.DEFAULT_GOAL := help

# Load .env if present (DATABASE_URL, POSTGRES_*, ...).
-include .env
export

# Container engine: `make up` (docker) or `make up CONTAINER=podman`.
CONTAINER ?= podman
COMPOSE   ?= $(CONTAINER)-compose

APP       := api
MODULE    := $(shell go list -m 2>/dev/null)
VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT    ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
LDFLAGS   := -s -w -X $(MODULE)/internal/version.Version=$(VERSION) -X $(MODULE)/internal/version.Commit=$(COMMIT)

MIGRATIONS_DIR := db/migrations
GOOSE := goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)"

.PHONY: help
help: ## Show this help
	@awk 'BEGIN{FS=":.*##"} /^[a-zA-Z_-]+:.*##/ {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

##@ Develop
.PHONY: run dev build test cover lint fmt vet tidy
run: ## Run the API locally (needs a database: make db-up migrate-up)
	go run ./cmd/api

dev: ## Run with hot reload (air)
	air

build: ## Build ./bin/api
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(APP) ./cmd/api

test: ## Run unit tests with the race detector
	go test -race ./...

cover: ## Tests with coverage report
	go test -race -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -n 1

lint: ## Run golangci-lint
	golangci-lint run

fmt: ## Format code
	gofmt -s -w .

vet: ## go vet
	go vet ./...

tidy: ## go mod tidy
	go mod tidy

##@ Database
.PHONY: sqlc migrate-up migrate-down migrate-status migrate-create db-up db-psql
sqlc: ## Regenerate Go code from SQL (internal/features/*/queries.sql)
	sqlc generate

migrate-up: ## Apply all pending migrations
	$(GOOSE) up

migrate-down: ## Roll back the last migration
	$(GOOSE) down

migrate-status: ## Show migration status
	$(GOOSE) status

migrate-create: ## Create a migration: make migrate-create name=add_posts
	@test -n "$(name)" || (echo "usage: make migrate-create name=add_posts" && exit 1)
	goose -dir $(MIGRATIONS_DIR) create $(name) sql

db-up: ## Start only PostgreSQL in a container
	$(COMPOSE) up -d db

db-psql: ## Open psql inside the db container
	$(COMPOSE) exec db psql -U $${POSTGRES_USER:-app} -d $${POSTGRES_DB:-app}

##@ Containers (docker or podman)
.PHONY: up down logs image
up: ## Build and start db + migrations + api
	$(COMPOSE) up -d --build

down: ## Stop everything (keeps data volume)
	$(COMPOSE) down

logs: ## Tail api logs
	$(COMPOSE) logs -f api

image: ## Build the runtime image
	$(CONTAINER) build --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) -t $(APP):$(VERSION) .

##@ Tooling
.PHONY: tools
tools: ## Install goose, sqlc, air, golangci-lint
	go install github.com/pressly/goose/v3/cmd/goose@v3.22.1
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.27.0
	go install github.com/air-verse/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.61.0
