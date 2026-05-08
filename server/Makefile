# StuHelper Server Makefile

.PHONY: help tidy fmt lint test test-coverage build run clean security ci check-casdoor-boundary \
       bundle-spec lint-spec generate generate-go generate-ts check-drift check-drift-go check-drift-ts check-drift-all \
       check-drift-capabilities check-doc-sync check-coverage-threshold \
       migrate-up migrate-down-one migrate-status migrate-verify seed-dev

# 默认目标
.DEFAULT_GOAL := help

# 变量
BINARY_NAME := stuhelper
BUILD_DIR := bin
COVERAGE_FILE := coverage.out
REDOCLY_VERSION := 2.21.1
REDOCLY := npx --yes @redocly/cli@$(REDOCLY_VERSION)
MIGRATE_VERSION := v4.18.3
MIGRATE := go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION)
MIGRATE_RUNTIME := go run ./cmd/migrate-runtime

define require_database_url
	@test -n "$(DATABASE_URL)" || (echo "ERROR: DATABASE_URL is required" && exit 1)
endef

help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Development:"
	@echo "  tidy          - go mod tidy"
	@echo "  fmt           - format code with gofmt and goimports"
	@echo "  lint          - run golangci-lint"
	@echo "  check-casdoor-boundary - enforce Casdoor/OpenFGA SDK import boundaries"
	@echo "  test          - run tests"
	@echo "  test-coverage - run tests with coverage"
	@echo "  build         - build binary"
	@echo "  run           - run application"
	@echo ""
	@echo "OpenAPI:"
	@echo "  bundle-spec   - bundle multi-file OpenAPI spec into single YAML"
	@echo "  lint-spec     - validate OpenAPI spec with redocly lint"
	@echo "  generate-go   - regenerate Go types and server interface"
	@echo "  generate-ts   - regenerate frontend TypeScript types"
	@echo "  generate      - bundle spec + regenerate all code (Go + TS)"
	@echo "  check-doc-sync - verify docs/reference/api-overview.md prefixes still cover bundled OpenAPI paths"
	@echo "  check-coverage-threshold - enforce minimum coverage for key packages"
	@echo ""
	@echo "CI/CD:"
	@echo "  ci            - run all CI checks (lint, test, build, drift check)"
	@echo "  security      - run security scan with gosec"
	@echo "  clean         - remove build artifacts"
	@echo ""
	@echo "Database:"
	@echo "  migrate-up       - apply current SQL baseline (requires DATABASE_URL)"
	@echo "  migrate-down-one - drop current SQL baseline (requires DATABASE_URL; local only)"
	@echo "  migrate-status   - show current migration version (requires DATABASE_URL)"
	@echo "  migrate-verify   - apply -> drop -> re-apply baseline (requires DATABASE_URL)"
	@echo "  seed-dev         - load development-only seed data (requires DATABASE_URL + psql)"

tidy:
	go mod tidy

fmt:
	gofmt -w .
	@command -v goimports >/dev/null 2>&1 && goimports -w -local git.stuhelper.com/StuHelper/StuHelper . || true

check-casdoor-boundary:
	bash scripts/check-casdoor-boundary.sh

lint: check-casdoor-boundary
	@command -v golangci-lint >/dev/null 2>&1 || { echo "ERROR: golangci-lint not found. Install: https://golangci-lint.run/welcome/install/"; exit 1; }
	golangci-lint run --timeout 5m

test:
	go test -race ./...

test-coverage:
	go test -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	go tool cover -func=$(COVERAGE_FILE)

build:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/stuhelper

run:
	go run ./cmd/stuhelper

security:
	go run github.com/securego/gosec/v2/cmd/gosec@v2.22.4 -exclude-generated ./...

clean:
	rm -rf $(BUILD_DIR)
	rm -f $(COVERAGE_FILE)

ci: lint test security build check-drift
	@echo "CI checks passed!"

# ── OpenAPI ──

bundle-spec:
	$(REDOCLY) bundle --config redocly.yaml api/openapi.yaml -o api/openapi.bundled.yaml

lint-spec:
	$(REDOCLY) lint --config redocly.yaml api/openapi.yaml

generate-go: bundle-spec
	go generate ./internal/api/gen/...

generate-ts: bundle-spec
	cd ../clients && npx --yes pnpm@10.32.1 run api:generate

generate: generate-go generate-ts
	@echo "All code generation complete."

# ── CI Drift Detection ──

check-drift: check-drift-all

check-drift-go: generate-go
	@git diff --exit-code -- internal/api/gen/ || (echo "ERROR: Generated Go code is out of sync with OpenAPI spec. Run 'make generate-go' and commit." && exit 1)

check-drift-ts: bundle-spec
	cd ../clients && npx --yes pnpm@10.32.1 run check:api-drift

check-drift-capabilities:
	cd ../clients && npx --yes pnpm@10.32.1 run check:capabilities-drift

check-drift-all: check-drift-go check-drift-ts check-drift-capabilities

check-doc-sync: bundle-spec
	node --test scripts/check-api-overview-sync.test.mjs
	node scripts/check-api-overview-sync.mjs

check-coverage-threshold:
	bash scripts/check-coverage-threshold.sh

# ── Database Baseline ──

migrate-up:
	$(call require_database_url)
	$(MIGRATE_RUNTIME)

migrate-down-one:
	$(call require_database_url)
	$(MIGRATE) -path migrations -database "$(DATABASE_URL)" down 1

migrate-status:
	$(call require_database_url)
	$(MIGRATE) -path migrations -database "$(DATABASE_URL)" version

migrate-verify:
	$(call require_database_url)
	$(MIGRATE_RUNTIME)
	$(MIGRATE) -path migrations -database "$(DATABASE_URL)" down 1
	$(MIGRATE_RUNTIME)

seed-dev:
	$(call require_database_url)
	@command -v psql >/dev/null 2>&1 || (echo "ERROR: psql is required for seed-dev" && exit 1)
	psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -f scripts/seed.sql
