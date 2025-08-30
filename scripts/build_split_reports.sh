#!/usr/bin/env bash
set -euo pipefail

# Build per-note-type XLSX reports for a completed run directory by creating
# lightweight filtered views (hardlinks or copies) and calling build_xlsx_report.sh.
#
# Usage:
#   scripts/build_split_reports.sh -p <run_dir> [--mode csv|summary] [--types <TYPE.####[,TYPE.####...]>]
#
# Notes:
# - <run_dir> should be a single pipeline run output folder that already has
#   consolidated subfolders (xmi/, csv_table_concepts/, cui_count/, ...).
# - If <run_dir> is a compare parent (contains multiple pipeline subdirs), this
#   script will auto-detect and iterate those subdirs.
# - We prefer csv mode (no XMI parse). Summary mode is even lighter.

RUN_DIR=""; MODE="csv"; TYPES=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    -p|--path) RUN_DIR="$2"; shift 2;;
    -M|--mode) MODE="$2"; shift 2;;
    --types) TYPES="$2"; shift 2;;
    *) echo "Unknown arg: $1" >&2; exit 2;;
  esac
done

[[ -z "$RUN_DIR" ]] && { echo "-p|--path is required (run directory)" >&2; exit 2; }
[[ -d "$RUN_DIR" ]] || { echo "Run dir not found: $RUN_DIR" >&2; exit 2; }

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
mkdir -p "$RUN_DIR"

is_pipeline_dir() {
  local d="$1"
  [[ -d "$d/xmi" || -d "$d/csv_table_concepts" || -d "$d/cui_count" ]] && return 0 || return 1
}

# Enumerate pipeline subdirs to process
declare -a PIPE_DIRS=()
shopt -s nullglob
if is_pipeline_dir "$RUN_DIR"; then
  PIPE_DIRS+=("$RUN_DIR")
else
  for sub in "$RUN_DIR"/*; do
    [[ -d "$sub" ]] || continue
    if is_pipeline_dir "$sub"; then PIPE_DIRS+=("$sub"); fi
  done
fi
shopt -u nullglob

[[ ${#PIPE_DIRS[@]} -gt 0 ]] || { echo "No pipeline output dirs found under: $RUN_DIR" >&2; exit 1; }

note_types_for_dir() {
  local d="$1"
  # derive note types from csv_table_concepts filenames if present, otherwise from xmi filenames
  local src="$d/csv_table_concepts"; local pat="*.csv"; local field_sep=','
  if [[ ! -d "$src" ]]; then src="$d/xmi"; pat="*.xmi"; fi
  [[ -d "$src" ]] || return 0
  ( 
    shopt -s nullglob
    for f in "$src"/$pat; do
      bn="$(basename "$f")"
      bn="${bn%.csv}"; bn="${bn%.txt.xmi}"
      # extract TYPE.######## from name, or print UNKNOWN
      if [[ "$bn" =~ TYPE\.([0-9]+) ]]; then echo "TYPE.${BASH_REMATCH[1]}"; else echo "TYPE.UNKNOWN"; fi
    done | sort -u
  )
}

ln_or_cp() {
  local src="$1" dest="$2"
  mkdir -p "$(dirname "$dest")"
  if ln "$src" "$dest" 2>/dev/null; then return 0; fi
  cp -f "$src" "$dest"
}

for PD in "${PIPE_DIRS[@]}"; do
  echo "[split] Scanning note types in: $PD"
  declare -a TYPES_LIST=()
  if [[ -n "$TYPES" ]]; then IFS=',' read -r -a TYPES_LIST <<< "$TYPES"; else mapfile -t TYPES_LIST < <(note_types_for_dir "$PD"); fi
  [[ ${#TYPES_LIST[@]} -gt 0 ]] || { echo "[split] No types detected in $PD; skipping"; continue; }

  for T in "${TYPES_LIST[@]}"; do
    tshort="${T#TYPE.}"
    [[ -z "$tshort" ]] && tshort="UNKNOWN"
    stage="$PD/split_note_type/$T"
    rm -rf "$stage" 2>/dev/null || true
    mkdir -p "$stage"

    echo "[split]   Building view for $T"
    # Link/copy files matching type from known subdirs
    for sub in xmi csv_table csv_table_concepts html_table bsv_table bsv_tokens cui_list cui_count; do
      src="$PD/$sub"; [[ -d "$src" ]] || continue
      shopt -s nullglob
      for f in "$src"/*; do
        bn="$(basename "$f")"; name="$bn"
        name="${name%.csv}"; name="${name%.HTML}"; name="${name%.BSV}"; name="${name%.bsv}"; name="${name%.txt.xmi}"
        if [[ "$name" == *"$T"* ]]; then
          ln_or_cp "$f" "$stage/$sub/$bn"
        fi
      done
      shopt -u nullglob
    done
    # Now build workbook for this type
    wb_name="ctakes-$(basename "$PD")-${T}.xlsx"
    bash "$BASE_DIR/scripts/build_xlsx_report.sh" -o "$stage" -w "$PD/$wb_name" -M "$MODE" || true
    echo "[split]   Wrote: $PD/$wb_name"
  done
done

echo "[split] Done."

