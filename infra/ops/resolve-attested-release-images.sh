#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf '[release-images][error] %s\n' "$*" >&2
  exit 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

require_cmd docker
require_cmd gh
require_cmd jq

[[ "${TARGET_SHA:-}" =~ ^[0-9a-f]{40}$ ]] ||
  fail "TARGET_SHA must be a full lowercase 40-character Git commit SHA"
[[ "${DEPLOY_ENVIRONMENT:-}" == "staging" || "${DEPLOY_ENVIRONMENT:-}" == "production" ]] ||
  fail "DEPLOY_ENVIRONMENT must be staging or production"
[[ "${GITHUB_REPOSITORY:-}" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]] ||
  fail "GITHUB_REPOSITORY must use owner/repository syntax"
[[ "${IMAGE_NAMESPACE:-}" =~ ^ghcr\.io/[a-z0-9][a-z0-9._-]*$ ]] ||
  fail "IMAGE_NAMESPACE must be a lowercase GHCR namespace"
[[ -n "${GITHUB_OUTPUT:-}" ]] || fail "GITHUB_OUTPUT is required"

signer_workflow="${GITHUB_REPOSITORY}/.github/workflows/publish-images.yml"
source_refs=("refs/heads/main")
if [[ "${DEPLOY_ENVIRONMENT}" == "staging" ]]; then
  source_refs=("refs/heads/develop" "refs/heads/main")
fi

output_file="$(mktemp)"
trap 'rm -f "${output_file}"' EXIT

for component in backend frontend admin; do
  tagged_ref="${IMAGE_NAMESPACE}/${component}:${TARGET_SHA}"
  manifest="$(
    docker buildx imagetools inspect \
      "${tagged_ref}" \
      --format '{{json .Manifest}}'
  )"
  digest="$(jq -er '.digest' <<<"${manifest}")" ||
    fail "unable to resolve ${component} manifest digest"
  [[ "${digest}" =~ ^sha256:[0-9a-f]{64}$ ]] ||
    fail "${component} resolved to an invalid manifest digest"

  digest_ref="${IMAGE_NAMESPACE}/${component}@${digest}"
  verified=false
  for source_ref in "${source_refs[@]}"; do
    if gh attestation verify \
      "oci://${digest_ref}" \
      --repo "${GITHUB_REPOSITORY}" \
      --signer-workflow "${signer_workflow}" \
      --source-digest "${TARGET_SHA}" \
      --source-ref "${source_ref}" \
      --deny-self-hosted-runners >/dev/null; then
      verified=true
      break
    fi
  done
  [[ "${verified}" == "true" ]] ||
    fail "${component} has no trusted provenance for ${TARGET_SHA}"

  printf '%s_image_ref=%s\n' "${component}" "${digest_ref}" >>"${output_file}"
done

cat "${output_file}" >>"${GITHUB_OUTPUT}"
printf '[release-images] three digest-pinned images verified\n'
