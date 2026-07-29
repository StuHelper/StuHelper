ARG NODE_BASE_IMAGE_REF=node:24.18.0-alpine@sha256:a0b9bf06e4e6193cf7a0f58816cc935ff8c2a908f81e6f1a95432d679c54fbfd
FROM ${NODE_BASE_IMAGE_REF}

ARG NPM_VERSION=11.18.0
ARG NPM_TARBALL_SHA512=T67M4L5wNm0cZ7EBLErcEkY1SmzEW/WJ+SADBzsFUY1UdAPfFHXFQtZ6SEXiK0+vzXysCvAsepbMaBTwnrAD+w==
ARG BRACE_EXPANSION_VERSION=5.0.8
ARG BRACE_EXPANSION_TARBALL_SHA512=JZyDyq3D4AUifKTPOB7DELf6XsB3WdPuNxCtob1vFXPsSXhdAiHBWJ/tJ8HAc9aH84BK+5JFZLNkJKx3G9kzQg==

RUN set -eu; \
    npm_tarball="/tmp/npm-${NPM_VERSION}.tgz"; \
    wget -qO "${npm_tarball}" "https://registry.npmjs.org/npm/-/npm-${NPM_VERSION}.tgz"; \
    expected_sha512="$(printf '%s' "${NPM_TARBALL_SHA512}" | base64 -d | od -An -v -tx1 | tr -d ' \n')"; \
    actual_sha512="$(sha512sum "${npm_tarball}" | cut -d ' ' -f 1)"; \
    test "${actual_sha512}" = "${expected_sha512}"; \
    npm install --global "${npm_tarball}" --ignore-scripts --no-audit --no-fund; \
    rm -f "${npm_tarball}"; \
    brace_expansion_tarball="/tmp/brace-expansion-${BRACE_EXPANSION_VERSION}.tgz"; \
    wget -qO "${brace_expansion_tarball}" "https://registry.npmjs.org/brace-expansion/-/brace-expansion-${BRACE_EXPANSION_VERSION}.tgz"; \
    expected_sha512="$(printf '%s' "${BRACE_EXPANSION_TARBALL_SHA512}" | base64 -d | od -An -v -tx1 | tr -d ' \n')"; \
    actual_sha512="$(sha512sum "${brace_expansion_tarball}" | cut -d ' ' -f 1)"; \
    test "${actual_sha512}" = "${expected_sha512}"; \
    brace_expansion_dir="/usr/local/lib/node_modules/npm/node_modules/brace-expansion"; \
    rm -rf "${brace_expansion_dir}"; \
    mkdir -p "${brace_expansion_dir}"; \
    tar -xzf "${brace_expansion_tarball}" -C "${brace_expansion_dir}" --strip-components=1; \
    rm -f "${brace_expansion_tarball}"; \
    npm cache clean --force; \
    test "$(npm --version)" = "${NPM_VERSION}"; \
    grep -Eq "\"version\":[[:space:]]*\"${BRACE_EXPANSION_VERSION}\"" "${brace_expansion_dir}/package.json"; \
    mkdir -p /app/node_modules /workspace/node_modules /pnpm/store; \
    chown -R node:node /app /workspace /pnpm

LABEL org.opencontainers.image.source="https://github.com/StuHelper/StuHelper" \
      org.opencontainers.image.title="StuHelper Node.js development environment" \
      org.opencontainers.image.version="24.18.0-npm.11.18.0"
