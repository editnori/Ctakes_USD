#!/usr/bin/env bash
set -euo pipefail

# Consolidate per-shard outputs into top-level folders for a single run parent dir.
#
# Moves files from:
#   parent/shard_*/{xmi,bsv_table,csv_table,html_table,cui_list,cui_count}
# into:
#   parent/{xmi,bsv_table,csv_table,html_table,cui_list,cui_count}
#
# By default, removes shard_* and shards/ after consolidation.
# Also ensures parent has a combined run.log and a copy of the tuned .piper
# used by shards if available (without overwriting existing files).
#
# Usage:
#   scripts/consolidate_shards.sh -p <run_parent_dir> [--keep-shards]

PARENT=""; KEEP=0; MAKE_WB=0; WB_PATH=""; WB_MODE="csv"
while [[ $# -gt 0 ]]; do
  case "$1" in
    -p|--parent) PARENT="$2"; shift 2;;
    --keep-shards) KEEP=1; shift 1;;
    -W|--workbook)
      MAKE_WB=1;
      # Optional path argument if provided and not another flag
      if [[ $# -ge 2 && ! "$2" =~ ^- ]]; then WB_PATH="$2"; shift 2; else shift 1; fi;;
    --wb-mode) WB_MODE="$2"; shift 2;;
    *) echo "Unknown arg: $1" >&2; exit 2;;
  esac
done

[[ -z "$PARENT" ]] && { echo "-p|--parent is required (run parent directory)" >&2; exit 2; }
[[ -d "$PARENT" ]] || { echo "Parent dir not found: $PARENT" >&2; exit 2; }

echo "[consolidate] Parent: $PARENT"

# Canonical output types (now include tokens for report Tokens sheet and per-doc concepts CSVs)
types=(xmi bsv_table csv_table csv_table_concepts html_table cui_list cui_count bsv_tokens)

shopt -s nullglob
for t in "${types[@]}"; do
  mkdir -p "$PARENT/$t"
  moved=0
  for sh in "$PARENT"/shard_*; do
    [[ -d "$sh/$t" ]] || continue
    # Move files, don't overwrite existing
    for f in "$sh/$t"/*; do
      [[ -e "$f" ]] || continue
      bn="$(basename "$f")"
      if [[ ! -e "$PARENT/$t/$bn" ]]; then
        mv "$f" "$PARENT/$t/$bn"
        moved=$((moved+1))
      fi
    done
  done
  if [[ "$moved" -gt 0 ]]; then
    echo "[consolidate] Moved $moved files into $t/"
  fi
done
shopt -u nullglob

# Sweep stray cuicounts anywhere under shard_* (not only shard_*/cui_count)
mkdir -p "$PARENT/cui_count"
stray_moved=0
while IFS= read -r -d '' f; do
  bn="$(basename "$f")"
  if [[ ! -e "$PARENT/cui_count/$bn" ]]; then
    mv "$f" "$PARENT/cui_count/$bn"
    stray_moved=$((stray_moved+1))
  fi
done < <(find "$PARENT" -maxdepth 2 -type f -name '*.cuicount.bsv' -print0)
if [[ "$stray_moved" -gt 0 ]]; then
  echo "[consolidate] Moved $stray_moved stray *.cuicount.bsv files into cui_count/"
fi

# Ensure a combined run.log exists at parent level (do not overwrite non-empty)
if [[ ! -s "$PARENT/run.log" ]]; then
  combined="$PARENT/run.log"
  : > "$combined" || true
  for sh in $(ls -1d "$PARENT"/shard_* 2>/dev/null | sort); do
    if [[ -f "$sh/run.log" ]]; then cat "$sh/run.log" >> "$combined"; fi
  done
  if [[ -s "$combined" ]]; then
    echo "[consolidate] Wrote combined run.log"
  else
    rm -f "$combined" 2>/dev/null || true
  fi
fi

# Ensure a copy of the tuned piper exists at parent level
if ! ls -1 "$PARENT"/*.piper >/dev/null 2>&1; then
  for sh in $(ls -1d "$PARENT"/shard_* 2>/dev/null | sort); do
    cand=$(ls -1 "$sh"/*.piper 2>/dev/null | head -n 1 || true)
    if [[ -n "$cand" && -f "$cand" ]]; then
      cp -f "$cand" "$PARENT/$(basename "$cand")"
      echo "[consolidate] Copied piper to parent: $(basename "$cand")"
      break
    fi
  done
fi

# Optionally remove the now-empty shard dirs and shards/ input links (and any pending_* scratch dirs)
if [[ "$KEEP" -eq 0 ]]; then
  echo "[consolidate] Removing shard_* and shards/ directories and pending_* scratch"
  rm -rf "$PARENT"/shard_* "$PARENT"/shards "$PARENT"/pending_* 2>/dev/null || true
fi

echo "[consolidate] Done: $PARENT"

# Optionally build a consolidated workbook (XLSX or XML), default mode=csv (fast)
if [[ "$MAKE_WB" -eq 1 ]]; then
  BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
  if [[ -z "$WB_PATH" ]]; then
    base="$(basename "$PARENT")"; ts="$(date +%Y%m%d-%H%M%S)"
    WB_PATH="$PARENT/ctakes-report-${base}-${ts}.xlsx"
  fi
  echo "[consolidate] Building workbook ($WB_MODE): $WB_PATH"
  bash "$BASE_DIR/scripts/build_xlsx_report.sh" -o "$PARENT" -w "$WB_PATH" -M "$WB_MODE" || true
  echo "[consolidate] Workbook created: $WB_PATH"
fi

# Build aggregated CUI counts workbook/CSVs (best-effort)
BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
if command -v python3 >/dev/null 2>&1; then
  COLORS="S_core=#00A3E0;S_core_rel=#4F81BD;S_core_temp=#9BBB59;S_core_temp_coref=#2C3E50;D_core_rel=#C0504D;D_core_temp=#8064A2;D_core_temp_coref=#4BACC6;WSD_Compare=#8E44AD;TsSectionedTemporalCoref=#E07A00"
  OUT_BASE="$PARENT/cui_count/cui_counts"
  python3 "$BASE_DIR/scripts/consolidate_cuicount.py" \
    --input-root "$PARENT" \
    --out-base "$OUT_BASE" \
    --derive-pipeline-from-path \
    --pipeline-colors "$COLORS" \
    --include-per-doc \
    || echo "[consolidate] WARN: cuicount aggregation step failed; continuing"
fi
