#!/usr/bin/env bash
set -euo pipefail

HOST="${HOST:-127.0.0.1}"
PORT="${PORT:-8277}"

uvicorn app.main:app --host "$HOST" --port "$PORT" --reload
