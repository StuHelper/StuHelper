#!/usr/bin/env bash
# 检测前端项目中未被任何源码引用的 .vue 页面文件
#
# 用法:
#   ./detect-orphan-pages.sh [project_dir]
#   # 默认检测 clients/web
#
# 原理：
#   扫描 src/**/views/*.vue，检查是否被 router 或其他源码文件 import / 引用。
#   默认发现孤儿页面即失败；可用 ORPHAN_PAGES_ALLOWLIST 显式豁免。
set -euo pipefail

PROJECT_DIR="${1:-clients/web}"
SRC_DIR="${PROJECT_DIR}/src"
ROUTER_DIR="${SRC_DIR}/router"
VIEWS_DIR="${SRC_DIR}"
ALLOWLIST_RAW="${ORPHAN_PAGES_ALLOWLIST:-}"

if [[ ! -d "${ROUTER_DIR}" ]]; then
  echo "❌ 错误：${ROUTER_DIR} 不存在"
  exit 1
fi

is_allowlisted() {
  local candidate="$1"
  local allowlist="${ALLOWLIST_RAW//,/ }"
  for item in ${allowlist}; do
    if [[ "${candidate}" == "${item}" ]]; then
      return 0
    fi
  done
  return 1
}

is_referenced() {
  local vue_file="$1"
  local relative="$2"
  local alias_path="$3"
  local alias_no_ext="$4"
  local filename="$5"

  if rg -lF -- "${alias_path}" "${SRC_DIR}" | grep -Fvx -- "${vue_file}" >/dev/null 2>&1; then
    return 0
  fi
  if rg -lF -- "${alias_no_ext}" "${SRC_DIR}" | grep -Fvx -- "${vue_file}" >/dev/null 2>&1; then
    return 0
  fi
  if rg -lF -- "${filename}" "${SRC_DIR}" | grep -Fvx -- "${vue_file}" >/dev/null 2>&1; then
    return 0
  fi
  if is_allowlisted "${relative}"; then
    return 0
  fi
  return 1
}

orphans=()

while IFS= read -r vue_file; do
  relative="${vue_file#${PROJECT_DIR}/src/}"
  alias_path="@/${relative}"
  alias_no_ext="${alias_path%.vue}"
  filename="$(basename "${vue_file}")"

  if ! is_referenced "${vue_file}" "${relative}" "${alias_path}" "${alias_no_ext}" "${filename}"; then
    orphans+=("${relative}")
  fi
done < <(find "${VIEWS_DIR}" -path "*/views/*.vue" -type f 2>/dev/null | sort)

if [[ ${#orphans[@]} -eq 0 ]]; then
  echo "✅ 未检测到孤儿页面 (${PROJECT_DIR})"
  exit 0
fi

echo "❌ 检测到 ${#orphans[@]} 个孤儿页面 (${PROJECT_DIR}):"
for f in "${orphans[@]}"; do
  echo "  - ${f}"
done

echo "提示：如属有意保留，请通过 ORPHAN_PAGES_ALLOWLIST=path1,path2 显式豁免。"
exit 1
