#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf '[publish-scanned-image][error] %s\n' "$*" >&2
  exit 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

require_cmd docker
require_cmd jq
require_cmd sleep

[[ "${COMMIT_SHA:-}" =~ ^[0-9a-f]{40}$ ]] ||
  fail "COMMIT_SHA must be a full lowercase 40-character Git commit SHA"
[[ "${IMAGE_NAME:-}" =~ ^ghcr\.io/[a-z0-9][a-z0-9._-]*/[a-z0-9][a-z0-9._-]*$ ]] ||
  fail "IMAGE_NAME must be a canonical lowercase GHCR image name"
[[ -n "${GITHUB_OUTPUT:-}" ]] || fail "GITHUB_OUTPUT is required"

max_attempts="${PUBLISH_MAX_ATTEMPTS:-4}"
base_delay_seconds="${PUBLISH_RETRY_BASE_DELAY_SECONDS:-30}"
[[ "${max_attempts}" =~ ^[1-5]$ ]] ||
  fail "PUBLISH_MAX_ATTEMPTS must be an integer from 1 through 5"
if [[ ! "${base_delay_seconds}" =~ ^[0-9]+$ ]] ||
  ((base_delay_seconds < 1 || base_delay_seconds > 120)); then
  fail "PUBLISH_RETRY_BASE_DELAY_SECONDS must be an integer from 1 through 120"
fi

immutable_ref="${IMAGE_NAME}:${COMMIT_SHA}"
push_log="$(mktemp)"
trap 'rm -f "${push_log}"' EXIT

attempt=1
while true; do
  : >"${push_log}"
  if docker push "${immutable_ref}" 2>&1 | tee "${push_log}"; then
    break
  fi

  if ! grep -Eqi \
    'secondary rate limit|too many requests|HTTP status code 429|unexpected status[^[:digit:]]*429' \
    "${push_log}"; then
    fail "docker push failed with a non-retryable registry error"
  fi
  if (( attempt >= max_attempts )); then
    fail "docker push exhausted ${max_attempts} attempts after registry rate limiting"
  fi

  delay_seconds=$((base_delay_seconds * (1 << (attempt - 1))))
  printf \
    '[publish-scanned-image] registry rate limited attempt %d/%d; retrying in %d seconds\n' \
    "${attempt}" \
    "${max_attempts}" \
    "${delay_seconds}" >&2
  sleep "${delay_seconds}"
  attempt=$((attempt + 1))
done

digest="$(
  docker buildx imagetools inspect \
    "${immutable_ref}" \
    --format '{{json .Manifest}}' |
    jq -er '.digest'
)" || fail "published image did not return a manifest digest"
[[ "${digest}" =~ ^sha256:[0-9a-f]{64}$ ]] ||
  fail "published image returned an invalid manifest digest"

printf 'digest=%s\n' "${digest}" >>"${GITHUB_OUTPUT}"
