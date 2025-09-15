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
# Default to minimal artifacts for speed unless explicitly overridden by env
# MAIN_WITH_FULL=1 will keep default writers
EXTRA_FLAGS=()
# Defaults: Minimal outputs (concepts + timing) and safer relations
if [[ "${MAIN_WITH_FULL:-0}" -ne 1 ]]; then
  EXTRA_FLAGS+=( --csv-only )
  # Default: relations-lite unless explicitly disabled
  if [[ "${MAIN_RELATIONS_LITE:-1}" -eq 1 ]]; then EXTRA_FLAGS+=( --relations-lite ); fi
  # Default: concepts-only (drop wide semantic csv_table)
  if [[ "${MAIN_CONCEPTS_ONLY:-1}" -eq 1 ]]; then EXTRA_FLAGS+=( --concepts-only ); fi
  # Default: no CUI list/count
  if [[ "${MAIN_NO_CUI_LIST:-1}" -eq 1 ]]; then EXTRA_FLAGS+=( --no-cui-list ); fi
  if [[ "${MAIN_NO_CUI_COUNT:-1}" -eq 1 ]]; then EXTRA_FLAGS+=( --no-cui-count ); fi
  # Default: produce single combined table and remove per-doc CSVs
  if [[ "${MAIN_SINGLE_TABLE_ONLY:-1}" -eq 1 ]]; then
    EXTRA_FLAGS+=( --single-table-only )
  elif [[ "${MAIN_SINGLE_TABLE:-0}" -eq 1 ]]; then
    EXTRA_FLAGS+=( --single-table )
  fi
else
  # Full mode: allow opting back into safer relations if requested
  if [[ "${MAIN_RELATIONS_LITE:-0}" -eq 1 ]]; then EXTRA_FLAGS+=( --relations-lite ); fi
fi
# Reduce noisy XMI serializer logs if XMI is enabled later
export XMI_LOG_LEVEL=${XMI_LOG_LEVEL:-error}
exec bash "$BASE_DIR/scripts/run_compare_cluster.sh" --only "$ONLY_SET" "${EXTRA_FLAGS[@]}" "$@"
