#!/usr/bin/env bash
set -euo pipefail

# Build a combined compare workbook across multiple run directories by
# symlinking their per-pipeline subfolders into a single staging folder,
# then invoking the existing report builder in summary mode.
#
# Usage:
#   scripts/build_multi_run_summary.sh -o <combined_dir> <run_dir1> [run_dir2 ...]
#
# Notes:
# - Each run_dir should be a compare run root that contains per-pipeline
#   subfolders (e.g., S_core, D_core_rel, ...). This script links those
#   subfolders into <combined_dir> with prefixed names <run>__<pipeline>.
# - Workbook is written as <combined_dir>/ctakes-runs-summary-<timestamp>.xlsx

OUT=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    -o|--out) OUT="$2"; shift 2;;
    *) break;;
  esac
done

[[ -z "$OUT" ]] && { echo "-o|--out is required (combined staging dir)" >&2; exit 2; }
[[ $# -lt 1 ]] && { echo "Provide one or more run directories" >&2; exit 2; }

mkdir -p "$OUT"

# Create links for each per-pipeline subdir that looks like a run output
for RUN in "$@"; do
  [[ -d "$RUN" ]] || { echo "WARN: run dir not found: $RUN" >&2; continue; }
  rname=$(basename "${RUN%/}")
  shopt -s nullglob
  for sub in "$RUN"/*; do
    [[ -d "$sub" ]] || continue
    b=$(basename "$sub")
    low="${b,,}"
    # Skip standard leaf dirs and shards
    if [[ "$low" =~ ^(xmi|bsv_table|csv_table|html_table|cui_list|cui_count|bsv_tokens|logs|pending_.*|shards|shard_.*)$ ]]; then
      continue
    fi
    # Detect a plausible pipeline output by xmi presence (direct or sharded)
    if [[ -d "$sub/xmi" ]] || ls -1d "$sub"/shard_*/xmi >/dev/null 2>&1; then
      link="$OUT/${rname}__${b}"
      if ln -s "$sub" "$link" 2>/dev/null; then
        echo "Linked: $link -> $sub"
      else
        # Fallback: create a lightweight dir with a marker file
        mkdir -p "$link"; echo "$sub" > "$link/.source_path"; echo "Copied marker: $link"
      fi
    fi
  done
  shopt -u nullglob
done

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
ts="$(date +%Y%m%d-%H%M%S)"
WB="$OUT/ctakes-runs-summary-${ts}.xlsx"
echo "Building combined workbook: $WB"
bash "$BASE_DIR/scripts/build_xlsx_report.sh" -o "$OUT" -w "$WB" -M summary
echo "Combined workbook: $WB"

