#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
cd "$ROOT_DIR"

FASTAPI_DIR="skeleton/backend/python-fastapi"
REQ_FILE="$FASTAPI_DIR/requirements.txt"

check_file() {
  if [ ! -f "$1" ]; then
    echo "Error: missing $1" >&2
    exit 1
  fi
}

check_dir() {
  if [ ! -d "$1" ]; then
    echo "Error: missing $1" >&2
    exit 1
  fi
}

check_dep() {
  if ! rg -i "^$1" "$REQ_FILE" >/dev/null 2>&1; then
    echo "Error: missing dependency $1 in $REQ_FILE" >&2
    exit 1
  fi
}

echo "=== FastAPI Smoke Tests Start ==="

check_dir "$FASTAPI_DIR"
check_file "$REQ_FILE"
check_file "$FASTAPI_DIR/app/main.py"
check_file "$FASTAPI_DIR/app/router/api.py"
check_file "$FASTAPI_DIR/app/transport/http/health.py"
check_file "$FASTAPI_DIR/app/middleware/auth_guard.py"
check_file "$FASTAPI_DIR/app/middleware/tenant_context.py"
check_dir "$FASTAPI_DIR/migrations/versions"

check_dep "fastapi"
check_dep "uvicorn"
check_dep "sqlalchemy"
check_dep "alembic"

echo "=== FastAPI Smoke Tests Finished ==="
