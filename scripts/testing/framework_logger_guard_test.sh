#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
cd "$ROOT_DIR"

SCRIPT="./scripts/testing/framework-logger-guard.sh"

tmp_dir="$(mktemp -d)"
cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

mkdir -p "$tmp_dir/src"

# Case 1: no violation => resolved
cat >"$tmp_dir/src/clean.go" <<'EOF'
package demo

func ok() {}
EOF

out="$($SCRIPT "$tmp_dir/src")"
echo "$out" | grep -q '^status=resolved$'

# Case 2: violation with warn mode => warned + exit 0
cat >"$tmp_dir/src/viol.go" <<'EOF'
package demo

import "github.com/sirupsen/logrus"

func bad() {
	logrus.Info("legacy")
}
EOF

out="$(FRAMEWORK_LOGGER_GUARD_MODE=warn "$SCRIPT" "$tmp_dir/src")"
echo "$out" | grep -q '^status=warned$'
echo "$out" | grep -E '^violations=[1-9][0-9]*$' >/dev/null

# Case 3: violation with block mode => blocked + non-zero
if FRAMEWORK_LOGGER_GUARD_MODE=block "$SCRIPT" "$tmp_dir/src" >/tmp/framework_logger_guard_block.out 2>&1; then
  echo "expected block mode to fail"
  exit 1
fi
grep -q '^status=blocked$' /tmp/framework_logger_guard_block.out

# Case 4: deadline reached => blocked even under warn mode
if FRAMEWORK_LOGGER_GUARD_MODE=warn FRAMEWORK_LOGGER_GUARD_CURRENT_VERSION=1.2.0 FRAMEWORK_LOGGER_GUARD_DEADLINE_VERSION=1.2.0 "$SCRIPT" "$tmp_dir/src" >/tmp/framework_logger_guard_deadline.out 2>&1; then
  echo "expected deadline-triggered block to fail"
  exit 1
fi
grep -q '^status=blocked$' /tmp/framework_logger_guard_deadline.out
grep -q '^deadline_triggered=true$' /tmp/framework_logger_guard_deadline.out

echo "framework_logger_guard_test: PASS"
