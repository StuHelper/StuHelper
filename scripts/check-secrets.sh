#!/usr/bin/env bash
set -euo pipefail

source_path="${SECRET_SCAN_SOURCE:-.}"

is_zero_sha() {
  [[ "$1" =~ ^0+$ ]]
}

resolve_log_opts() {
  if [[ -n "${SECRET_SCAN_LOG_OPTS:-}" ]]; then
    printf '%s\n' "${SECRET_SCAN_LOG_OPTS}"
    return
  fi

  if [[ -n "${CI_MERGE_REQUEST_DIFF_BASE_SHA:-}" && -n "${CI_COMMIT_SHA:-}" ]]; then
    printf '%s..%s\n' "${CI_MERGE_REQUEST_DIFF_BASE_SHA}" "${CI_COMMIT_SHA}"
    return
  fi

  if [[ -n "${CI_COMMIT_BEFORE_SHA:-}" && -n "${CI_COMMIT_SHA:-}" ]] && ! is_zero_sha "${CI_COMMIT_BEFORE_SHA}"; then
    printf '%s..%s\n' "${CI_COMMIT_BEFORE_SHA}" "${CI_COMMIT_SHA}"
    return
  fi

  if git rev-parse --verify HEAD~1 >/dev/null 2>&1; then
    printf 'HEAD~1..HEAD\n'
    return
  fi

  printf 'HEAD\n'
}

if ! command -v gitleaks >/dev/null 2>&1; then
  echo "gitleaks is required for secret scanning" >&2
  exit 127
fi

log_opts="$(resolve_log_opts)"
echo "Running gitleaks secret scan on ${source_path} with git log opts: ${log_opts}"
gitleaks detect \
  --source "${source_path}" \
  --log-opts "${log_opts}" \
  --redact \
  --verbose \
  --exit-code 1 \
  "$@"
