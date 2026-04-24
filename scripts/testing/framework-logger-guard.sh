#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
cd "$ROOT_DIR"

usage() {
  cat <<'EOF'
Usage:
  scripts/testing/framework-logger-guard.sh [path...]

Env:
  FRAMEWORK_LOGGER_GUARD_MODE=detect|warn|block   (default: warn)
  FRAMEWORK_LOGGER_GUARD_CURRENT_VERSION=<ver>     (optional)
  FRAMEWORK_LOGGER_GUARD_DEADLINE_VERSION=<ver>    (optional)
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

mode="${FRAMEWORK_LOGGER_GUARD_MODE:-warn}"
mode="$(echo "$mode" | tr '[:upper:]' '[:lower:]' | xargs)"
case "$mode" in
  detect|warn|block) ;;
  *)
    echo "invalid FRAMEWORK_LOGGER_GUARD_MODE: $mode" >&2
    exit 2
    ;;
esac

current_version="${FRAMEWORK_LOGGER_GUARD_CURRENT_VERSION:-${POWERX_PLUGIN_VERSION:-}}"
deadline_version="${FRAMEWORK_LOGGER_GUARD_DEADLINE_VERSION:-${POWERX_LOG_GOVERNANCE_DEADLINE_VERSION:-}}"

compare_version() {
  local current="$1"
  local target="$2"
  local script
  script='
import re, sys
def norm(v):
    v = (v or "").strip().lower()
    if v.startswith("v"):
        v = v[1:]
    parts = re.split(r"[._-]+", v)
    out = []
    for p in parts:
        if not p:
            continue
        m = re.match(r"(\d+)", p)
        out.append(int(m.group(1)) if m else 0)
    return out or [0]
a = norm(sys.argv[1]); b = norm(sys.argv[2])
n = max(len(a), len(b))
a += [0]*(n-len(a)); b += [0]*(n-len(b))
if a > b: print(1)
elif a < b: print(-1)
else: print(0)
'
  python3 -c "$script" "$current" "$target"
}

should_block_by_deadline=false
if [[ -n "$current_version" && -n "$deadline_version" ]]; then
  cmp="$(compare_version "$current_version" "$deadline_version")"
  if [[ "$cmp" -ge 0 ]]; then
    should_block_by_deadline=true
  fi
fi

if [[ "$#" -gt 0 ]]; then
  targets=("$@")
else
  targets=("skeleton/backend/go-gin" "framework/backend/go")
fi

rg_base=(rg -n --color never --glob '*.go' --glob '!**/*_test.go')

exclude_args=(
  -g '!skeleton/backend/go-gin/internal/logger/**'
  -g '!framework/backend/go/runtime/common/logging/**'
  -g '!**/vendor/**'
  -g '!**/.tmp/**'
)

logrus_hits="$("${rg_base[@]}" "${exclude_args[@]}" 'github.com/sirupsen/logrus|logrus\.' "${targets[@]}" || true)"
zap_hits="$("${rg_base[@]}" "${exclude_args[@]}" 'go\.uber\.org/zap|zap\.' "${targets[@]}" || true)"
file_hits="$("${rg_base[@]}" "${exclude_args[@]}" 'backend/logs/|\.log"' "${targets[@]}" || true)"

count_lines() {
  local input="$1"
  if [[ -z "$input" ]]; then
    echo 0
    return
  fi
  printf '%s\n' "$input" | sed '/^[[:space:]]*$/d' | wc -l | awk '{print $1}'
}

logrus_count="$(count_lines "$logrus_hits")"
zap_count="$(count_lines "$zap_hits")"
file_count="$(count_lines "$file_hits")"
violation_count=$((logrus_count + zap_count + file_count))

status="resolved"
if [[ "$violation_count" -gt 0 ]]; then
  if [[ "$mode" == "detect" ]]; then
    status="detected"
  else
    status="warned"
  fi
  if [[ "$mode" == "block" || "$should_block_by_deadline" == "true" ]]; then
    status="blocked"
  fi
fi

echo "status=$status"
echo "violations=$violation_count"
echo "direct_logrus=$logrus_count"
echo "direct_zap=$zap_count"
echo "direct_file=$file_count"
if [[ "$should_block_by_deadline" == "true" ]]; then
  echo "deadline_triggered=true"
  echo "current_version=$current_version"
  echo "deadline_version=$deadline_version"
fi

if [[ -n "$logrus_hits" ]]; then
  echo "--- direct_logrus ---"
  printf '%s\n' "$logrus_hits"
fi
if [[ -n "$zap_hits" ]]; then
  echo "--- direct_zap ---"
  printf '%s\n' "$zap_hits"
fi
if [[ -n "$file_hits" ]]; then
  echo "--- direct_file ---"
  printf '%s\n' "$file_hits"
fi

if [[ "$status" == "blocked" ]]; then
  exit 1
fi
exit 0
