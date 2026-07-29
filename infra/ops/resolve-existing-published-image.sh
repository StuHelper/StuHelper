#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf '[existing-published-image][error] %s\n' "$*" >&2
  exit 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

require_cmd docker
require_cmd gh
require_cmd jq

[[ "${COMMIT_SHA:-}" =~ ^[0-9a-f]{40}$ ]] ||
  fail "COMMIT_SHA must be a full lowercase 40-character Git commit SHA"
[[ "${IMAGE_NAME:-}" =~ ^ghcr\.io/[a-z0-9][a-z0-9._-]*/[a-z0-9][a-z0-9._-]*$ ]] ||
  fail "IMAGE_NAME must be a canonical lowercase GHCR image name"
[[ "${GITHUB_REPOSITORY:-}" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]] ||
  fail "GITHUB_REPOSITORY must use owner/repository syntax"
[[ -n "${GITHUB_OUTPUT:-}" ]] || fail "GITHUB_OUTPUT is required"
gh attestation verify --help >/dev/null 2>&1 ||
  fail "gh CLI must support artifact attestation verification"

immutable_ref="${IMAGE_NAME}:${COMMIT_SHA}"
inspect_error="$(mktemp)"
verify_error="$(mktemp)"
trap 'rm -f "${inspect_error}" "${verify_error}"' EXIT

if ! manifest="$(
  docker buildx imagetools inspect \
    "${immutable_ref}" \
    --format '{{json .Manifest}}' \
    2>"${inspect_error}"
)"; then
  if grep -Eqi 'manifest unknown|(^|[[:space:]:])not found([[:space:]]|$)' "${inspect_error}"; then
    printf 'found=false\n' >>"${GITHUB_OUTPUT}"
    exit 0
  fi
  cat "${inspect_error}" >&2
  exit 1
fi

digest="$(jq -er '.digest' <<<"${manifest}")" ||
  fail "existing image did not return a manifest digest"
[[ "${digest}" =~ ^sha256:[0-9a-f]{64}$ ]] ||
  fail "existing image returned an invalid manifest digest"

trusted=false
for source_ref in refs/heads/develop refs/heads/main; do
  : >"${verify_error}"
  if gh attestation verify \
    "oci://${IMAGE_NAME}@${digest}" \
    --repo "${GITHUB_REPOSITORY}" \
    --signer-workflow "${GITHUB_REPOSITORY}/.github/workflows/publish-images.yml" \
    --source-digest "${COMMIT_SHA}" \
    --source-ref "${source_ref}" \
    --deny-self-hosted-runners >/dev/null 2>"${verify_error}"; then
    trusted=true
    break
  fi
done
if [[ "${trusted}" != "true" ]]; then
  cat "${verify_error}" >&2
  fail "immutable tag already exists without trusted provenance"
fi

printf 'found=true\n' >>"${GITHUB_OUTPUT}"
printf 'digest=%s\n' "${digest}" >>"${GITHUB_OUTPUT}"
