# StuHelper Server Makefile

.PHONY: help tidy fmt lint test test-coverage build run clean security ci

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
	@echo "CI/CD:"
	@echo "  ci            - run all CI checks (lint, test, build)"
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

ci: lint test build
	@echo "CI checks passed!"
