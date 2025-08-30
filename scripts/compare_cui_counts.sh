#!/usr/bin/env bash
set -euo pipefail

# Compare aggregated CUI counts between two runs or pipeline output dirs.
# Usage: scripts/compare_cui_counts.sh -a <runA_dir> -b <runB_dir> [-o <diff.csv>]

A=""; B=""; OUT=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    -a) A="$2"; shift 2;;
    -b) B="$2"; shift 2;;
    -o) OUT="$2"; shift 2;;
    *) echo "Unknown arg: $1" >&2; exit 2;;
  esac
done

[[ -z "$A" || -z "$B" ]] && { echo "-a and -b are required" >&2; exit 2; }
[[ -d "$A" ]] || { echo "Not found: $A" >&2; exit 2; }
[[ -d "$B" ]] || { echo "Not found: $B" >&2; exit 2; }

sum_counts() {
  local dir="$1"; shift || true
  # Accept a pipeline run dir directly or a compare parent: pick first plausible pipeline subdir
  local cc="$dir/cui_count";
  if [[ ! -d "$cc" ]]; then
    for sub in "$dir"/*; do
      [[ -d "$sub/cui_count" ]] || continue
      cc="$sub/cui_count"; break
    done
  fi
  declare -A map=()
  shopt -s nullglob
  for f in "$cc"/*.bsv; do
    while IFS='|' read -r key cnt _rest; do
      [[ -z "$key" ]] && continue
      key="${key#-}" # strip leading '-' for negated bucket
      cnt=${cnt//[$' \t\r\n']}
      [[ -z "$cnt" ]] && continue
      if [[ -n "${map[$key]:-}" ]]; then map[$key]=$(( map[$key] + cnt )); else map[$key]=$cnt; fi
    done < "$f"
  done
  for k in "${!map[@]}"; do echo "$k,${map[$k]}"; done | sort -t',' -k1,1
}

tmpA="$(mktemp)"; tmpB="$(mktemp)"
sum_counts "$A" > "$tmpA"
sum_counts "$B" > "$tmpB"

OUT=${OUT:-"cui_diff.csv"}
echo "CUI,CountA,CountB,Delta" > "$OUT"
join -t',' -a1 -a2 -e0 -o auto "$tmpA" "$tmpB" | awk -F',' '{ a=$2; b=$3; d=(b-a); print $1","a","b","d }' | sort -t',' -k4,4nr >> "$OUT"
echo "Wrote: $OUT"

