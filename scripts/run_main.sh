#!/usr/bin/env bash
set -euo pipefail

# Focused main run: Single-pass Sectioned core + Relation + Smoking Status.
# Wraps scripts/run_compare_cluster.sh with a constrained --only set (S_core_rel_smoke).
#
# Usage:
#   scripts/run_main.sh -i <input_root> -o <output_base> [--reports] [--autoscale]
#   scripts/run_main.sh --help

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"

if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
  cat <<EOF
Focused Main Run
Runs only these pipelines:
  - S_core_rel_smoke -> TsSectionedCoreRelSmoke_WSD_Compare.piper

Examples:
  bash scripts/run_main.sh -i inputs/SD5000_1 -o outputs/main --reports --autoscale
  RUNNERS=24 THREADS=6 XMX_MB=6144 bash scripts/run_main.sh -i inputs/SD5000_1 -o outputs/main --reports
EOF
  exit 0
fi

ONLY_SET="S_core_rel_smoke"
exec bash "$BASE_DIR/scripts/run_compare_cluster.sh" --only "$ONLY_SET" "$@"
