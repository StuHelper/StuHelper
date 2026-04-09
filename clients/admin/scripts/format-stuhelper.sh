#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

FORMAT_TARGETS=(
  "package.json"
  "eslint.config.mjs"
  "oxfmt.config.ts"
  "oxlint.config.ts"
  "stylelint.config.mjs"
  "apps/backend-mock/routes"
  "apps/web-ele/src"
  "apps/web-ele/tests/e2e"
  "apps/web-ele/playwright.config.ts"
  "apps/web-ele/vite.config.ts"
  "apps/web-antd/src/api/core/menu.ts"
  "apps/web-antd/src/api/request.ts"
  "apps/web-antd/src/router/access.ts"
  "apps/web-antdv-next/src/api/core/menu.ts"
  "apps/web-antdv-next/src/api/request.ts"
  "apps/web-antdv-next/src/router/access.ts"
  "apps/web-naive/src/api/core/menu.ts"
  "apps/web-naive/src/api/request.ts"
  "apps/web-naive/src/router/access.ts"
  "apps/web-tdesign/src/api/core/menu.ts"
  "apps/web-tdesign/src/api/request.ts"
  "apps/web-tdesign/src/router/access.ts"
  "playground/src/api/core/menu.ts"
  "playground/src/api/request.ts"
  "playground/src/router/access.ts"
)

JS_TARGETS=(
  "apps/backend-mock/routes"
  "apps/web-ele/src"
  "apps/web-ele/tests/e2e"
  "apps/web-ele/playwright.config.ts"
  "apps/web-ele/vite.config.ts"
  "apps/web-antd/src/api/core/menu.ts"
  "apps/web-antd/src/api/request.ts"
  "apps/web-antd/src/router/access.ts"
  "apps/web-antdv-next/src/api/core/menu.ts"
  "apps/web-antdv-next/src/api/request.ts"
  "apps/web-antdv-next/src/router/access.ts"
  "apps/web-naive/src/api/core/menu.ts"
  "apps/web-naive/src/api/request.ts"
  "apps/web-naive/src/router/access.ts"
  "apps/web-tdesign/src/api/core/menu.ts"
  "apps/web-tdesign/src/api/request.ts"
  "apps/web-tdesign/src/router/access.ts"
  "playground/src/api/core/menu.ts"
  "playground/src/api/request.ts"
  "playground/src/router/access.ts"
)

pnpm exec oxfmt "${FORMAT_TARGETS[@]}"
pnpm exec oxlint --vue-plugin --fix "${JS_TARGETS[@]}"
pnpm exec eslint --cache --fix "${JS_TARGETS[@]}"
pnpm exec stylelint "apps/web-ele/**/*.{vue,css,less,scss}" --cache --fix --allow-empty-input
