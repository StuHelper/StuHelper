#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf '[registry-release-images][error] %s\n' "$*" >&2
  exit 1
}

require_nonempty() {
  local name="$1"
  local value="${2:-}"
  [[ -n "${value}" ]] || fail "${name} is required"
}

resolve_image() {
  local repository="$1"
  local tagged_ref="${repository}:${TARGET_SHA}"
  local digest

  [[ "${repository}" =~ ^[A-Za-z0-9._:/-]+$ ]] ||
    fail "invalid image repository: ${repository}"
  digest="$(
    docker buildx imagetools inspect "${tagged_ref}" |
      awk '$1 == "Digest:" {print $2; exit}'
  )"
  [[ "${digest}" =~ ^sha256:[0-9a-f]{64}$ ]] ||
    fail "unable to resolve an immutable digest for ${tagged_ref}"
  printf '%s@%s' "${repository}" "${digest}"
}

output_file="${1:-}"
require_nonempty output_file "${output_file}"
require_nonempty TARGET_SHA "${TARGET_SHA:-}"
require_nonempty BACKEND_IMAGE "${BACKEND_IMAGE:-}"
require_nonempty FRONTEND_IMAGE "${FRONTEND_IMAGE:-}"
require_nonempty ADMIN_IMAGE "${ADMIN_IMAGE:-}"
command -v docker >/dev/null 2>&1 || fail "docker is required"

[[ "${TARGET_SHA}" =~ ^[0-9a-f]{40}$ ]] ||
  fail "TARGET_SHA must be a full lowercase 40-character Git commit SHA"

backend_ref="$(resolve_image "${BACKEND_IMAGE}")"
frontend_ref="$(resolve_image "${FRONTEND_IMAGE}")"
admin_ref="$(resolve_image "${ADMIN_IMAGE}")"

output_dir="$(dirname "${output_file}")"
output_name="$(basename "${output_file}")"
mkdir -p "${output_dir}"
tmp_file="$(mktemp "${output_dir}/.${output_name}.XXXXXX")"
cleanup() {
  rm -f "${tmp_file}"
}
trap cleanup EXIT

{
  printf 'ROLLBACK_BACKEND_IMAGE_REF=%s\n' "${backend_ref}"
  printf 'ROLLBACK_FRONTEND_IMAGE_REF=%s\n' "${frontend_ref}"
  printf 'ROLLBACK_ADMIN_IMAGE_REF=%s\n' "${admin_ref}"
} >"${tmp_file}"
chmod 600 "${tmp_file}"
mv "${tmp_file}" "${output_file}"
trap - EXIT

printf '[registry-release-images] resolved three immutable image references\n'
