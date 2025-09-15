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

PARENT=""; KEEP=0; MAKE_WB=0; WB_PATH=""; WB_MODE="csv"; SINGLE_TABLE=0; SINGLE_TABLE_ONLY=0
while [[ $# -gt 0 ]]; do
  case "$1" in
    -p|--parent) PARENT="$2"; shift 2;;
    --keep-shards) KEEP=1; shift 1;;
    --single-table) SINGLE_TABLE=1; shift 1;;
    --single-table-only) SINGLE_TABLE=1; SINGLE_TABLE_ONLY=1; shift 1;;
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
  moved=0
  need_dir=0
  for sh in "$PARENT"/shard_*; do
    [[ -d "$sh/$t" ]] || continue
    if compgen -G "$sh/$t/*" >/dev/null 2>&1; then need_dir=1; break; fi
  done
  [[ "$need_dir" -eq 1 ]] || continue
  mkdir -p "$PARENT/$t"
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

# Sweep stray *.cuicount.bsv anywhere under immediate shards into parent/cui_count
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
    # Deduplicate repeated Build Version/Date lines (keep first occurrence)
    tmp_combined="$(mktemp)"
    awk '{ if ($0 ~ /\] Build (Version|Date):/) { if (!seen[$0]++) print; } else { print } }' "$combined" > "$tmp_combined" && mv "$tmp_combined" "$combined"
    echo "[consolidate] Wrote combined run.log (dedup Build Version/Date lines)"
  else
    rm -f "$combined" 2>/dev/null || true
  fi
fi

# Build timing CSV from combined run.log to accelerate reports
if [[ -s "$PARENT/run.log" ]]; then
  tdir="$PARENT/timing_csv"; mkdir -p "$tdir"
  tcsv="$tdir/timing.csv"
  if [[ ! -s "$tcsv" ]]; then
    echo "[consolidate] Building timing CSV from run.log"
    {
      echo "Document,StartMillis,EndMillis,DurationMillis,DurationSeconds"
      awk -F"\t" '/\[timing\] END\t/ { doc=$2; start=$3; end=$4; dur=$5; if (dur=="" && start!="" && end!="") { dur=end-start } if (doc!="") { printf "%s,%s,%s,%s,%.3f\n", doc, start, end, dur, (dur==""?0:dur/1000.0) } }' "$PARENT/run.log"
    } > "$tcsv" || true
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

# Build a single combined concepts CSV if requested
if [[ "$SINGLE_TABLE" -eq 1 ]]; then
  src_dir=""; out_file="$PARENT/concepts_all.csv"
  if [[ -d "$PARENT/csv_table_concepts" ]] && ls -1 "$PARENT/csv_table_concepts"/*.CSV >/dev/null 2>&1; then
    src_dir="$PARENT/csv_table_concepts"
  elif [[ -d "$PARENT/csv_table" ]] && ls -1 "$PARENT/csv_table"/*.CSV >/dev/null 2>&1; then
    src_dir="$PARENT/csv_table"
  fi
  if [[ -n "$src_dir" ]]; then
    echo "[consolidate] Writing single concepts table: $out_file"
    {
      # Take header from the first CSV, write once
      first="$(ls -1 "$src_dir"/*.CSV | head -n 1)"
      head -n 1 "$first"
      for f in $(ls -1 "$src_dir"/*.CSV); do
        tail -n +2 "$f"
      done
    } > "$out_file" || true
    if [[ "$SINGLE_TABLE_ONLY" -eq 1 ]]; then
      echo "[consolidate] SINGLE_TABLE_ONLY set: removing per-doc CSVs under $src_dir"
      rm -f "$src_dir"/*.CSV 2>/dev/null || true
      # Remove now-empty directory if no files remain
      rmdir "$src_dir" 2>/dev/null || true
    fi
  else
    echo "[consolidate] SINGLE_TABLE requested but no source CSVs found under csv_table_concepts/ or csv_table/"
  fi
fi

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

# Dedicated CUI-count workbook is now provided within the Java-built per-pipeline workbooks (CuiCounts sheet).

# Write a lightweight metrics.json for fast report path (docCount, mentionCount, distinctCuiCount)
{
  xmi_docs=0
  if [[ -d "$PARENT/xmi" ]]; then xmi_docs=$(find "$PARENT/xmi" -type f -name '*.xmi' | wc -l | tr -d ' '); fi
  # Sum cui_count totals
  mentions=0
  distinct=$(mktemp)
  if [[ -d "$PARENT/cui_count" ]]; then
    while IFS='|' read -r key cnt _rest; do
      [[ -z "$key" ]] && continue
      key="${key#-}"
      echo "$key" >> "$distinct"
      cnt=${cnt//[$' \t\r\n']}
      [[ -n "$cnt" ]] && mentions=$(( mentions + cnt ))
    done < <(cat "$PARENT"/cui_count/*.bsv 2>/dev/null || true)
  fi
  distinct_count=$(sort -u "$distinct" 2>/dev/null | wc -l | tr -d ' ')
  rm -f "$distinct" 2>/dev/null || true
  cat > "$PARENT/metrics.json" <<EOF
{
  "docCount": $xmi_docs,
  "mentionCount": $mentions,
  "distinctCuiCount": $distinct_count
}
EOF
} || true
