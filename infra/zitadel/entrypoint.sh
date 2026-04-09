#!/bin/sh
set -eu

MASTERKEY_FILE="${ZITADEL_MASTERKEY_FILE:-/run/secrets/zitadel_masterkey}"

if [ ! -r "${MASTERKEY_FILE}" ]; then
  echo "[zitadel-api] masterkey file is missing: ${MASTERKEY_FILE}" >&2
  exit 1
fi

export ZITADEL_MASTERKEY="$(tr -d '\r\n' < "${MASTERKEY_FILE}")"
[ -n "${ZITADEL_MASTERKEY}" ] || {
  echo "[zitadel-api] masterkey file is empty: ${MASTERKEY_FILE}" >&2
  exit 1
}

exec /app/zitadel start "$@"
