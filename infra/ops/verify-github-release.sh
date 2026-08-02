#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf '[github-release][error] %s\n' "$*" >&2
  exit 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

require_cmd gh
require_cmd jq

target_sha="${TARGET_SHA:-}"
workflow_sha="${WORKFLOW_SHA:-}"
source_ref="${SOURCE_REF:-}"
repository="${GITHUB_REPOSITORY:-}"
environment="${DEPLOY_ENVIRONMENT:-}"
reason="${DEPLOY_REASON:-}"
required_checks="${REQUIRED_RELEASE_CHECKS:-Required,go,javascript-typescript}"
max_attempts="${RELEASE_CHECK_MAX_ATTEMPTS:-20}"
poll_seconds="${RELEASE_CHECK_POLL_SECONDS:-15}"
release_operation="${RELEASE_OPERATION:-}"
promotion_mode="${PRODUCTION_PROMOTION_MODE:-}"

[[ "${target_sha}" =~ ^[0-9a-f]{40}$ ]] ||
  fail "TARGET_SHA must be a full lowercase 40-character Git commit SHA"
[[ "${workflow_sha}" =~ ^[0-9a-f]{40}$ ]] ||
  fail "WORKFLOW_SHA must be a full lowercase 40-character Git commit SHA"
[[ "${repository}" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]] ||
  fail "GITHUB_REPOSITORY must use owner/repository syntax"
[[ "${environment}" == "staging" || "${environment}" == "production" ]] ||
  fail "DEPLOY_ENVIRONMENT must be staging or production"
[[ "${source_ref}" == "refs/heads/main" || "${source_ref}" == "refs/heads/develop" ]] ||
  fail "SOURCE_REF must be refs/heads/main or refs/heads/develop"
if [[ ! "${max_attempts}" =~ ^[1-9][0-9]*$ ]] || ((max_attempts > 40)); then
  fail "RELEASE_CHECK_MAX_ATTEMPTS must be between 1 and 40"
fi
if [[ ! "${poll_seconds}" =~ ^[0-9]+$ ]] || ((poll_seconds > 60)); then
  fail "RELEASE_CHECK_POLL_SECONDS must be between 0 and 60"
fi

case "${release_operation}" in
  publish | forward | rollback) ;;
  *) fail "RELEASE_OPERATION must be publish, forward, or rollback" ;;
esac

case "${release_operation}" in
  publish | rollback)
    [[ "${promotion_mode}" == "not-applicable" ]] ||
      fail "${release_operation} requires PRODUCTION_PROMOTION_MODE=not-applicable"
    ;;
  forward)
    if [[ "${environment}" == "staging" ]]; then
      [[ "${promotion_mode}" == "staging" ]] ||
        fail "staging deployment requires PRODUCTION_PROMOTION_MODE=staging"
    else
      case "${promotion_mode}" in
        after-staging | direct | break-glass) ;;
        *)
          fail "production deployment requires PRODUCTION_PROMOTION_MODE=after-staging, direct, or break-glass"
          ;;
      esac
    fi
    ;;
esac

if ((${#reason} < 12 || ${#reason} > 500)); then
  fail "DEPLOY_REASON must contain 12-500 characters"
fi
if [[ "${reason}" =~ [[:cntrl:]] ]]; then
  fail "DEPLOY_REASON must contain printable characters only"
fi

if [[ "${environment}" == "production" && "${source_ref}" != "refs/heads/main" ]]; then
  fail "production deployments must run from refs/heads/main"
fi
if [[ "${release_operation}" == "forward" && "${promotion_mode}" == "direct" ]]; then
  ((${#reason} >= 24)) ||
    fail "direct production promotion requires at least 24 characters of change context"
  printf '[github-release][warning] direct production promotion selected; no same-SHA staging success is required\n' >&2
fi
if [[ "${release_operation}" == "forward" && "${promotion_mode}" == "break-glass" ]]; then
  ((${#reason} >= 24)) ||
    fail "break-glass production promotion requires at least 24 characters of incident context"
  printf '[github-release][warning] break-glass production promotion selected; no same-SHA staging success is required\n' >&2
fi

# Every release operation executes the controller from the exact live branch
# head. rollback.yml keeps this controller current while selecting a previously
# attested image set through a separate, validated input.
[[ "${target_sha}" == "${workflow_sha}" ]] ||
  fail "release controller target must equal the trusted workflow commit"

branch="${source_ref#refs/heads/}"
branch_sha="$(
  gh api "repos/${repository}/git/ref/heads/${branch}" --jq '.object.sha'
)" || fail "unable to resolve ${source_ref}"
[[ "${branch_sha}" == "${target_sha}" ]] ||
  fail "${source_ref} advanced or no longer points at TARGET_SHA; start a new deployment"

IFS=',' read -r -a check_names <<<"${required_checks}"
((${#check_names[@]} > 0)) || fail "REQUIRED_RELEASE_CHECKS must not be empty"

for attempt in $(seq 1 "${max_attempts}"); do
  payload="$(
    gh api "repos/${repository}/commits/${target_sha}/check-runs?per_page=100"
  )" || fail "unable to read GitHub check runs for ${target_sha}"

  pending=()
  for raw_name in "${check_names[@]}"; do
    check_name="${raw_name#"${raw_name%%[![:space:]]*}"}"
    check_name="${check_name%"${check_name##*[![:space:]]}"}"
    [[ -n "${check_name}" ]] || fail "REQUIRED_RELEASE_CHECKS contains an empty name"

    latest="$(
      jq -cer --arg name "${check_name}" '
        [
          .check_runs[]
          | select(.name == $name and .app.slug == "github-actions")
        ]
        | sort_by(.id)
        | last
      ' <<<"${payload}" 2>/dev/null || true
    )"
    if [[ -z "${latest}" || "${latest}" == "null" ]]; then
      pending+=("${check_name}=missing")
      continue
    fi

    status="$(jq -er '.status' <<<"${latest}")" ||
      fail "check ${check_name} returned no status"
    conclusion="$(jq -r '.conclusion // ""' <<<"${latest}")"
    if [[ "${status}" != "completed" ]]; then
      pending+=("${check_name}=${status}")
      continue
    fi
    [[ "${conclusion}" == "success" ]] ||
      fail "required check ${check_name} concluded ${conclusion:-unknown}"
  done

  if ((${#pending[@]} == 0)); then
    break
  fi

  if ((attempt == max_attempts)); then
    fail "required checks did not become ready: ${pending[*]}"
  fi
  printf '[github-release] waiting for required checks (%s/%s): %s\n' \
    "${attempt}" "${max_attempts}" "${pending[*]}"
  sleep "${poll_seconds}"
done

if [[ "${release_operation}" == "forward" && "${environment}" == "production" && "${promotion_mode}" == "after-staging" ]]; then
  for attempt in $(seq 1 "${max_attempts}"); do
    deployments="$(
      gh api "repos/${repository}/deployments?sha=${target_sha}&environment=staging&per_page=100"
    )" || fail "unable to read staging deployments for ${target_sha}"
    staging_deployment_id="$(
      jq -er 'sort_by(.id) | last | .id' <<<"${deployments}" 2>/dev/null || true
    )"
    staging_state=""
    if [[ -n "${staging_deployment_id}" ]]; then
      deployment_statuses="$(
        gh api "repos/${repository}/deployments/${staging_deployment_id}/statuses?per_page=100"
      )" || fail "unable to read staging deployment status"
      staging_state="$(
        jq -er 'sort_by(.id) | last | .state' <<<"${deployment_statuses}" 2>/dev/null || true
      )"
    fi

    case "${staging_state}" in
      success)
        break
        ;;
      error | failure | inactive)
        fail "latest staging deployment for ${target_sha} is ${staging_state}, not success"
        ;;
    esac
    if ((attempt == max_attempts)); then
      fail "production requires a successful staging deployment of the same commit; latest state is ${staging_state:-missing}"
    fi
    printf '[github-release] waiting for same-SHA staging success (%s/%s): %s\n' \
      "${attempt}" "${max_attempts}" "${staging_state:-missing}"
    sleep "${poll_seconds}"
  done
fi

printf '[github-release] branch head, required checks, and promotion policy verified for %s\n' \
  "${target_sha}"
