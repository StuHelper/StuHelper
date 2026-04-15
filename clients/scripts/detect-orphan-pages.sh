#!/usr/bin/env bash
# 检测前端项目中未被路由引用的 .vue 页面文件
#
# 用法:
#   ./detect-orphan-pages.sh [project_dir]
#   # 默认检测 clients/web
#
# 原理：
#   扫描 views/ 下所有 .vue 文件，检查其路径是否出现在 router/ 文件中。
#   命中率不保证 100%（动态 import 变量化的场景无法覆盖），但能捕捉绝大多数孤儿页面。
set -euo pipefail

PROJECT_DIR="${1:-clients/web}"
ROUTER_DIR="${PROJECT_DIR}/src/router"
VIEWS_DIR="${PROJECT_DIR}/src"

if [[ ! -d "${ROUTER_DIR}" ]]; then
  echo "跳过：${ROUTER_DIR} 不存在"
  exit 0
fi

# 收集所有路由文件内容
router_content=$(cat "${ROUTER_DIR}"/*.ts 2>/dev/null || true)

orphans=()

while IFS= read -r vue_file; do
  # 将绝对路径转为 @ 别名路径（去掉 src/ 前缀，加 @/）
  relative="${vue_file#${PROJECT_DIR}/src/}"
  alias_path="@/${relative}"

  # 去掉 .vue 后缀也搜索（有些 import 不带后缀）
  alias_no_ext="${alias_path%.vue}"
  filename="$(basename "${vue_file}")"

  # 在路由文件中搜索
  if ! echo "${router_content}" | grep -qF "${alias_path}" &&
     ! echo "${router_content}" | grep -qF "${alias_no_ext}" &&
     ! echo "${router_content}" | grep -qF "${filename}"; then
    orphans+=("${relative}")
  fi
done < <(find "${VIEWS_DIR}" -path "*/views/*.vue" -type f 2>/dev/null | sort)

if [[ ${#orphans[@]} -eq 0 ]]; then
  echo "✅ 未检测到孤儿页面 (${PROJECT_DIR})"
  exit 0
fi

echo "⚠️  检测到 ${#orphans[@]} 个可能的孤儿页面 (${PROJECT_DIR}):"
for f in "${orphans[@]}"; do
  echo "  - ${f}"
done

# 默认不阻断 CI，仅警告；设置 ORPHAN_PAGES_STRICT=1 可改为失败退出
if [[ "${ORPHAN_PAGES_STRICT:-0}" == "1" ]]; then
  echo "❌ ORPHAN_PAGES_STRICT=1，视为失败"
  exit 1
fi
