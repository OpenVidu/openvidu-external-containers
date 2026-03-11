#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Allow overriding image via env var
export MONGODB_IMAGE="${MONGODB_IMAGE:-openvidu/mongodb:8.0}"

echo "============================================"
echo " MongoDB Test Suite"
echo " Image: ${MONGODB_IMAGE}"
echo "============================================"
echo ""

# Check prerequisites
for cmd in docker bats; do
    if ! command -v "$cmd" &>/dev/null; then
        echo "ERROR: '${cmd}' is required but not found in PATH" >&2
        exit 1
    fi
done

if ! docker compose version &>/dev/null; then
    echo "ERROR: 'docker compose' plugin is required" >&2
    exit 1
fi

# Run all .bats files in order
bats --timing "${SCRIPT_DIR}"/*.bats
