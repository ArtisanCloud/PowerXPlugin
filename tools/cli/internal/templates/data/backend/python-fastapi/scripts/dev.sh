#!/usr/bin/env bash
set -euo pipefail

HOST="${HOST:-127.0.0.1}"
PORT="${PORT:-{{ .BackendPort }}}"

uvicorn app.main:app --host "$HOST" --port "$PORT" --reload
