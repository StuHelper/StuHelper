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
  "apps/web-ele/src"
  "apps/web-ele/tests/e2e"
  "apps/web-ele/playwright.config.ts"
  "apps/web-ele/vite.config.ts"
)

JS_TARGETS=(
  "apps/web-ele/src"
  "apps/web-ele/tests/e2e"
  "apps/web-ele/playwright.config.ts"
  "apps/web-ele/vite.config.ts"
)

pnpm exec oxfmt "${FORMAT_TARGETS[@]}"
pnpm exec oxlint --vue-plugin --fix "${JS_TARGETS[@]}"
pnpm exec eslint --cache --fix "${JS_TARGETS[@]}"
pnpm exec stylelint "apps/web-ele/**/*.{vue,css,less,scss}" --cache --fix --allow-empty-input
