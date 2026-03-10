#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

usage() {
  cat <<'EOF'
Usage:
  bash scripts/release/bump-framework-go-version.sh <framework-go-version> [--tag] [--push]

Examples:
  bash scripts/release/bump-framework-go-version.sh v0.0.4-alpha
  bash scripts/release/bump-framework-go-version.sh v0.0.4-alpha --tag
  bash scripts/release/bump-framework-go-version.sh v0.0.4-alpha --tag --push
EOF
}

if [[ $# -lt 1 ]]; then
  usage
  exit 1
fi

FRAMEWORK_GO_VERSION="$1"
shift || true

if [[ ! "$FRAMEWORK_GO_VERSION" =~ ^v[0-9]+(\.[0-9]+)*([-.][0-9A-Za-z]+)*$ ]]; then
  echo "Invalid version: $FRAMEWORK_GO_VERSION"
  echo "Expected format like: v0.0.4-alpha"
  exit 1
fi

DO_TAG=false
DO_PUSH=false
for arg in "$@"; do
  case "$arg" in
    --tag) DO_TAG=true ;;
    --push) DO_PUSH=true ;;
    *)
      echo "Unknown argument: $arg"
      usage
      exit 1
      ;;
  esac
done

FRAMEWORK_GO_TAG="framework/backend/go/${FRAMEWORK_GO_VERSION}"

echo "[1/4] Update versions in repository files..."
perl -0pi -e "s|github\\.com/ArtisanCloud/PowerXPlugin/framework/backend/go\\s+v[0-9A-Za-z._-]+|github.com/ArtisanCloud/PowerXPlugin/framework/backend/go ${FRAMEWORK_GO_VERSION}|g" \
  "skeleton/backend/go-gin/go.mod"
perl -0pi -e "s|defaultFrameworkVersion\\s*=\\s*\"v[0-9A-Za-z._-]+\"|defaultFrameworkVersion = \"${FRAMEWORK_GO_VERSION}\"|g" \
  "tools/cli/cmd/init.go"

echo "[2/4] Sync scaffold/templates from skeleton..."
npm run sync:templates

echo "[3/4] Summary:"
echo "  - framework-go version: ${FRAMEWORK_GO_VERSION}"
echo "  - framework-go tag:     ${FRAMEWORK_GO_TAG}"
echo "  - updated files:"
echo "      * skeleton/backend/go-gin/go.mod"
echo "      * tools/cli/cmd/init.go"
echo "      * scaffold/templates/** (via sync)"
echo "      * tools/cli/internal/templates/** (via sync)"

if [[ "$DO_TAG" == "true" ]]; then
  echo "[4/4] Create git tag: ${FRAMEWORK_GO_TAG}"
  git tag "${FRAMEWORK_GO_TAG}"
  if [[ "$DO_PUSH" == "true" ]]; then
    echo "       Push git tag to origin..."
    git push origin "${FRAMEWORK_GO_TAG}"
  else
    echo "       Skip push (pass --push to push)."
  fi
else
  echo "[4/4] Skip git tag (pass --tag to create tag)."
fi

echo "Done."
