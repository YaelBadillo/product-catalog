# Product Catalog development commands. `make` (or `make help`) lists them.
# Every command lives here with a `##` description.

# Load .env (if present) and export only the variables it defines.
ifneq (,$(wildcard .env))
include .env
export $(shell sed -n 's/^\([A-Za-z_][A-Za-z0-9_]*\)=.*/\1/p' .env)
endif

API_DIR := api
WEB_DIR := web
# Dev tools are pinned per tool in api/tools/<tool>.mod.
go_tool = cd $(API_DIR) && go tool -modfile=tools/$(1).mod $(1)

# Docker-only variants: the same Go (from go.mod) and Node as CI, with the
# repo mounted and caches in named volumes. The API tests reach the host's
# Docker through its socket so testcontainers can start Postgres, and reach
# its published ports through the host network on Linux (host firewalls
# often block the bridge gateway) or host.docker.internal on Docker Desktop.
GO_IMAGE := golang:$(shell sed -n 's/^go //p' $(API_DIR)/go.mod)
NODE_IMAGE := node:24
ifeq ($(shell uname -s),Linux)
docker_host_access := --network host -e TESTCONTAINERS_HOST_OVERRIDE=localhost
else
docker_host_access := -e TESTCONTAINERS_HOST_OVERRIDE=host.docker.internal
endif
docker_go = docker run --rm -v "$(CURDIR)":/repo -w /repo/$(API_DIR) \
	-v product-catalog-go-mod:/go/pkg/mod -v product-catalog-go-cache:/root/.cache \
	-v /var/run/docker.sock:/var/run/docker.sock $(docker_host_access) \
	-e GOFLAGS=-buildvcs=false \
	$(GO_IMAGE) sh -c '$(1)'
docker_node = docker run --rm -v "$(CURDIR)":/repo -w /repo/$(WEB_DIR) \
	-v product-catalog-node-modules:/repo/$(WEB_DIR)/node_modules \
	-v product-catalog-node-cache:/root/.cache -v product-catalog-pnpm-store:/root/.local/share/pnpm \
	-e COREPACK_ENABLE_DOWNLOAD_PROMPT=0 \
	$(NODE_IMAGE) sh -c 'corepack enable && pnpm install --frozen-lockfile --silent --store-dir /root/.local/share/pnpm/store && $(1)'

.DEFAULT_GOAL := help
.PHONY: help up down db dev-api dev-web seed test test-unit test-api test-web \
	test-docker lint lint-api lint-web lint-docker fmt gen-types sqlc sync-openapi migrate-new

help: ## List the available commands
	@awk 'BEGIN {FS = ":.*## "} \
		/^##@/ {printf "\n%s\n", substr($$0, 5)} \
		/^[a-zA-Z_-]+:.*## / {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

##@ Environment
up: ## Build and start db, api and web (web on http://localhost:3000)
	docker compose up -d --build --wait

down: ## Stop the stack
	docker compose down

db: ## Start only Postgres
	docker compose up -d --wait db

dev-api: ## Run the API locally with go run (needs `make db`)
	cd $(API_DIR) && go run ./cmd/api

dev-web: ## Run the web dev server (Vite)
	pnpm -C $(WEB_DIR) dev

seed: ## Load 20 sample products with the API image (no Go needed; reruns create no duplicates)
	docker compose run --rm api seed

##@ Tests
test: test-api test-web ## Run every test

test-unit: ## Fast TDD loop: backend with -short + frontend
	cd $(API_DIR) && go test -race -short ./...
	pnpm -C $(WEB_DIR) test

test-api: ## Backend tests, including integration (needs Docker)
	cd $(API_DIR) && go test -race ./...

test-web: ## Frontend tests
	pnpm -C $(WEB_DIR) test

test-docker: ## Every test with Docker only (no local Go or Node)
	$(call docker_go,go test -race ./...)
	$(call docker_node,pnpm test)

##@ Quality
lint: lint-api lint-web ## Lint, check formatting and typecheck both sides

lint-api: ## golangci-lint (linters + formatting check)
	$(call go_tool,golangci-lint) run ./...

lint-web: ## ESLint, Prettier check and typecheck
	pnpm -C $(WEB_DIR) lint
	pnpm -C $(WEB_DIR) format:check
	pnpm -C $(WEB_DIR) typecheck

lint-docker: ## lint-api and lint-web with Docker only (no local Go or Node)
	$(call docker_go,go tool -modfile=tools/golangci-lint.mod golangci-lint run ./...)
	$(call docker_node,pnpm lint && pnpm format:check && pnpm typecheck)

fmt: ## Format both sides
	$(call go_tool,golangci-lint) fmt
	pnpm -C $(WEB_DIR) format

##@ Generation
gen-types: ## Regenerate web/src/api/schema.d.ts from openapi.yaml
	pnpm -C $(WEB_DIR) gen:api

sqlc: ## Regenerate api/internal/repository/db from migrations and queries
	$(call go_tool,sqlc) generate

sync-openapi: ## Copy openapi.yaml into api/internal/apidocs (embedded by /docs)
	mkdir -p $(API_DIR)/internal/apidocs
	cp openapi.yaml $(API_DIR)/internal/apidocs/openapi.yaml

migrate-new: ## Create a goose migration: make migrate-new name=create_products
	@test -n "$(name)" || { echo "usage: make migrate-new name=<name>" >&2; exit 1; }
	mkdir -p $(API_DIR)/migrations
	$(call go_tool,goose) -dir migrations create $(name) sql
