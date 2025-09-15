#!/usr/bin/env bash
set -euo pipefail

# Print a quick status of what a compare run would do, without running it.
# Shows: inputs, pipelines planned, env (RUNNERS/THREADS/XMX), dictionary + shared cache,
# output locations, and report mode.
#
# Usage:
#   scripts/status.sh -i <input_root_or_dir> [-o <output_base>] [--only "S_core ..."] \
#                     [-n RUNNERS] [-t THREADS] [-m XMX_MB] [--seed VAL]

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
CTAKES_HOME="${CTAKES_HOME:-$BASE_DIR/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0}"

IN=""; OUT=""; RUNNERS="${RUNNERS:-16}"; THREADS="${THREADS:-6}"; XMX_MB="${XMX_MB:-6144}"; ONLY=""; SEED="${SEED:-}"
DICT_SHARED="${DICT_SHARED:-1}"; DICT_SHARED_PATH="${DICT_SHARED_PATH:-/dev/shm}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    -i|--in) IN="$2"; shift 2;;
    -o|--out) OUT="$2"; shift 2;;
    --only) ONLY="$2"; shift 2;;
    -n|--runners) RUNNERS="$2"; shift 2;;
    -t|--threads) THREADS="$2"; shift 2;;
    -m|--xmx) XMX_MB="$2"; shift 2;;
    --seed) SEED="$2"; shift 2;;
    *) echo "Unknown arg: $1" >&2; exit 2;;
  esac
done

[[ -z "$IN" ]] && { echo "-i|--in is required" >&2; exit 2; }
OUT="${OUT:-$BASE_DIR/outputs/compare}"

# Detect Temporal model availability (same heuristic as cluster runner)
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

# Pipelines
declare -A SETS=(
  [S_core]="$BASE_DIR/pipelines/compare/TsSectionedFast_WSD_Compare.piper"
  [S_core_rel]="$BASE_DIR/pipelines/compare/TsSectionedRelation_WSD_Compare.piper"
  [S_core_smoke]="$BASE_DIR/pipelines/compare/TsSectionedSmoking_WSD_Compare.piper"
  [S_core_temp]="$BASE_DIR/pipelines/compare/TsSectionedTemporal_WSD_Compare.piper"
  [S_core_temp_coref]="$BASE_DIR/pipelines/compare/TsSectionedTemporalCoref_WSD_Compare.piper"
  [S_core_temp_coref_smoke]="$BASE_DIR/pipelines/compare/TsSectionedTemporalCoref_WSD_Smoking_Compare.piper"
  [D_core_rel]="$BASE_DIR/pipelines/compare/TsDefaultRelation_WSD_Compare.piper"
  [D_core_temp]="$BASE_DIR/pipelines/compare/TsDefaultTemporal_WSD_Compare.piper"
  [D_core_temp_coref]="$BASE_DIR/pipelines/compare/TsDefaultTemporalCoref_WSD_Compare.piper"
  [D_core_temp_coref_smoke]="$BASE_DIR/pipelines/compare/TsDefaultTemporalCoref_WSD_Smoking_Compare.piper"
  [D_core_coref]="$BASE_DIR/pipelines/compare/TsDefaultCoref_WSD_Compare.piper"
)

declare -a keys
if [[ -n "$ONLY" ]]; then
  # shellcheck disable=SC2206
  keys=($ONLY)
else
  keys=(S_core S_core_rel D_core_rel D_core_coref)
  if [[ "$HAS_TEMP_MODELS" -eq 1 ]]; then
    keys+=(S_core_temp S_core_temp_coref D_core_temp D_core_temp_coref S_core_temp_coref_smoke D_core_temp_coref_smoke)
  fi
fi

# Validate and collect present pipelines
declare -a planned=()
for k in "${keys[@]}"; do
  p="${SETS[$k]:-}"
  [[ -n "$p" && -f "$p" ]] && planned+=("$k|$p")
done

# Count input notes
count_txt() { find "$1" -type f -name '*.txt' 2>/dev/null | wc -l | awk '{print $1}'; }
DOCS_TOTAL=0
GROUPS=()
if [[ -d "$IN" ]]; then
  shopt -s nullglob
  for d in "$IN"/*; do
    [[ -d "$d" ]] || continue
    if find "$d" -type f -name '*.txt' | head -n 1 | grep -q .; then
      n=$(count_txt "$d"); DOCS_TOTAL=$((DOCS_TOTAL + n)); GROUPS+=("$(basename "$d"):$n")
    fi
  done
  shopt -u nullglob
  if [[ ${#GROUPS[@]} -eq 0 ]]; then DOCS_TOTAL=$(count_txt "$IN"); fi
else
  echo "Input is not a directory: $IN" >&2; exit 2
fi

# Dictionary info
PROPS="$BASE_DIR/docs/builder_full_clinical.properties"
DICT_NAME=$(awk -F= '/^\s*dictionary.name\s*=/{print $2}' "$PROPS" | tr -d '\r' | xargs)
[[ -n "$DICT_NAME" ]] || DICT_NAME="FullClinical_AllTUIs"
DICT_XML="$CTAKES_HOME/resources/org/apache/ctakes/dictionary/lookup/fast/${DICT_NAME}.xml"
SHARED_PREFIX="${DICT_SHARED_PATH%/}/${DICT_NAME}_shared"
HAS_SHARED="no"; [[ -f "${SHARED_PREFIX}.script" && -f "${SHARED_PREFIX}.properties" ]] && HAS_SHARED="yes"

echo "== Compare Run Status =="
echo "Input Dir     : $IN"
if [[ ${#GROUPS[@]} -gt 0 ]]; then
  echo "Input Groups  : ${#GROUPS[@]}"
  for g in "${GROUPS[@]}"; do echo "  - $g"; done
fi
echo "Documents     : $DOCS_TOTAL"
echo "Output Base   : ${OUT}"
echo "Reports       : per-pipeline + parent (CSV mode)"
echo "Pipelines     : ${#planned[@]} planned (temporal_models=$HAS_TEMP_MODELS)"
for kv in "${planned[@]}"; do k="${kv%%|*}"; p="${kv#*|}"; echo "  - $k -> $(basename "$p")"; done
echo "Env           : RUNNERS=$RUNNERS THREADS=$THREADS XMX_MB=$XMX_MB SEED=${SEED:-NA}"
echo "Dictionary    : $DICT_NAME"
echo "  XML         : $DICT_XML $( [[ -f "$DICT_XML" ]] && echo '[OK]' || echo '[MISSING]')"
echo "  Shared DB   : DICT_SHARED=$DICT_SHARED DICT_SHARED_PATH=$DICT_SHARED_PATH"
echo "  Cache Files : ${SHARED_PREFIX}.(script|properties) exists=$HAS_SHARED"
echo "Writers       : XMI + tables + lists + tokens (default)"
echo "Consolidation : moves xmi, bsv_table, csv_table, csv_table_concepts, html_table, cui_list, cui_count, bsv_tokens"
echo "Reports Build : build_xlsx_report.sh -M csv (no XMI parsing)"

echo "\nRun example:"
echo "  export RUNNERS=$RUNNERS THREADS=$THREADS XMX_MB=$XMX_MB SEED=${SEED:-42}"
echo "  export DICT_SHARED=$DICT_SHARED DICT_SHARED_PATH=\"$DICT_SHARED_PATH\""
echo "  bash scripts/run_compare_cluster.sh -i \"$IN\" -o \"$OUT\" --reports"
