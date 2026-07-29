#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
POLICY_FILE="${RUNTIME_IMAGE_POLICY_FILE:-${REPO_ROOT}/infra/security/runtime-images.json}"
VALIDATOR="${REPO_ROOT}/infra/ops/validate-runtime-image-scan.py"
SCAN_TIMEOUT="${RUNTIME_IMAGE_SCAN_TIMEOUT:-10m}"

command -v docker >/dev/null 2>&1 || {
  echo "[runtime-image-scan][error] docker is required" >&2
  exit 1
}
command -v python3 >/dev/null 2>&1 || {
  echo "[runtime-image-scan][error] python3 is required" >&2
  exit 1
}

created_output_dir=false
if [[ -n "${RUNTIME_IMAGE_SCAN_OUTPUT_DIR:-}" ]]; then
  OUTPUT_DIR="${RUNTIME_IMAGE_SCAN_OUTPUT_DIR}"
  mkdir -p "${OUTPUT_DIR}"
else
  OUTPUT_DIR="$(mktemp -d /tmp/stuhelper-runtime-image-scan.XXXXXX)"
  created_output_dir=true
fi
OUTPUT_DIR="$(cd "${OUTPUT_DIR}" && pwd)"

created_cache_dir=false
if [[ -n "${TRIVY_CACHE_DIR:-}" ]]; then
  mkdir -p "${TRIVY_CACHE_DIR}"
else
  TRIVY_CACHE_DIR="$(mktemp -d /tmp/stuhelper-trivy-cache.XXXXXX)"
  created_cache_dir=true
fi
TRIVY_CACHE_DIR="$(cd "${TRIVY_CACHE_DIR}" && pwd)"

declare -a local_scan_images=()
cleanup() {
  local image_ref
  for image_ref in "${local_scan_images[@]}"; do
    docker image rm "${image_ref}" >/dev/null 2>&1 || true
  done
  if [[ "${created_output_dir}" == "true" ]]; then
    case "${OUTPUT_DIR}" in
      /tmp/stuhelper-runtime-image-scan.*)
        rm -rf -- "${OUTPUT_DIR}"
        ;;
      *)
        echo "[runtime-image-scan][error] refusing to remove unexpected output path: ${OUTPUT_DIR}" >&2
        ;;
    esac
  fi
  if [[ "${created_cache_dir}" == "true" ]]; then
    case "${TRIVY_CACHE_DIR}" in
      /tmp/stuhelper-trivy-cache.*)
        rm -rf -- "${TRIVY_CACHE_DIR}"
        ;;
      *)
        echo "[runtime-image-scan][error] refusing to remove unexpected cache path: ${TRIVY_CACHE_DIR}" >&2
        ;;
    esac
  fi
}
trap cleanup EXIT

cache_write_probe="${TRIVY_CACHE_DIR}/.stuhelper-write-probe-${BASHPID}"
if ! (umask 077 && : > "${cache_write_probe}"); then
  echo "[runtime-image-scan][error] Trivy cache is not writable by uid $(id -u): ${TRIVY_CACHE_DIR}" >&2
  exit 1
fi
rm -f -- "${cache_write_probe}"

python3 "${VALIDATOR}" \
  --repo-root "${REPO_ROOT}" \
  --policy "${POLICY_FILE}" \
  --policy-only

trivy_image="$(
  python3 - "${POLICY_FILE}" <<'PY'
import json
import sys
with open(sys.argv[1], encoding="utf-8") as handle:
    print(json.load(handle)["scanner"]["image"])
PY
)"

docker run --rm \
  --user "$(id -u):$(id -g)" \
  --mount "type=bind,src=${TRIVY_CACHE_DIR},dst=/cache" \
  "${trivy_image}" \
  image \
  --quiet \
  --cache-dir /cache \
  --download-db-only

while IFS=$'\t' read -r image_id kind image_ref context dockerfile build_args_json; do
  scan_source=("${image_ref}")
  if [[ "${kind}" == "build" ]]; then
    scan_image_ref="${image_ref}-${BASHPID}"
    local_scan_images+=("${scan_image_ref}")
    build_command=(
      docker build
      --pull=false
      --file "${REPO_ROOT}/${dockerfile}"
      --tag "${scan_image_ref}"
    )
    mapfile -t build_args < <(
      python3 - "${build_args_json}" <<'PY'
import json
import sys
for item in json.loads(sys.argv[1]):
    print(item)
PY
    )
    for build_arg in "${build_args[@]}"; do
      build_command+=(--build-arg "${build_arg}")
    done
    build_command+=("${REPO_ROOT}/${context}")
    echo "[runtime-image-scan] building ${image_id}"
    "${build_command[@]}"
    docker save --output "${OUTPUT_DIR}/${image_id}.tar" "${scan_image_ref}"
    scan_source=(--input "/scan/${image_id}.tar")
  fi

  echo "[runtime-image-scan] scanning ${image_id}"
  docker run --rm \
    --user "$(id -u):$(id -g)" \
    --mount "type=bind,src=${TRIVY_CACHE_DIR},dst=/cache" \
    --mount "type=bind,src=${OUTPUT_DIR},dst=/scan" \
    "${trivy_image}" \
    image \
    --quiet \
    --skip-db-update \
    --cache-dir /cache \
    --timeout "${SCAN_TIMEOUT}" \
    --scanners vuln \
    --severity HIGH,CRITICAL,UNKNOWN \
    --ignore-unfixed=false \
    --format json \
    --output "/scan/${image_id}.json" \
    "${scan_source[@]}"
done < <(
  python3 "${VALIDATOR}" \
    --repo-root "${REPO_ROOT}" \
    --policy "${POLICY_FILE}" \
    --policy-only \
    --print-plan
)

find "${OUTPUT_DIR}" -maxdepth 1 -type f -name '*.tar' -delete

python3 "${VALIDATOR}" \
  --repo-root "${REPO_ROOT}" \
  --policy "${POLICY_FILE}" \
  --scan-dir "${OUTPUT_DIR}"
