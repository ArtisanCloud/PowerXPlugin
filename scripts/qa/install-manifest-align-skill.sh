#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
  cat <<'EOF'
用法:
  bash scripts/qa/install-manifest-align-skill.sh <插件仓库绝对路径或相对路径>

示例:
  bash scripts/qa/install-manifest-align-skill.sh /path/to/com.powerx.plugin.demo
EOF
  exit 1
fi

TARGET_REPO="$(cd "$1" && pwd)"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SOURCE_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

SKILL_SRC_DIR="${SOURCE_ROOT}/tools/cli/internal/templates/data/.codex/skills/ci/manifest-align"
SKILL_DST_DIR="${TARGET_REPO}/.codex/skills/ci/manifest-align"

if [[ ! -d "${TARGET_REPO}" ]]; then
  echo "❌ 目标目录不存在: ${TARGET_REPO}"
  exit 1
fi

if [[ ! -f "${SKILL_SRC_DIR}/SKILL.md" || ! -f "${SKILL_SRC_DIR}/scripts/manifest-align-check.mjs" ]]; then
  echo "❌ 源 Skill 文件不存在: ${SKILL_SRC_DIR}"
  exit 1
fi

echo "==> 注入 ci-manifest-align skill 到目标插件仓"
mkdir -p "${SKILL_DST_DIR}/scripts"
cp "${SKILL_SRC_DIR}/SKILL.md" "${SKILL_DST_DIR}/SKILL.md"
cp "${SKILL_SRC_DIR}/scripts/manifest-align-check.mjs" "${SKILL_DST_DIR}/scripts/manifest-align-check.mjs"
echo "✅ Skill 已写入: ${SKILL_DST_DIR}"

MANIFEST_MK="${TARGET_REPO}/make-files/manifest.mk"
if [[ -f "${MANIFEST_MK}" ]]; then
  if ! grep -q "manifest-align-fix" "${MANIFEST_MK}"; then
    cat >> "${MANIFEST_MK}" <<'EOF'

.PHONY: manifest-align-fix
manifest-align-fix: ## 自动同步 plugin.d 并校验 capability -> exposure/rbac 映射
	@node .codex/skills/ci/manifest-align/scripts/manifest-align-check.mjs --fix

.PHONY: manifest-align-check
manifest-align-check: ## 严格检查 plugin.d 漂移与映射（CI gate）
	@node .codex/skills/ci/manifest-align/scripts/manifest-align-check.mjs
EOF
    echo "✅ 已注入 make 目标: ${MANIFEST_MK}"
  else
    echo "ℹ️ 已存在 make 目标，跳过: ${MANIFEST_MK}"
  fi
else
  echo "⚠️ 未找到 ${MANIFEST_MK}，请手工加入 manifest-align-fix/check 目标"
fi

README_FILE="${TARGET_REPO}/README.md"
if [[ -f "${README_FILE}" ]]; then
  if ! grep -q "manifest-align-fix" "${README_FILE}"; then
    cat >> "${README_FILE}" <<'EOF'

## 清单对齐守卫（推荐）

```bash
# 功能迭代后先自动同步并校验 catalogs
make manifest-align-fix

# CI 使用严格模式（有漂移即失败）
make manifest-align-check
```

- Skill 说明：`.codex/skills/ci/manifest-align/SKILL.md`
EOF
    echo "✅ 已更新 README: ${README_FILE}"
  else
    echo "ℹ️ README 已包含 manifest-align 说明，跳过"
  fi
fi

INSTALL_DOC="${TARGET_REPO}/docs/INSTALL_TO_POWERX.md"
if [[ -f "${INSTALL_DOC}" ]]; then
  if ! grep -q "步骤 1.5：先做清单对齐" "${INSTALL_DOC}"; then
    cat >> "${INSTALL_DOC}" <<'EOF'

### 步骤 1.5：先做清单对齐（强烈建议）

功能迭代涉及 `contracts/capabilities/*.yaml` 时，先执行：

```bash
make manifest-align-fix
```

CI 建议使用严格模式（有漂移即失败）：

```bash
make manifest-align-check
```
EOF
    echo "✅ 已更新安装文档: ${INSTALL_DOC}"
  else
    echo "ℹ️ INSTALL_TO_POWERX 已包含清单对齐说明，跳过"
  fi
fi

echo ""
echo "==> 下一步（在目标插件仓执行）"
echo "1) make manifest-align-fix"
echo "2) make dist"
echo "3) make local-install 或 make skeleton-reinstall"
