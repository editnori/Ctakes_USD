#!/usr/bin/env bash
set -euo pipefail

# Build per-run timing CSV from a combined run.log or from shard logs.
# Usage: scripts/build_timeline_csv.sh -p <run_parent_dir> [-o <out_csv>]

PARENT=""; OUT_CSV=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    -p|--parent) PARENT="$2"; shift 2;;
    -o|--out) OUT_CSV="$2"; shift 2;;
    *) echo "Unknown arg: $1" >&2; exit 2;;
  esac
done

[[ -z "$PARENT" ]] && { echo "-p|--parent is required" >&2; exit 2; }
[[ -d "$PARENT" ]] || { echo "Parent dir not found: $PARENT" >&2; exit 2; }

tdir="$PARENT/timing_csv"; mkdir -p "$tdir"
if [[ -z "$OUT_CSV" ]]; then OUT_CSV="$tdir/timing.csv"; fi

src_log="$PARENT/run.log"
if [[ ! -s "$src_log" ]]; then
  # Compose a temporary combined log
  tmp="$(mktemp)"
  : > "$tmp"
  for sh in $(ls -1d "$PARENT"/shard_* 2>/dev/null | sort); do
    if [[ -f "$sh/run.log" ]]; then cat "$sh/run.log" >> "$tmp"; fi
  done
  src_log="$tmp"
fi

echo "Document,StartMillis,EndMillis,DurationMillis,DurationSeconds" > "$OUT_CSV"
awk -F"\t" '/\[timing\] END\t/ { doc=$2; start=$3; end=$4; dur=$5; if (dur=="" && start!="" && end!="") { dur=end-start } if (doc!="") { printf "%s,%s,%s,%s,%.3f\n", doc, start, end, dur, (dur==""?0:dur/1000.0) } }' "$src_log" >> "$OUT_CSV" || true
echo "Wrote: $OUT_CSV"

