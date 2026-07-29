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

  if [[ -n "${GITHUB_BASE_REF:-}" && -n "${GITHUB_SHA:-}" ]] &&
    git rev-parse --verify "origin/${GITHUB_BASE_REF}" >/dev/null 2>&1; then
    printf '%s..%s\n' "origin/${GITHUB_BASE_REF}" "${GITHUB_SHA}"
    return
  fi

  if [[ -n "${GITHUB_EVENT_BEFORE:-}" && -n "${GITHUB_SHA:-}" ]] &&
    ! is_zero_sha "${GITHUB_EVENT_BEFORE}"; then
    printf '%s..%s\n' "${GITHUB_EVENT_BEFORE}" "${GITHUB_SHA}"
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
gitleaks git "${source_path}" \
  --log-opts "${log_opts}" \
  --gitleaks-ignore-path "${source_path%/}/.gitleaksignore" \
  --platform github \
  --redact=100 \
  --no-banner \
  --verbose \
  --exit-code 1 \
  "$@"
