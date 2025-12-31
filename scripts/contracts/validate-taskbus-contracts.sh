#!/usr/bin/env sh

set -eu

# Validate TaskBus event contracts (topic uniqueness + required meta + basic safety lint).
# Defaults to the 008 contracts file; override via TASKBUS_CONTRACTS env var.

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)

: "${TASKBUS_CONTRACTS:=$ROOT_DIR/specs/008-framework-task-bus/contracts/channel-events.yaml}"

cd "$ROOT_DIR"

make validate-taskbus-contracts TASKBUS_CONTRACTS="$TASKBUS_CONTRACTS"

