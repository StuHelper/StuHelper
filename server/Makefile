# StuHelper Server Makefile

.PHONY: help tidy fmt lint test test-coverage build run clean security ci \
       bundle-spec lint-spec generate generate-go generate-ts check-drift

# 默认目标
.DEFAULT_GOAL := help

# 变量
BINARY_NAME := stuhelper
BUILD_DIR := bin
COVERAGE_FILE := coverage.out

help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Development:"
	@echo "  tidy          - go mod tidy"
	@echo "  fmt           - format code with gofmt and goimports"
	@echo "  lint          - run golangci-lint"
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
	@echo ""
	@echo "CI/CD:"
	@echo "  ci            - run all CI checks (lint, test, build, drift check)"
	@echo "  security      - run security scan with gosec"
	@echo "  clean         - remove build artifacts"

tidy:
	go mod tidy

fmt:
	gofmt -w .
	@command -v goimports >/dev/null 2>&1 && goimports -w -local gitea.stuhelper.com/StuHelper/StuHelper . || true

lint:
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
	@command -v gosec >/dev/null 2>&1 || go install github.com/securego/gosec/v2/cmd/gosec@latest
	gosec -exclude-generated ./...

clean:
	rm -rf $(BUILD_DIR)
	rm -f $(COVERAGE_FILE)

ci: lint test security build check-drift
	@echo "CI checks passed!"

# ── OpenAPI ──

bundle-spec:
	@if command -v redocly >/dev/null 2>&1; then \
		redocly bundle api/openapi.yaml -o api/openapi.bundled.yaml; \
	else \
		npx --yes @redocly/cli@latest bundle api/openapi.yaml -o api/openapi.bundled.yaml; \
	fi

lint-spec:
	@if command -v redocly >/dev/null 2>&1; then \
		redocly lint api/openapi.yaml; \
	else \
		npx --yes @redocly/cli@latest lint api/openapi.yaml; \
	fi

generate-go: bundle-spec
	go generate ./internal/api/gen/...

generate-ts: bundle-spec
	cd ../clients && pnpm run api:generate

generate: generate-go generate-ts
	@echo "All code generation complete."

# ── CI Drift Detection ──

check-drift: generate
	@git diff --exit-code -- internal/api/gen/ ../clients/shared/src/types/api.gen.ts || (echo "ERROR: Generated code is out of sync with OpenAPI spec. Run 'make generate' and commit." && exit 1)
