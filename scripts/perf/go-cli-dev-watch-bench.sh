#!/usr/bin/env bash
# Go CLI dev --watch performance + health benchmark
# Produces structured JSON metrics for CI/Grafana consumption and a human readable summary.

set -euo pipefail

# ----------------------------------------------------------------------
# Paths & configuration
# ----------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CLI_DIR="$PROJECT_ROOT/tools/cli"
BENCH_ROOT="$PROJECT_ROOT/tmp/go-cli-dev-watch-bench"
PLUGIN_DIR="$BENCH_ROOT/plugin"
CLI_BIN="$BENCH_ROOT/px-plugin"
CLI_LOG="$BENCH_ROOT/dev-watch.log"
MOCK_API_LOG="$BENCH_ROOT/mock-dev-api.log"
MOCK_API_EVENTS="$BENCH_ROOT/mock-dev-api-events.log"
JSON_REPORT="$BENCH_ROOT/go-cli-dev-watch-bench.json"
MARKDOWN_REPORT="$BENCH_ROOT/go-cli-dev-watch-bench.md"

mkdir -p "$BENCH_ROOT" "$PLUGIN_DIR"

# Ensure standalone builds are not affected by repository go.work.
export GOWORK=off

# Thresholds (ms or MB)
STARTUP_THRESHOLD=500           # CLI help
DEV_READY_THRESHOLD=5000        # Initial build ready
RELOAD_THRESHOLD=2000           # Reload P95
FILE_TO_API_THRESHOLD=250       # File change -> API call
MEMORY_THRESHOLD_MB=100         # RSS mega bytes
LIST_SESSIONS_THRESHOLD=1000    # ms

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

CLI_PID=""
MOCK_API_PID=""

# ----------------------------------------------------------------------
# Helpers
# ----------------------------------------------------------------------
log_section() {
    printf "\n=== %s ===\n" "$1"
}

log_info() {
    printf "${YELLOW}ℹ${NC} %s\n" "$1"
}

log_success() {
    printf "${GREEN}✓${NC} %s\n" "$1"
}

log_error() {
    printf "${RED}✗${NC} %s\n" "$1"
}

upper() {
    echo "$1" | tr '[:lower:]' '[:upper:]'
}

require_cmd() {
    if ! command -v "$1" >/dev/null 2>&1; then
        log_error "Missing dependency: $1"
        exit 1
    fi
}

now_ms() {
    python3 - <<'PY'
import time
print(int(time.time() * 1000))
PY
}

measure_command_time() {
    local start
    start="$(now_ms)"
    if "$@" >/dev/null 2>&1; then
        local end
        end="$(now_ms)"
        echo $((end - start))
        return 0
    else
        local status=$?
        local end
        end="$(now_ms)"
        echo $((end - start))
        return $status
    fi
}

get_memory_usage_mb() {
    local pid=$1
    if ! ps -p "$pid" >/dev/null 2>&1; then
        echo ""
        return 1
    fi
    local rss
    rss="$(ps -o rss= -p "$pid" 2>/dev/null | awk '{print $1}')"
    if [[ -z "$rss" ]]; then
        echo ""
        return 1
    fi
    python3 - "$rss" <<'PY'
import sys
rss_kb = int(sys.argv[1])
print(round(rss_kb / 1024.0, 2))
PY
}

wait_for_text_pattern() {
    local file=$1
    local pattern=$2
    local timeout=$3
    local offset=$4
    python3 - "$file" "$pattern" "$timeout" "$offset" <<'PY'
import sys, time, os
file, pattern, timeout, offset = sys.argv[1], sys.argv[2], float(sys.argv[3]), int(sys.argv[4])
deadline = time.time() + timeout
try:
    f = open(file, 'r', encoding='utf-8', errors='ignore')
except FileNotFoundError:
    print(offset)
    sys.exit(1)
f.seek(offset)
while time.time() < deadline:
    line = f.readline()
    if line:
        offset = f.tell()
        if pattern in line:
            print(offset)
            sys.exit(0)
    else:
        time.sleep(0.1)
print(offset)
sys.exit(1)
PY
}

wait_for_event() {
    local file=$1
    local event=$2
    local timeout=$3
    local offset=$4
    python3 - "$file" "$event" "$timeout" "$offset" <<'PY'
import sys, time, json
file, event, timeout, offset = sys.argv[1], sys.argv[2], float(sys.argv[3]), int(sys.argv[4])
deadline = time.time() + timeout
try:
    f = open(file, 'r', encoding='utf-8')
except FileNotFoundError:
    print(f"{offset}|0")
    sys.exit(1)
f.seek(offset)
while time.time() < deadline:
    line = f.readline()
    if line:
        offset = f.tell()
        try:
            payload = json.loads(line)
        except json.JSONDecodeError:
            continue
        if payload.get("event") == event:
            ts = int(payload.get("ts", 0))
            print(f"{offset}|{ts}")
            sys.exit(0)
    else:
        time.sleep(0.1)
print(f"{offset}|0")
sys.exit(1)
PY
}

cleanup() {
    if [[ -n "$CLI_PID" ]] && ps -p "$CLI_PID" >/dev/null 2>&1; then
        kill "$CLI_PID" >/dev/null 2>&1 || true
        sleep 1
        kill -9 "$CLI_PID" >/dev/null 2>&1 || true
    fi
    if [[ -n "$MOCK_API_PID" ]] && ps -p "$MOCK_API_PID" >/dev/null 2>&1; then
        kill "$MOCK_API_PID" >/dev/null 2>&1 || true
    fi
}

trap cleanup EXIT

# ----------------------------------------------------------------------
# Pre-flight
# ----------------------------------------------------------------------
require_cmd go
require_cmd python3

log_section "Go CLI dev --watch Benchmark"
log_info "Project root: $PROJECT_ROOT"
log_info "Bench artifacts: $BENCH_ROOT"

# ----------------------------------------------------------------------
# Prepare sample plugin
# ----------------------------------------------------------------------
cat > "$PLUGIN_DIR/plugin.yaml" <<'EOF'
id: go-cli-perf-bench
version: 0.1.0
backend:
  entry: ./main.go
EOF

cat > "$PLUGIN_DIR/main.go" <<'EOF'
package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("bench boot")
	time.Sleep(50 * time.Millisecond)
}
EOF

# Minimal go.mod to ensure builder detects Go project.
cat > "$PLUGIN_DIR/go.mod" <<'EOF'
module go-cli-perf-bench

go 1.24
EOF

log_success "Sample plugin prepared at $PLUGIN_DIR"

# ----------------------------------------------------------------------
# Build CLI binary
# ----------------------------------------------------------------------
log_section "Building CLI"
if (cd "$CLI_DIR" && go build -o "$CLI_BIN" ./cmd/px-plugin); then
    log_success "CLI built -> $CLI_BIN"
else
    log_error "Failed to build CLI"
    exit 1
fi

# ----------------------------------------------------------------------
# Start mock Dev API
# ----------------------------------------------------------------------
log_section "Bootstrapping mock Dev API"
cat > "$BENCH_ROOT/mock_dev_api.py" <<'PY'
import json, sys, time
from http.server import HTTPServer, BaseHTTPRequestHandler

RELOAD_TOKEN = "bench-reload-token"
SESSION_ID = "bench-session"
EVENT_LOG = sys.argv[2]

def record_event(event):
    if not EVENT_LOG:
        return
    payload = {"event": event, "ts": int(time.time() * 1000)}
    with open(EVENT_LOG, "a", encoding="utf-8") as f:
        f.write(json.dumps(payload) + "\n")

class Handler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, format, *args):
        return

    def _json(self, status, payload):
        body = json.dumps(payload).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.send_header("Cache-Control", "no-store")
        self.end_headers()
        self.wfile.write(body)

    def do_POST(self):
        length = int(self.headers.get("Content-Length") or 0)
        raw = self.rfile.read(length) if length else b"{}"
        try:
            body = json.loads(raw)
        except Exception:
            body = {}

        if self.path == "/internal/dev/plugins/register":
            record_event("register")
            self._json(201, {
                "sessionId": SESSION_ID,
                "reloadToken": RELOAD_TOKEN,
                "devUrl": f"http://{self.server.server_address[0]}:{self.server.server_port}/dev/{SESSION_ID}",
                "expiresAt": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(time.time() + 3600))
            })
        elif self.path == "/internal/dev/plugins/reload":
            if self.headers.get("Authorization") != f"Bearer {RELOAD_TOKEN}":
                self._json(401, {"error": "UNAUTHORIZED", "message": "invalid token"})
                return
            record_event("reload")
            self._json(200, {
                "status": "success",
                "reloadId": "reload-1",
                "estimatedTime": 120,
                "message": "Mock reload completed"
            })
        else:
            self._json(404, {"error": "NOT_FOUND", "message": self.path})

    def do_DELETE(self):
        if self.headers.get("Authorization") != f"Bearer {RELOAD_TOKEN}":
            self._json(401, {"error": "UNAUTHORIZED", "message": "invalid token"})
            return
        if self.path.startswith("/internal/dev/plugins/register/"):
            record_event("delete")
            self._json(200, {"status": "success"})
        else:
            self._json(404, {"error": "NOT_FOUND", "message": self.path})

    def do_GET(self):
        if self.path == "/healthz":
            self._json(200, {"status": "ok"})
        else:
            self._json(404, {"error": "NOT_FOUND", "message": self.path})

if __name__ == "__main__":
    port = int(sys.argv[1])
    server = HTTPServer(("127.0.0.1", port), Handler)
    record_event("mock-started")
    server.serve_forever()
PY

MOCK_PORT="$(python3 - <<'PY'
import socket
s = socket.socket()
s.bind(("127.0.0.1", 0))
port = s.getsockname()[1]
s.close()
print(port)
PY
)"
: > "$MOCK_API_EVENTS"

python3 "$BENCH_ROOT/mock_dev_api.py" "$MOCK_PORT" "$MOCK_API_EVENTS" >"$MOCK_API_LOG" 2>&1 &
MOCK_API_PID=$!
sleep 1
log_success "Mock Dev API listening at http://127.0.0.1:$MOCK_PORT"

DEV_API_URL="http://127.0.0.1:$MOCK_PORT"

# ----------------------------------------------------------------------
# Run benchmarks
# ----------------------------------------------------------------------
CLI_START_MS="$(now_ms)"
: > "$CLI_LOG"
"$CLI_BIN" dev --watch --entry "$PLUGIN_DIR" --dev-api "$DEV_API_URL" >"$CLI_LOG" 2>&1 &
CLI_PID=$!
sleep 1

CLI_LOG_OFFSET=0
EVENT_OFFSET=0

ready_output="$(wait_for_text_pattern "$CLI_LOG" "Initial build complete" 30 "$CLI_LOG_OFFSET" || true)"
READY_STATUS=$?
CLI_LOG_OFFSET="${ready_output:-0}"
DEV_READY_TIME_MS=""
if [[ $READY_STATUS -eq 0 ]]; then
    DEV_READY_TIME_MS=$(( $(now_ms) - CLI_START_MS ))
    log_success "Initial build completed (≈ ${DEV_READY_TIME_MS}ms)"
else
    log_error "Failed to observe initial build completion (see $CLI_LOG)"
fi

# CLI startup measurement
log_section "Measuring CLI commands"
STARTUP_TIME_MS="$(measure_command_time "$CLI_BIN" --help || true)"
log_info "CLI startup (--help) took ${STARTUP_TIME_MS}ms"

# List sessions measurement
LIST_SESSIONS_TIME_MS="$(measure_command_time "$CLI_BIN" dev --list-sessions || true)"
log_info "List sessions took ${LIST_SESSIONS_TIME_MS}ms"

# Memory usage snapshot
sleep 1
MEMORY_MB="$(get_memory_usage_mb "$CLI_PID" || true)"
if [[ -n "$MEMORY_MB" ]]; then
    log_info "dev --watch memory usage ≈ ${MEMORY_MB}MB"
else
    log_info "Unable to determine memory usage (likely unsupported on host OS)"
fi

# Modify file to trigger reload
log_section "Measuring reload latency"
FILE_CHANGE_MS="$(now_ms)"
cat > "$PLUGIN_DIR/main.go" <<'EOF'
package main

import "fmt"

func main() {
	fmt.Println("bench reload trigger")
}
EOF

reload_event_output="$(wait_for_event "$MOCK_API_EVENTS" "reload" 30 "$EVENT_OFFSET" || true)"
EVENT_STATUS=$?
FILE_TO_API_MS=""
if [[ -n "$reload_event_output" ]]; then
    EVENT_OFFSET="$(echo "$reload_event_output" | cut -d'|' -f1)"
    EVENT_TS="$(echo "$reload_event_output" | cut -d'|' -f2)"
    if [[ "$EVENT_STATUS" -eq 0 && "$EVENT_TS" -gt 0 ]]; then
        FILE_TO_API_MS=$(( EVENT_TS - FILE_CHANGE_MS ))
        if (( FILE_TO_API_MS < 0 )); then FILE_TO_API_MS=0; fi
        log_success "File change reached Dev API in ${FILE_TO_API_MS}ms"
    else
        log_error "Did not observe reload call in mock Dev API (see $MOCK_API_LOG)"
    fi
else
    log_error "Failed to read mock Dev API events"
fi

reload_log_output="$(wait_for_text_pattern "$CLI_LOG" "Reload applied" 30 "$CLI_LOG_OFFSET" || true)"
RELOAD_STATUS=$?
RELOAD_LATENCY_MS=""
if [[ -n "$reload_log_output" ]]; then
    CLI_LOG_OFFSET="$reload_log_output"
    if [[ $RELOAD_STATUS -eq 0 ]]; then
        RELOAD_LATENCY_MS=$(( $(now_ms) - FILE_CHANGE_MS ))
        log_success "End-to-end reload completed in ${RELOAD_LATENCY_MS}ms"
    else
        log_error "Reload completion log not detected"
    fi
else
    log_error "Failed to read CLI log for reload completion"
fi

sleep 1

# ----------------------------------------------------------------------
# Stop processes
# ----------------------------------------------------------------------
cleanup
CLI_PID=""
MOCK_API_PID=""

# ----------------------------------------------------------------------
# Evaluate metrics
# ----------------------------------------------------------------------
STATUS_STARTUP=$([[ "${STARTUP_TIME_MS:-0}" -lt "$STARTUP_THRESHOLD" ]] && echo "pass" || echo "fail")
STATUS_READY=$([[ -n "${DEV_READY_TIME_MS}" && "${DEV_READY_TIME_MS:-0}" -lt "$DEV_READY_THRESHOLD" ]] && echo "pass" || echo "fail")
MEMORY_OK=0
if [[ -n "${MEMORY_MB}" ]]; then
    MEMORY_OK=$(python3 - "$MEMORY_MB" "$MEMORY_THRESHOLD_MB" <<'PY'
import sys
value=float(sys.argv[1])
threshold=float(sys.argv[2])
print(1 if value < threshold else 0)
PY
)
fi
STATUS_MEMORY=$([ "$MEMORY_OK" -eq 1 ] && echo "pass" || echo "fail")
STATUS_LIST=$([[ "${LIST_SESSIONS_TIME_MS:-0}" -lt "$LIST_SESSIONS_THRESHOLD" ]] && echo "pass" || echo "fail")
STATUS_RELOAD=$([[ -n "${RELOAD_LATENCY_MS}" && "${RELOAD_LATENCY_MS:-0}" -lt "$RELOAD_THRESHOLD" ]] && echo "pass" || echo "fail")
STATUS_FILE_TO_API=$([[ -n "${FILE_TO_API_MS}" && "${FILE_TO_API_MS:-0}" -lt "$FILE_TO_API_THRESHOLD" ]] && echo "pass" || echo "fail")

TEST_TOTAL=6
TEST_PASS=0
for status in "$STATUS_STARTUP" "$STATUS_READY" "$STATUS_MEMORY" "$STATUS_LIST" "$STATUS_RELOAD" "$STATUS_FILE_TO_API"; do
    if [[ "$status" == "pass" ]]; then
        ((TEST_PASS++))
    fi
done
OVERALL_STATUS=$([ "$TEST_PASS" -eq "$TEST_TOTAL" ] && echo "pass" || echo "fail")

# ----------------------------------------------------------------------
# Write reports
# ----------------------------------------------------------------------
log_section "Generating reports"
TIMESTAMP="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

json_value() {
    local value=$1
    if [[ -z "$value" ]]; then
        echo "null"
    else
        echo "$value"
    fi
}

cat > "$JSON_REPORT" <<EOF
{
  "generatedAt": "$TIMESTAMP",
  "devApiBase": "$DEV_API_URL",
  "artifacts": {
    "binary": "$CLI_BIN",
    "logFile": "$CLI_LOG",
    "mockApiLog": "$MOCK_API_LOG",
    "mockApiEvents": "$MOCK_API_EVENTS"
  },
  "thresholds": {
    "startupMs": $STARTUP_THRESHOLD,
    "devReadyMs": $DEV_READY_THRESHOLD,
    "reloadMs": $RELOAD_THRESHOLD,
    "fileChangeToApiMs": $FILE_TO_API_THRESHOLD,
    "memoryMb": $MEMORY_THRESHOLD_MB,
    "listSessionsMs": $LIST_SESSIONS_THRESHOLD
  },
  "metrics": {
    "startupTimeMs": { "value": $(json_value "$STARTUP_TIME_MS"), "status": "$STATUS_STARTUP" },
    "devReadyTimeMs": { "value": $(json_value "$DEV_READY_TIME_MS"), "status": "$STATUS_READY" },
    "memoryUsageMb": { "value": $(json_value "$MEMORY_MB"), "status": "$STATUS_MEMORY" },
    "listSessionsMs": { "value": $(json_value "$LIST_SESSIONS_TIME_MS"), "status": "$STATUS_LIST" },
    "fileChangeToApiMs": { "value": $(json_value "$FILE_TO_API_MS"), "status": "$STATUS_FILE_TO_API" },
    "reloadLatencyMs": { "value": $(json_value "$RELOAD_LATENCY_MS"), "status": "$STATUS_RELOAD" }
  },
  "summary": {
    "testsPassed": $TEST_PASS,
    "testsTotal": $TEST_TOTAL,
    "status": "$OVERALL_STATUS"
  }
}
EOF

cat > "$MARKDOWN_REPORT" <<EOF
# Go CLI dev --watch Benchmark

- Generated: ${TIMESTAMP}
- Dev API: ${DEV_API_URL}
- CLI Binary: ${CLI_BIN}
- CLI Log: ${CLI_LOG}

| Metric | Value | Threshold | Status |
|--------|-------|-----------|--------|
| Startup (--help) | ${STARTUP_TIME_MS}ms | < ${STARTUP_THRESHOLD}ms | $(upper "${STATUS_STARTUP}") |
| Initial build ready | ${DEV_READY_TIME_MS:-n/a}ms | < ${DEV_READY_THRESHOLD}ms | $(upper "${STATUS_READY}") |
| Memory usage | ${MEMORY_MB:-n/a}MB | < ${MEMORY_THRESHOLD_MB}MB | $(upper "${STATUS_MEMORY}") |
| List sessions | ${LIST_SESSIONS_TIME_MS}ms | < ${LIST_SESSIONS_THRESHOLD}ms | $(upper "${STATUS_LIST}") |
| File change → Dev API | ${FILE_TO_API_MS:-n/a}ms | < ${FILE_TO_API_THRESHOLD}ms | $(upper "${STATUS_FILE_TO_API}") |
| Reload latency | ${RELOAD_LATENCY_MS:-n/a}ms | < ${RELOAD_THRESHOLD}ms | $(upper "${STATUS_RELOAD}") |

**Overall:** $(upper "${OVERALL_STATUS}") (${TEST_PASS}/${TEST_TOTAL} tests passed)
EOF

log_success "JSON report -> $JSON_REPORT"
log_success "Markdown report -> $MARKDOWN_REPORT"
log_info "Done."

if [[ "$OVERALL_STATUS" == "pass" ]]; then
    exit 0
else
    exit 1
fi
