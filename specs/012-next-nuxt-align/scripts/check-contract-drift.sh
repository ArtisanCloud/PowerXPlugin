#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FEATURE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
ROOT_DIR="$(cd "$FEATURE_DIR/.."/.. && pwd)"
CONTRACT_FILE="$FEATURE_DIR/contracts/openapi.yaml"
NEXT_API_DIR="$ROOT_DIR/skeleton/web-admin/next/lib/api"

if [[ ! -f "$CONTRACT_FILE" ]]; then
  echo "[contract-drift] missing contract file: $CONTRACT_FILE" >&2
  exit 2
fi

if [[ ! -d "$NEXT_API_DIR" ]]; then
  echo "[contract-drift] missing next api dir: $NEXT_API_DIR" >&2
  exit 2
fi

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

CONTRACT_PATHS="$TMP_DIR/contract-paths.txt"
NEXT_PATHS_RAW="$TMP_DIR/next-paths-raw.txt"
NEXT_PATHS="$TMP_DIR/next-paths.txt"
UNEXPECTED="$TMP_DIR/unexpected.txt"

# OpenAPI paths
awk '/^  \/.*/ {gsub(":", "", $1); print $1}' "$CONTRACT_FILE" | sort -u > "$CONTRACT_PATHS"

# Extract literal and template-literal endpoints used by apiRequest(...)
rg -No "apiRequest<[^>]*>\(([^\n]+)" "$NEXT_API_DIR" \
  | sed -E "s/.*apiRequest<[^>]*>\((.*)$/\1/" \
  | rg -No "['\"]/admin[^'\"]+['\"]|\`/admin[^\`]+\`" \
  | sed -E "s/^['\"\`]//; s/['\"\`]$//" > "$NEXT_PATHS_RAW"

sed -E \
  -e 's/\$\{[^}]+\}/{id}/g' \
  -e 's/\?.*$//' \
  "$NEXT_PATHS_RAW" | sort -u > "$NEXT_PATHS"

comm -23 "$NEXT_PATHS" "$CONTRACT_PATHS" > "$UNEXPECTED" || true

if [[ -s "$UNEXPECTED" ]]; then
  echo "[contract-drift] detected unexpected next api paths (not in openapi baseline):"
  cat "$UNEXPECTED" | sed 's/^/  - /'
  echo
  echo "[contract-drift] action: register gap + block release until resolved/approved."
  exit 1
fi

echo "[contract-drift] PASS: no contract drift detected."
