#!/usr/bin/env bash
# DEPRECATED: use scripts/run_compare_cluster.sh instead.
set -euo pipefail
BASE_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
exec bash "$BASE_DIR/scripts/run_compare_cluster.sh" "$@"
