#!/usr/bin/env bash
set -euo pipefail

# Progress estimator for scripts/run_compare_cluster.sh runs.
# It counts produced files (XMI and other writer outputs) and
# compares against the expected totals based on number of input
# notes and number of pipelines that will run.
#
# Usage:
#   scripts/progress_compare_cluster.sh -i <input_root_or_dir> -o <output_base>
#
# Notes:
# - Uses XMI files as the canonical per-document signal.
# - Optionally considers all writer outputs to compute a second estimate.
# - Assumes the output base is dedicated to a single batch/run to avoid mixing.

IN=""; OUT=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    -i|--in) IN="$2"; shift 2;;
    -o|--out) OUT="$2"; shift 2;;
    *) echo "Unknown arg: $1" >&2; exit 2;;
  esac
done

[[ -z "${IN}" ]] && { echo "-i|--in is required" >&2; exit 2; }
[[ -z "${OUT}" ]] && { echo "-o|--out is required" >&2; exit 2; }
[[ -d "$IN" ]] || { echo "Input not found: $IN" >&2; exit 2; }
[[ -d "$OUT" ]] || { echo "Output base not found: $OUT" >&2; exit 2; }

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
CTAKES_HOME="${CTAKES_HOME:-$BASE_DIR/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0}"

# Determine which pipelines are planned (same logic as run_compare_cluster.sh)
declare -A SETS=(
  [S_core]="$BASE_DIR/pipelines/compare/TsSectionedFast_WSD_Compare.piper"
  [S_core_rel]="$BASE_DIR/pipelines/compare/TsSectionedRelation_WSD_Compare.piper"
  [S_core_temp]="$BASE_DIR/pipelines/compare/TsSectionedTemporal_WSD_Compare.piper"
  [S_core_temp_coref]="$BASE_DIR/pipelines/compare/TsSectionedTemporalCoref_WSD_Compare.piper"
  [S_core_temp_coref_smoke]="$BASE_DIR/pipelines/compare/TsSectionedTemporalCoref_WSD_Smoking_Compare.piper"
  [D_core_rel]="$BASE_DIR/pipelines/compare/TsDefaultRelation_WSD_Compare.piper"
  [D_core_temp]="$BASE_DIR/pipelines/compare/TsDefaultTemporal_WSD_Compare.piper"
  [D_core_temp_coref]="$BASE_DIR/pipelines/compare/TsDefaultTemporalCoref_WSD_Compare.piper"
  [D_core_temp_coref_smoke]="$BASE_DIR/pipelines/compare/TsDefaultTemporalCoref_WSD_Smoking_Compare.piper"
  [D_core_coref]="$BASE_DIR/pipelines/compare/TsDefaultCoref_WSD_Compare.piper"
)
keys=(S_core S_core_rel D_core_rel D_core_coref)

# Detect Temporal model availability (same heuristic as run_compare_cluster.sh)
HAS_TEMP_MODELS=0
if [[ -f "$CTAKES_HOME/resources/org/apache/ctakes/temporal/models/eventannotator/model.jar" ]]; then
  HAS_TEMP_MODELS=1
else
  for J in "$CTAKES_HOME"/lib/*.jar; do
    if jar tf "$J" 2>/dev/null | grep -q "org/apache/ctakes/temporal/models/eventannotator/model.jar"; then
      HAS_TEMP_MODELS=1; break
    fi
  done
fi
if [[ "$HAS_TEMP_MODELS" -eq 1 ]]; then
  keys+=(S_core_temp S_core_temp_coref D_core_temp D_core_temp_coref S_core_temp_coref_smoke D_core_temp_coref_smoke)
fi

# Keep only keys whose piper file exists
declare -a KEYS_PRESENT=()
for k in "${keys[@]}"; do
  p="${SETS[$k]}"; [[ -f "$p" ]] && KEYS_PRESENT+=("$k") || true
done
PIPE_COUNT=${#KEYS_PRESENT[@]}
if [[ "$PIPE_COUNT" -eq 0 ]]; then
  echo "No compare pipelines found. Check repository layout." >&2
  exit 2
fi

# Resolve input groups (same as run script) and count source notes
declare -a INPUT_GROUPS=()
shopt -s nullglob
for d in "$IN"/*; do
  [[ -d "$d" ]] || continue
  if find "$d" -type f -name '*.txt' | head -n 1 | grep -q .; then INPUT_GROUPS+=("$d"); fi
done
shopt -u nullglob
if [[ ${#INPUT_GROUPS[@]} -eq 0 ]]; then INPUT_GROUPS=("$IN"); fi

DOCS_TOTAL=0
for g in "${INPUT_GROUPS[@]}"; do
  c=$(find "$g" -type f -name '*.txt' | wc -l | tr -d ' ')
  DOCS_TOTAL=$(( DOCS_TOTAL + c ))
done

EXPECTED_XMI=$(( DOCS_TOTAL * PIPE_COUNT ))

# Count produced files under shards
CURRENT_XMI=$(find "$OUT" -type f -path '*/shard_*/xmi/*.xmi' | wc -l | tr -d ' ')
# Fallback if shards have been consolidated/removed: count top-level xmi files
if [[ "$CURRENT_XMI" -eq 0 ]]; then
  CURRENT_XMI=$(find "$OUT" -type f -path '*/xmi/*.xmi' | wc -l | tr -d ' ')
fi

# Other writer outputs (per doc per pipeline): 6 more folders + tokens (7 total types)
declare -a TYPES=("bsv_table" "csv_table" "html_table" "cui_list" "cui_count" "bsv_tokens")
CURRENT_ALL=0
for t in "${TYPES[@]}"; do
  ct=$(find "$OUT" -type f -path "*/shard_*/${t}/*" | wc -l | tr -d ' ')
  if [[ "$ct" -eq 0 ]]; then
    ct=$(find "$OUT" -type f -path "*/${t}/*" | wc -l | tr -d ' ')
  fi
  CURRENT_ALL=$(( CURRENT_ALL + ct ))
done
CURRENT_ALL=$(( CURRENT_ALL + CURRENT_XMI ))

EXPECTED_ALL=$(( EXPECTED_XMI * 7 ))

percent() {
  local num=$1; local den=$2
  if [[ "$den" -eq 0 ]]; then echo "0.00"; return; fi
  awk -v n="$num" -v d="$den" 'BEGIN { printf "%.2f", (n*100.0)/d }'
}

PCT_XMI=$(percent "$CURRENT_XMI" "$EXPECTED_XMI")
PCT_ALL=$(percent "$CURRENT_ALL" "$EXPECTED_ALL")

echo "Input notes:        $DOCS_TOTAL"
echo "Pipelines planned:   $PIPE_COUNT (${KEYS_PRESENT[*]})"
echo "Expected XMI total:  $EXPECTED_XMI"
echo "Current XMI count:   $CURRENT_XMI"
echo "Progress (XMI):      ${PCT_XMI}%"
echo
echo "Expected all files:  $EXPECTED_ALL  (7 per doc per pipeline)"
echo "Current all files:   $CURRENT_ALL"
echo "Progress (all types): ${PCT_ALL}%"
