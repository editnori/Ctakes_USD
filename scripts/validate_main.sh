#!/usr/bin/env bash
set -euo pipefail

# Focused validation (~100 notes) using a single-pass main pipeline.
# Wraps scripts/validate_mimic.sh with a constrained --only set (S_core_rel_smoke).
#
# Usage:
#   scripts/validate_main.sh [options]
#   scripts/validate_main.sh --help

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  cat <<EOF
Focused Validation (MIMIC ~100 notes)
Validates only these pipelines:
  - S_core_rel_smoke -> TsSectionedCoreRelSmoke_WSD_Compare.piper

Examples:
  bash scripts/validate_main.sh
  bash scripts/validate_main.sh -i samples/mimic -o outputs/validation_main --runners 4 --threads 4 --xmx 4096
EOF
  exit 0
fi

ONLY_SET="S_core_rel_smoke"
exec bash "$BASE_DIR/scripts/validate_mimic.sh" --only "$ONLY_SET" "$@"
