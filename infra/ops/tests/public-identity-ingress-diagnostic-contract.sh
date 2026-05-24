#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
DIAG_SCRIPT="${REPO_ROOT}/infra/ops/public-identity-ingress-diagnostic.sh"

fail() {
  echo "[public-identity-ingress-diagnostic-contract][error] $*" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Eq -- "${pattern}" "${file}"; then
    fail "expected ${file} to contain pattern: ${pattern}"
  fi
}

assert_json() {
  local file="$1"
  local jq_filter="$2"
  if ! jq -e "${jq_filter}" "${file}" >/dev/null; then
    cat "${file}" >&2
    fail "expected ${file} to satisfy jq filter: ${jq_filter}"
  fi
}

[[ -f "${DIAG_SCRIPT}" ]] || fail "missing diagnostic script: ${DIAG_SCRIPT}"
[[ -x "${DIAG_SCRIPT}" ]] || fail "diagnostic script must be executable: ${DIAG_SCRIPT}"

bash -n "${DIAG_SCRIPT}"

assert_contains "${DIAG_SCRIPT}" 'PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_FILE'
assert_contains "${DIAG_SCRIPT}" 'PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_STRICT'
assert_contains "${DIAG_SCRIPT}" 'PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_PUBLIC_DNS_ENABLED'
assert_contains "${DIAG_SCRIPT}" 'PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_USE_ENV_TARGETS'
assert_contains "${DIAG_SCRIPT}" 'public-identity-ingress-diagnostic\.json'
assert_contains "${DIAG_SCRIPT}" 'openssl.*s_client'
assert_contains "${DIAG_SCRIPT}" '-servername'
assert_contains "${DIAG_SCRIPT}" '-verify_hostname'
assert_contains "${DIAG_SCRIPT}" 'SSL_ERROR_SYSCALL|ssl_error_syscall'
assert_contains "${DIAG_SCRIPT}" 'dns_non_public_address'
assert_contains "${DIAG_SCRIPT}" 'dns_resolution_failed'
assert_contains "${DIAG_SCRIPT}" 'public_dns_nxdomain'
assert_contains "${DIAG_SCRIPT}" 'public_dns_non_public_address'
assert_contains "${DIAG_SCRIPT}" 'casdoor_well_known_served_by_spa'
assert_contains "${DIAG_SCRIPT}" 'identity_well_known_not_proxied'
assert_contains "${DIAG_SCRIPT}" 'identity_oauth_as_metadata_not_proxied'
assert_contains "${DIAG_SCRIPT}" 'casdoor_jwks_not_proxied'
assert_contains "${DIAG_SCRIPT}" '/.well-known/openid-configuration'
assert_contains "${DIAG_SCRIPT}" '/.well-known/oauth-authorization-server'
assert_contains "${DIAG_SCRIPT}" '/.well-known/jwks.json'
assert_contains "${DIAG_SCRIPT}" 'casdoorJWKS'

tmpdir="$(mktemp -d)"
cleanup() {
  rm -rf "${tmpdir}"
}
trap cleanup EXIT

fake_bin="${tmpdir}/bin"
mkdir -p "${fake_bin}"

cat >"${fake_bin}/openssl" <<'OPENSSL'
#!/usr/bin/env bash
set -euo pipefail
case "${PUBLIC_IDENTITY_DIAG_FAKE_TLS_MODE:-ok}" in
  ok)
    printf '%s\n' 'Protocol version: TLSv1.3'
    printf '%s\n' 'Verification: OK'
    exit 0
    ;;
  syscall)
    printf '%s\n' 'error:0A000126:SSL routines::unexpected eof while reading' >&2
    printf '%s\n' 'curl: (35) OpenSSL SSL_connect: SSL_ERROR_SYSCALL in connection to localhost:443' >&2
    exit 1
    ;;
  *)
    printf 'unknown fake TLS mode\n' >&2
    exit 2
    ;;
esac
OPENSSL
chmod +x "${fake_bin}/openssl"

cat >"${fake_bin}/curl" <<'CURL'
#!/usr/bin/env bash
set -euo pipefail

output_file=""
headers_file=""
write_out=""
url=""
args=("$@")
index=0
while [[ "${index}" -lt "${#args[@]}" ]]; do
  case "${args[${index}]}" in
    -o)
      index=$((index + 1))
      output_file="${args[${index}]}"
      ;;
    -D)
      index=$((index + 1))
      headers_file="${args[${index}]}"
      ;;
    -w)
      index=$((index + 1))
      write_out="${args[${index}]}"
      ;;
    http://* | https://*)
      url="${args[${index}]}"
      ;;
  esac
  index=$((index + 1))
done

status=200
content_type="application/json"
body='{}'

if [[ "${PUBLIC_IDENTITY_DIAG_FAKE_CURL_MODE:-ok}" == "tls_error" ]]; then
  printf '%s\n' 'curl: (35) OpenSSL SSL_connect: SSL_ERROR_SYSCALL in connection to localhost:443' >&2
  exit 35
fi

if [[ "${url}" == https://dns.google/resolve* ]]; then
  content_type="application/dns-json"
  case "${url}" in
    *name=missing.example.com*)
      body='{"Status":3}'
      ;;
    *name=private.example.com*type=A*)
      body='{"Status":0,"Answer":[{"name":"private.example.com.","type":1,"TTL":60,"data":"10.0.0.10"}]}'
      ;;
    *name=private.example.com*type=AAAA*)
      body='{"Status":0,"Answer":[]}'
      ;;
    *type=AAAA*)
      body='{"Status":0,"Answer":[]}'
      ;;
    *)
      body='{"Status":0,"Answer":[{"name":"example.com.","type":1,"TTL":60,"data":"93.184.216.34"}]}'
      ;;
  esac
  printf '%s\n' "${body}"
  exit 0
fi

case "${url}" in
  */health/ready)
    body='{"status":"ready"}'
    ;;
  https://id.stuhelper.com/.well-known/openid-configuration)
    body='{"issuer":"https://id.stuhelper.com","authorization_endpoint":"https://id.stuhelper.com/oauth2/authorize","token_endpoint":"https://id.stuhelper.com/oauth2/token","jwks_uri":"https://id.stuhelper.com/.well-known/jwks.json"}'
    ;;
  https://id.stuhelper.com/.well-known/oauth-authorization-server)
    body='{"issuer":"https://id.stuhelper.com","authorization_endpoint":"https://id.stuhelper.com/oauth2/authorize","token_endpoint":"https://id.stuhelper.com/oauth2/token","jwks_uri":"https://id.stuhelper.com/.well-known/jwks.json","revocation_endpoint":"https://id.stuhelper.com/oauth2/revoke","introspection_endpoint":"https://id.stuhelper.com/oauth2/introspect"}'
    ;;
  */id/.well-known/openid-configuration)
    body='{"issuer":"https://localhost/id","authorization_endpoint":"https://localhost/id/oauth2/authorize","token_endpoint":"https://localhost/id/oauth2/token","jwks_uri":"https://localhost/id/.well-known/jwks.json"}'
    ;;
  */id/.well-known/oauth-authorization-server)
    body='{"issuer":"https://localhost/id","authorization_endpoint":"https://localhost/id/oauth2/authorize","token_endpoint":"https://localhost/id/oauth2/token","jwks_uri":"https://localhost/id/.well-known/jwks.json","revocation_endpoint":"https://localhost/id/oauth2/revoke","introspection_endpoint":"https://localhost/id/oauth2/introspect"}'
    ;;
  https://id.stuhelper.com/.well-known/jwks.json | */id/.well-known/jwks.json)
    body='{"keys":[]}'
    ;;
  https://sso.stuhelper.com/.well-known/openid-configuration)
    if [[ "${PUBLIC_IDENTITY_DIAG_FAKE_CURL_MODE:-ok}" == "sso_spa_404" ]]; then
      status=404
      content_type="text/html"
      body='<!doctype html><html><head><title>Casdoor</title></head><body>Casdoor SPA</body></html>'
    else
      body='{"issuer":"https://sso.stuhelper.com","authorization_endpoint":"https://sso.stuhelper.com/login/oauth/authorize","token_endpoint":"https://sso.stuhelper.com/api/login/oauth/access_token","jwks_uri":"https://sso.stuhelper.com/.well-known/jwks"}'
    fi
    ;;
  https://sso.stuhelper.com/.well-known/jwks)
    if [[ "${PUBLIC_IDENTITY_DIAG_FAKE_CURL_MODE:-ok}" == "sso_jwks_404" ]]; then
      status=404
      content_type="text/plain"
      body='not found'
    else
      body='{"keys":[]}'
    fi
    ;;
  */sso/.well-known/openid-configuration)
    if [[ "${PUBLIC_IDENTITY_DIAG_FAKE_CURL_MODE:-ok}" == "sso_spa_404" ]]; then
      status=404
      content_type="text/html"
      body='<!doctype html><html><head><title>Casdoor</title></head><body>Casdoor SPA</body></html>'
    else
      body='{"issuer":"https://localhost/sso","authorization_endpoint":"https://localhost/sso/login/oauth/authorize","token_endpoint":"https://localhost/sso/api/login/oauth/access_token","jwks_uri":"https://localhost/sso/.well-known/jwks"}'
    fi
    ;;
  */sso/.well-known/jwks)
    if [[ "${PUBLIC_IDENTITY_DIAG_FAKE_CURL_MODE:-ok}" == "sso_jwks_404" ]]; then
      status=404
      content_type="text/plain"
      body='not found'
    else
      body='{"keys":[]}'
    fi
    ;;
  *)
    status=404
    content_type="text/plain"
    body='not found'
    ;;
esac

if [[ -n "${output_file}" ]]; then
  printf '%s\n' "${body}" >"${output_file}"
else
  printf '%s\n' "${body}"
fi
if [[ -n "${headers_file}" ]]; then
  {
    printf 'HTTP/2 %s\r\n' "${status}"
    printf 'content-type: %s\r\n' "${content_type}"
    printf 'set-cookie: fake-session=should-not-leak; HttpOnly\r\n'
    printf '\r\n'
  } >"${headers_file}"
fi
if [[ -n "${write_out}" ]]; then
  printf '\n__STUHELPER_CURL_META__:%s\t%s\t0\t127.0.0.1\t2\t0.001\t0.002' "${status}" "${content_type}"
fi
CURL
chmod +x "${fake_bin}/curl"

ok_file="${tmpdir}/ok.json"
PATH="${fake_bin}:${PATH}" \
  WEB_PUBLIC_URL=https://localhost/web \
  IDENTITY_ISSUER=https://localhost/id \
  CASDOOR_ISSUER=https://localhost/sso \
  PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_FILE="${ok_file}" \
  PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_STRICT=true \
  "${DIAG_SCRIPT}" >"${tmpdir}/ok.stdout" 2>"${tmpdir}/ok.stderr"

assert_json "${ok_file}" '.passed == true'
assert_json "${ok_file}" '.summary.failed == 0'
assert_json "${ok_file}" '.endpoints.identityDiscovery.issuer == "https://localhost/id"'
assert_json "${ok_file}" '.endpoints.identityAuthorizationServerMetadata.issuer == "https://localhost/id"'
assert_json "${ok_file}" '.endpoints.casdoorDiscovery.issuer == "https://localhost/sso"'
assert_json "${ok_file}" '.endpoints.casdoorDiscovery.jwksURI == "https://localhost/sso/.well-known/jwks"'
assert_json "${ok_file}" '.endpoints.casdoorJWKS.passed == true'
if grep -Eiq 'secret|token=' "${ok_file}"; then
  fail "diagnostic evidence must not contain secrets or raw token values"
fi
if grep -Eiq 'set-cookie: [^<]' "${ok_file}"; then
  fail "diagnostic evidence must redact Set-Cookie header values"
fi

local_env_file="${tmpdir}/local.env"
cat >"${local_env_file}" <<'ENV'
WEB_PUBLIC_URL=http://localhost:3000
IDENTITY_ISSUER=http://localhost:3000
CASDOOR_ISSUER=http://localhost:8085
ENV

default_targets_file="${tmpdir}/default-targets.json"
PATH="${fake_bin}:${PATH}" \
  ENV_FILE="${local_env_file}" \
  PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_FILE="${default_targets_file}" \
  "${DIAG_SCRIPT}" >"${tmpdir}/default-targets.stdout" 2>"${tmpdir}/default-targets.stderr"

assert_json "${default_targets_file}" '.webPublicURL == "https://stuhelper.com"'
assert_json "${default_targets_file}" '.identityIssuer == "https://id.stuhelper.com"'
assert_json "${default_targets_file}" '.casdoorIssuer == "https://sso.stuhelper.com"'

env_targets_file="${tmpdir}/env-targets.json"
PATH="${fake_bin}:${PATH}" \
  ENV_FILE="${local_env_file}" \
  PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_USE_ENV_TARGETS=true \
  PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_FILE="${env_targets_file}" \
  "${DIAG_SCRIPT}" >"${tmpdir}/env-targets.stdout" 2>"${tmpdir}/env-targets.stderr"

assert_json "${env_targets_file}" '.webPublicURL == "http://localhost:3000"'
assert_json "${env_targets_file}" '.identityIssuer == "http://localhost:3000"'
assert_json "${env_targets_file}" '.casdoorIssuer == "http://localhost:8085"'

non_public_dns_file="${tmpdir}/non-public-dns.json"
PATH="${fake_bin}:${PATH}" \
  WEB_PUBLIC_URL=https://198.18.0.19 \
  IDENTITY_ISSUER=https://localhost/id \
  CASDOOR_ISSUER=https://localhost/sso \
  PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_FILE="${non_public_dns_file}" \
  "${DIAG_SCRIPT}" >"${tmpdir}/non-public-dns.stdout" 2>"${tmpdir}/non-public-dns.stderr"

assert_json "${non_public_dns_file}" '.passed == false'
assert_json "${non_public_dns_file}" '.hosts.web.dns.diagnosis == "dns_non_public_address"'
assert_json "${non_public_dns_file}" '.hosts.web.dns.nonPublicAddresses == ["198.18.0.19"]'
assert_json "${non_public_dns_file}" '.diagnoses[] | select(.target == "hosts.web" and .diagnosis == "dns_non_public_address")'

public_dns_missing_file="${tmpdir}/public-dns-missing.json"
PATH="${fake_bin}:${PATH}" \
  WEB_PUBLIC_URL=https://missing.example.com \
  IDENTITY_ISSUER=https://localhost/id \
  CASDOOR_ISSUER=https://localhost/sso \
  PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_FILE="${public_dns_missing_file}" \
  "${DIAG_SCRIPT}" >"${tmpdir}/public-dns-missing.stdout" 2>"${tmpdir}/public-dns-missing.stderr"

assert_json "${public_dns_missing_file}" '.passed == false'
assert_json "${public_dns_missing_file}" '.hosts.web.publicDNS.provider == "dns.google"'
assert_json "${public_dns_missing_file}" '.hosts.web.publicDNS.diagnosis == "public_dns_nxdomain"'
assert_json "${public_dns_missing_file}" '.diagnoses[] | select(.target == "hosts.web" and .diagnosis == "public_dns_nxdomain")'

public_dns_private_file="${tmpdir}/public-dns-private.json"
PATH="${fake_bin}:${PATH}" \
  WEB_PUBLIC_URL=https://private.example.com \
  IDENTITY_ISSUER=https://localhost/id \
  CASDOOR_ISSUER=https://localhost/sso \
  PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_FILE="${public_dns_private_file}" \
  "${DIAG_SCRIPT}" >"${tmpdir}/public-dns-private.stdout" 2>"${tmpdir}/public-dns-private.stderr"

assert_json "${public_dns_private_file}" '.passed == false'
assert_json "${public_dns_private_file}" '.hosts.web.publicDNS.diagnosis == "public_dns_non_public_address"'
assert_json "${public_dns_private_file}" '.hosts.web.publicDNS.nonPublicAddresses == ["10.0.0.10"]'
assert_json "${public_dns_private_file}" '.diagnoses[] | select(.target == "hosts.web" and .diagnosis == "public_dns_non_public_address")'

sso_spa_file="${tmpdir}/sso-spa.json"
PATH="${fake_bin}:${PATH}" \
  WEB_PUBLIC_URL=https://localhost/web \
  IDENTITY_ISSUER=https://localhost/id \
  CASDOOR_ISSUER=https://localhost/sso \
  PUBLIC_IDENTITY_DIAG_FAKE_CURL_MODE=sso_spa_404 \
  PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_FILE="${sso_spa_file}" \
  "${DIAG_SCRIPT}" >"${tmpdir}/sso-spa.stdout" 2>"${tmpdir}/sso-spa.stderr"

assert_json "${sso_spa_file}" '.passed == false'
assert_json "${sso_spa_file}" '.endpoints.casdoorDiscovery.diagnosis == "casdoor_well_known_served_by_spa"'
assert_json "${sso_spa_file}" '.diagnoses[] | select(.diagnosis == "casdoor_well_known_served_by_spa")'
grep -q 'set-cookie: <redacted>' "${sso_spa_file}" || fail "sso diagnostic should retain only redacted Set-Cookie marker"

sso_jwks_file="${tmpdir}/sso-jwks.json"
PATH="${fake_bin}:${PATH}" \
  WEB_PUBLIC_URL=https://localhost/web \
  IDENTITY_ISSUER=https://localhost/id \
  CASDOOR_ISSUER=https://localhost/sso \
  PUBLIC_IDENTITY_DIAG_FAKE_CURL_MODE=sso_jwks_404 \
  PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_FILE="${sso_jwks_file}" \
  "${DIAG_SCRIPT}" >"${tmpdir}/sso-jwks.stdout" 2>"${tmpdir}/sso-jwks.stderr"

assert_json "${sso_jwks_file}" '.passed == false'
assert_json "${sso_jwks_file}" '.endpoints.casdoorJWKS.diagnosis == "casdoor_jwks_not_proxied"'
assert_json "${sso_jwks_file}" '.diagnoses[] | select(.target == "endpoints.casdoorJWKS" and .diagnosis == "casdoor_jwks_not_proxied")'

if PATH="${fake_bin}:${PATH}" \
  WEB_PUBLIC_URL=https://localhost/web \
  IDENTITY_ISSUER=https://localhost/id \
  CASDOOR_ISSUER=https://localhost/sso \
  PUBLIC_IDENTITY_DIAG_FAKE_TLS_MODE=syscall \
  PUBLIC_IDENTITY_DIAG_FAKE_CURL_MODE=tls_error \
  PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_FILE="${tmpdir}/tls-error.json" \
  PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_STRICT=true \
  "${DIAG_SCRIPT}" >"${tmpdir}/tls-error.stdout" 2>"${tmpdir}/tls-error.stderr"; then
  fail "strict diagnostic should fail when TLS and HTTP checks fail"
fi
assert_json "${tmpdir}/tls-error.json" '.passed == false'
assert_json "${tmpdir}/tls-error.json" '.diagnoses[] | select(.diagnosis == "tls_handshake_failed")'
grep -q 'public identity ingress diagnostic found failing checks' "${tmpdir}/tls-error.stderr" || \
  fail "strict diagnostic did not report the expected failure"

echo "[public-identity-ingress-diagnostic-contract] all assertions passed"
