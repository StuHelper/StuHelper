#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
SERVER_DOCKERFILE="${REPO_ROOT}/server/Dockerfile"
SERVER_DEV_DOCKERFILE="${REPO_ROOT}/server/Dockerfile.dev"
WEB_DOCKERFILE="${REPO_ROOT}/clients/web/Dockerfile"
ADMIN_DOCKERFILE="${REPO_ROOT}/clients/admin/scripts/deploy/Dockerfile"

fail() {
  printf '[dockerfile-supply-chain-contract][error] %s\n' "$*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  grep -Eq -- "${pattern}" "${file}" ||
    fail "expected ${file} to contain pattern: ${pattern}"
}

for dockerfile in \
  "${SERVER_DOCKERFILE}" \
  "${SERVER_DEV_DOCKERFILE}" \
  "${WEB_DOCKERFILE}" \
  "${ADMIN_DOCKERFILE}"; do
  while IFS= read -r image_arg; do
    [[ "${image_arg}" =~ @sha256:[0-9a-f]{64}$ ]] ||
      fail "${dockerfile} contains an unpinned image argument: ${image_arg}"
  done < <(sed -nE 's/^ARG [A-Z_]*IMAGE=(.+)$/\1/p' "${dockerfile}")

  if grep -Eq '^FROM[[:space:]]+[^$].*(:latest|:stable|:main|:master)([[:space:]]|$)' "${dockerfile}"; then
    fail "${dockerfile} contains a mutable base image"
  fi
done

assert_contains "${SERVER_DOCKERFILE}" '^ARG GO_IMAGE=golang:1\.26\.5-alpine3\.24@sha256:[0-9a-f]{64}$'
assert_contains "${SERVER_DOCKERFILE}" '^ARG RUNTIME_IMAGE=alpine:3\.24@sha256:[0-9a-f]{64}$'
assert_contains "${SERVER_DEV_DOCKERFILE}" 'air@v1\.67\.3'
assert_contains "${WEB_DOCKERFILE}" '^ARG NODE_IMAGE=node:24\.18\.0-alpine@sha256:[0-9a-f]{64}$'
assert_contains "${WEB_DOCKERFILE}" '^ARG NGINX_IMAGE=nginx:1\.30\.4-alpine@sha256:[0-9a-f]{64}$'
assert_contains "${ADMIN_DOCKERFILE}" '^ARG NODE_IMAGE=node:24\.18\.0-bookworm-slim@sha256:[0-9a-f]{64}$'
assert_contains "${ADMIN_DOCKERFILE}" '^ARG NGINX_IMAGE=nginx:1\.30\.4-alpine@sha256:[0-9a-f]{64}$'
assert_contains "${ADMIN_DOCKERFILE}" '^ENV TURBO_TELEMETRY_DISABLED=1$'
assert_contains "${ADMIN_DOCKERFILE}" '^USER nginx$'

printf '[dockerfile-supply-chain-contract] all assertions passed\n'
