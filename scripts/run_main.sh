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
# Detect runner and feature-gate flags for compatibility with older versions
RUNNER="$BASE_DIR/scripts/run_compare_cluster.sh"
supports_flag() {
  local f="$1"; [[ -f "$RUNNER" ]] && grep -q -- " $f)" "$RUNNER" 2>/dev/null
}
# Default to minimal artifacts for speed unless explicitly overridden by env
# MAIN_WITH_FULL=1 will keep default writers
EXTRA_FLAGS=()
# Performance defaults (override via env): enable tmpfs, async writer, and progress
MAIN_TMPFS_WRITES=${MAIN_TMPFS_WRITES:-1}
MAIN_WRITER_ASYNC=${MAIN_WRITER_ASYNC:-1}
MAIN_WRITER_THREADS=${MAIN_WRITER_THREADS:-4}
MAIN_WRITER_BUFFER_KB=${MAIN_WRITER_BUFFER_KB:-256}
MAIN_PROGRESS=${MAIN_PROGRESS:-1}
MAIN_PROGRESS_EVERY=${MAIN_PROGRESS_EVERY:-10}
# Defaults: Minimal outputs (concepts + timing) and safer relations
if [[ "${MAIN_WITH_FULL:-0}" -ne 1 ]]; then
  EXTRA_FLAGS+=( --csv-only )
  # Default: relations-lite unless explicitly disabled; fallback to --skip-relations if lite unsupported
  if [[ "${MAIN_RELATIONS_LITE:-1}" -eq 1 ]]; then
    if supports_flag "--relations-lite"; then EXTRA_FLAGS+=( --relations-lite );
    elif supports_flag "--skip-relations"; then EXTRA_FLAGS+=( --skip-relations ); fi
  fi
  # Default: concepts-only (drop wide semantic csv_table) if supported
  if [[ "${MAIN_CONCEPTS_ONLY:-1}" -eq 1 && $(supports_flag "--concepts-only" && echo 1 || echo 0) -eq 1 ]]; then
    EXTRA_FLAGS+=( --concepts-only )
  fi
  # Default: no CUI list/count if supported
  if [[ "${MAIN_NO_CUI_LIST:-1}" -eq 1 && $(supports_flag "--no-cui-list" && echo 1 || echo 0) -eq 1 ]]; then EXTRA_FLAGS+=( --no-cui-list ); fi
  if [[ "${MAIN_NO_CUI_COUNT:-1}" -eq 1 && $(supports_flag "--no-cui-count" && echo 1 || echo 0) -eq 1 ]]; then EXTRA_FLAGS+=( --no-cui-count ); fi
  # Default: produce single combined table and remove per-doc CSVs if supported
  if [[ "${MAIN_SINGLE_TABLE_ONLY:-1}" -eq 1 && $(supports_flag "--single-table-only" && echo 1 || echo 0) -eq 1 ]]; then
    EXTRA_FLAGS+=( --single-table-only )
  elif [[ "${MAIN_SINGLE_TABLE:-0}" -eq 1 && $(supports_flag "--single-table" && echo 1 || echo 0) -eq 1 ]]; then
    EXTRA_FLAGS+=( --single-table )
  fi
else
  # Full mode: allow opting back into safer relations if requested
  if [[ "${MAIN_RELATIONS_LITE:-0}" -eq 1 ]]; then
    if supports_flag "--relations-lite"; then EXTRA_FLAGS+=( --relations-lite );
    elif supports_flag "--skip-relations"; then EXTRA_FLAGS+=( --skip-relations ); fi
  fi
fi
# Defaults: tmpfs staging if supported
if [[ "$MAIN_TMPFS_WRITES" -eq 1 && $(supports_flag "--tmpfs-writes" && echo 1 || echo 0) -eq 1 ]]; then
  EXTRA_FLAGS+=( --tmpfs-writes )
fi
# Defaults: async ClinicalConceptCsvWriter tuning if supported
if [[ "$MAIN_WRITER_ASYNC" -eq 1 ]]; then
  if [[ $(supports_flag "--writer-async" && echo 1 || echo 0) -eq 1 ]]; then EXTRA_FLAGS+=( --writer-async ); fi
  if [[ -n "${MAIN_WRITER_THREADS}" && $(supports_flag "--writer-threads" && echo 1 || echo 0) -eq 1 ]]; then EXTRA_FLAGS+=( --writer-threads "${MAIN_WRITER_THREADS}" ); fi
  if [[ -n "${MAIN_WRITER_BUFFER_KB}" && $(supports_flag "--writer-buffer-kb" && echo 1 || echo 0) -eq 1 ]]; then EXTRA_FLAGS+=( --writer-buffer-kb "${MAIN_WRITER_BUFFER_KB}" ); fi
fi
# Defaults: progress logging if supported
if [[ "$MAIN_PROGRESS" -eq 1 && $(supports_flag "--progress" && echo 1 || echo 0) -eq 1 ]]; then
  EXTRA_FLAGS+=( --progress )
  if [[ -n "${MAIN_PROGRESS_EVERY}" && $(supports_flag "--progress-every" && echo 1 || echo 0) -eq 1 ]]; then EXTRA_FLAGS+=( --progress-every "${MAIN_PROGRESS_EVERY}" ); fi
fi
# Reduce noisy XMI serializer logs if XMI is enabled later
export XMI_LOG_LEVEL=${XMI_LOG_LEVEL:-error}
exec bash "$BASE_DIR/scripts/run_compare_cluster.sh" --only "$ONLY_SET" "${EXTRA_FLAGS[@]}" "$@"
