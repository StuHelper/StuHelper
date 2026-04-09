#!/bin/sh
set -eu

ADMIN_PAT_FILE="/zitadel/bootstrap/admin.pat"
LOGIN_PAT_FILE="/zitadel/bootstrap/login-client.pat"
MASTERKEY_FILE="${ZITADEL_MASTERKEY_FILE:-/run/secrets/zitadel_masterkey}"

if [[ ! -r "${MASTERKEY_FILE}" ]]; then
  echo "[zitadel-init] masterkey file is missing: ${MASTERKEY_FILE}" >&2
  exit 1
fi

export ZITADEL_MASTERKEY="$(tr -d '\r\n' < "${MASTERKEY_FILE}")"
if [ -z "${ZITADEL_MASTERKEY}" ]; then
  echo "[zitadel-init] masterkey file is empty: ${MASTERKEY_FILE}" >&2
  exit 1
fi

if [ -s "${ADMIN_PAT_FILE}" ] && [ -s "${LOGIN_PAT_FILE}" ]; then
  echo "[zitadel-init] bootstrap PAT files already exist, skipping init"
  exit 0
fi

/app/zitadel start-from-init &
pid="$!"

cleanup() {
  if kill -0 "${pid}" 2>/dev/null; then
    kill "${pid}" 2>/dev/null || true
    wait "${pid}" 2>/dev/null || true
  fi
}

trap cleanup EXIT INT TERM

attempt=0
max_attempts=180
while [ "${attempt}" -lt "${max_attempts}" ]; do
  if [ -s "${ADMIN_PAT_FILE}" ] && [ -s "${LOGIN_PAT_FILE}" ]; then
    if wget -q -T 5 -O- http://127.0.0.1:8080/debug/healthz >/dev/null 2>&1; then
      echo "[zitadel-init] bootstrap completed"
      exit 0
    fi
  fi

  if ! kill -0 "${pid}" 2>/dev/null; then
    wait "${pid}"
    echo "[zitadel-init] start-from-init exited before bootstrap completed" >&2
    exit 1
  fi

  attempt=$((attempt + 1))
  sleep 2
done

echo "[zitadel-init] timed out waiting for bootstrap PAT files" >&2
exit 1
