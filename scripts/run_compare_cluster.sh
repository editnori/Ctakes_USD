#!/usr/bin/env bash
set -euo pipefail

# Parallel compare runner for large corpora.
# - Shards inputs across N runners per pipeline using hardlinks
# - Uses offline dictionary (HSQL DB copied to /dev/shm) per runner
# - Runs the same pipelines as scripts/run_compare_smoke.sh
# - Generates short-named summary workbook per pipeline/group if --reports is set
# - Post-process: consolidates shard_* outputs into top-level folders (xmi, bsv_table, csv_table, html_table, cui_list, cui_count)
#                 before building any reports (can be disabled via --no-consolidate)
# - Optional: build enhanced per-document CSVs (Clinical Concepts columns) only when --enhanced-csv is set
#
# Usage:
#   scripts/run_compare_cluster.sh -i <input_root_or_dir> -o <output_base> \
#     [-n RUNNERS] [-m XMX_MB] [-t THREADS] [--reports|--reports-async] [--no-consolidate|--keep-shards]

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
CTAKES_HOME="${CTAKES_HOME:-$BASE_DIR/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0}"
JAVA_CP="$CTAKES_HOME/desc:$CTAKES_HOME/resources:$CTAKES_HOME/config:$CTAKES_HOME/config/*:$CTAKES_HOME/lib/*:$BASE_DIR/.build_tools"

IN=""; OUT=""; RUNNERS="${RUNNERS:-16}"; XMX_MB="${XMX_MB:-6144}"; THREADS="${THREADS:-6}"; MAKE_REPORTS=0
PARENT_DIR=""; RESUME=0; ONLY=""; SKIP_PARENT=0; CONSOLIDATE=1; KEEP_SHARDS=0
declare -a REPORT_PIDS=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    -i|--in) IN="$2"; shift 2;;
    -o|--out) OUT="$2"; shift 2;;
    -n|--runners) RUNNERS="$2"; shift 2;;
    -m|--xmx) XMX_MB="$2"; shift 2;;
    -t|--threads) THREADS="$2"; shift 2;;
    --reports) MAKE_REPORTS=1; shift 1;;
    --reports-async) MAKE_REPORTS=2; shift 1;;
    --parent) PARENT_DIR="$2"; shift 2;;
    --resume) RESUME=1; shift 1;;
    --only) ONLY="$2"; shift 2;;
    --no-parent-report) SKIP_PARENT=1; shift 1;;
    --no-consolidate) CONSOLIDATE=0; shift 1;;
    --keep-shards) KEEP_SHARDS=1; shift 1;;
    *) echo "Unknown arg: $1" >&2; exit 2;;
  esac
done

[[ -z "$IN" ]] && { echo "-i|--in is required" >&2; exit 2; }
OUT="${OUT:-$BASE_DIR/outputs/compare_cluster}"
mkdir -p "$OUT"

# Detect Temporal models
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

# Compile local tools (WSD + wrappers), skip Jupyter checkpoints
find "$BASE_DIR/tools" -type f -name "*.java" ! -path "*/.ipynb_checkpoints/*" -print0 | \
  xargs -0 javac -cp "$JAVA_CP" -d "$BASE_DIR/.build_tools"

# Dictionary
PROPS="$BASE_DIR/docs/builder_full_clinical.properties"
DICT_NAME=$(awk -F= '/^\s*dictionary.name\s*=/{print $2}' "$PROPS" | tr -d '\r' | xargs)
if [[ -z "$DICT_NAME" ]]; then DICT_NAME="FullClinical_AllTUIs"; fi
DICT_XML="$CTAKES_HOME/resources/org/apache/ctakes/dictionary/lookup/fast/${DICT_NAME}.xml"
SRC_DB_DIR="$CTAKES_HOME/resources/org/apache/ctakes/dictionary/lookup/fast/$DICT_NAME"
[[ -f "$DICT_XML" ]] || { echo "Dictionary XML not found: $DICT_XML" >&2; exit 1; }

short_name() { local p="$1"; p="${p%/}"; basename "$p" | tr ' ' '_' | cut -c1-40; }

# Resolve input groups
declare -a INPUT_GROUPS=()
if [[ -d "$IN" ]]; then
  shopt -s nullglob
  for d in "$IN"/*; do
    [[ -d "$d" ]] || continue
    if find "$d" -type f -name '*.txt' | head -n 1 | grep -q .; then INPUT_GROUPS+=("$d"); fi
  done
  shopt -u nullglob
  if [[ ${#INPUT_GROUPS[@]} -eq 0 ]]; then INPUT_GROUPS=("$IN"); fi
else
  echo "Input must be a directory: $IN" >&2; exit 1
fi

# Pipelines (same as compare script)
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
if [[ -n "$ONLY" ]]; then
  keys=($ONLY)
else
  keys=(S_core S_core_rel D_core_rel D_core_coref)
  if [[ "$HAS_TEMP_MODELS" -eq 1 ]]; then
    keys+=(S_core_temp S_core_temp_coref D_core_temp D_core_temp_coref S_core_temp_coref_smoke D_core_temp_coref_smoke)
  fi
fi

sanitize_dict() {
  local in="$1"; local out="$2"; cp -f "$in" "$out"
  sed -i -E \
    -e 's#<implementationName>org.apache.ctakes.dictionary.lookup2.dictionary.UmlsJdbcRareWordDictionary</implementationName>#<implementationName>org.apache.ctakes.dictionary.lookup2.dictionary.JdbcRareWordDictionary</implementationName>#' \
    -e 's#<implementationName>org.apache.ctakes.dictionary.lookup2.concept.UmlsJdbcConceptFactory</implementationName>#<implementationName>org.apache.ctakes.dictionary.lookup2.concept.JdbcConceptFactory</implementationName>#' \
    -e 's#(key=\"jdbcDriver\" value)=\"[^\"]*\"#\1=\"org.hsqldb.jdbc.JDBCDriver\"#' \
    -e '/<property key=\"umlsUrl\"/d' -e '/<property key=\"umlsVendor\"/d' -e '/<property key=\"umlsUser\"/d' -e '/<property key=\"umlsPass\"/d' \
    "$out"
}

make_shards() {
  local src="$1"; local shards_dir="$2"; local n="$3"
  mkdir -p "$shards_dir"
  local i=0
  while IFS= read -r -d '' f; do
    local g; g=$(( i % n ))
    local gd; gd=$(printf "%03d" "$g")
    mkdir -p "$shards_dir/$gd"
    ln "$f" "$shards_dir/$gd/"
    i=$((i+1))
  done < <(find "$src" -type f -name '*.txt' -print0)
}

run_pipeline_sharded() {
  local name="$1"; local piper="$2"; local group_dir="$3"; local out_base="$4"
  local gshort; gshort=$(short_name "$group_dir")
  local stamp; stamp=$(date +%Y%m%d-%H%M%S)
  local parent
  if [[ -n "$PARENT_DIR" ]]; then
    parent="$PARENT_DIR"
  else
    parent="$out_base/${name}_${gshort}_$stamp"
  fi
  mkdir -p "$parent"
  local shards_dir="$parent/shards"
  if [[ -d "$shards_dir" ]]; then
    echo "[resume] Using existing shards at $shards_dir" >&2
  else
    make_shards "$group_dir" "$shards_dir" "$RUNNERS"
  fi

  # Ensure Temporal model path variant exists
  local event_src="$CTAKES_HOME/resources/org/apache/ctakes/temporal/models/eventannotator/model.jar"
  local event_dst="$CTAKES_HOME/resources/org/apache/ctakes/temporal/ae/eventannotator/model.jar"
  if [[ -f "$event_src" && ! -f "$event_dst" ]]; then mkdir -p "$(dirname "$event_dst")"; cp -f "$event_src" "$event_dst"; fi

  local -a pids=()
  for i in $(seq -f "%03g" 0 $((RUNNERS-1))); do
    shard="$shards_dir/$i"; [[ -d "$shard" ]] || continue
    outdir="$parent/shard_$i"; mkdir -p "$outdir"
    # Build a pending set if resuming: include only notes that don't have XMI yet
    local in_dir="$shard"
    if [[ "$RESUME" -eq 1 ]]; then
      local pending="$parent/pending_$i"
      rm -rf "$pending" && mkdir -p "$pending"
      # Index processed docs by basename
      declare -A done=()
      if [[ -d "$outdir/xmi" ]]; then
        while IFS= read -r -d '' xmi; do
          bn=$(basename "$xmi")
          bn="${bn%.txt.xmi}"
          done["$bn"]=1
        done < <(find "$outdir/xmi" -type f -name '*.txt.xmi' -print0)
      fi
      # Link only missing docs
      while IFS= read -r -d '' txt; do
        bn=$(basename "$txt")
        bn="${bn%.txt}"
        if [[ -z "${done[$bn]:-}" ]]; then
          ln "$txt" "$pending/"
        fi
      done < <(find "$shard" -type f -name '*.txt' -print0)
      # If nothing pending, skip this shard
      if ! find "$pending" -type f -name '*.txt' | head -n1 | grep -q .; then
        echo "[resume] shard $i already complete; skipping" >&2
        continue
      fi
      in_dir="$pending"
    fi
    xml="$outdir/${DICT_NAME}_local.xml"; sanitize_dict "$DICT_XML" "$xml"
    # Create a per-run piper file with the requested thread count
    tuned_piper="$outdir/$(basename "$piper")"
    if grep -Eq "^\s*threads\s+[0-9]+" "$piper" 2>/dev/null; then
      sed -E "s#^\s*threads\s+[0-9]+#threads ${THREADS}#" "$piper" > "$tuned_piper"
    else
      { echo "threads ${THREADS}"; cat "$piper"; } > "$tuned_piper"
    fi
    if [[ -f "$SRC_DB_DIR/$DICT_NAME.script" ]]; then
      workdb="/dev/shm/${DICT_NAME}_${name}_$i"; mkdir -p "$(dirname "$workdb")"
      cp -f "$SRC_DB_DIR/$DICT_NAME.properties" "$workdb.properties"
      cp -f "$SRC_DB_DIR/$DICT_NAME.script" "$workdb.script"
      sed -i -E "s#(key=\"jdbcUrl\" value)=\"[^\"]+\"#\1=\"jdbc:hsqldb:file:${workdb}\"#" "$xml"
    fi
    (
      cd "$CTAKES_HOME" >/dev/null
      stdbuf -oL -eL java -Xms${XMX_MB}m -Xmx${XMX_MB}m \
        -Dorg.slf4j.simpleLogger.defaultLogLevel=info \
        -cp "$JAVA_CP" \
        org.apache.ctakes.core.pipeline.PiperFileRunner \
        -p "$tuned_piper" -i "$in_dir" -o "$outdir" -l "$xml" \
        | sed -u "s/^/[${name}_$i] /" | tee "$outdir/run.log"
    ) &
    pids+=($!)
  done
  # Wait for shards but do not abort the whole script if one fails
  local any_fail=0
  for pid in "${pids[@]}"; do
    if ! wait "$pid"; then any_fail=1; echo "WARN: shard PID $pid failed in $name/$gshort" >&2; fi
  done

  # Save a parent-level combined run.log and pipeline file for reporting/metrics
  # - Combined log: concatenation of shard_*/run.log in numeric order
  # - Piper: copy tuned piper used by shards (same across shards) to parent
  {
    local combined="$parent/run.log"
    : > "$combined" || true
    for sh in $(ls -1d "$parent"/shard_* 2>/dev/null | sort); do
      if [[ -f "$sh/run.log" ]]; then cat "$sh/run.log" >> "$combined"; fi
    done
    # Copy a piper file from first shard that has it
    local base_piper_name; base_piper_name="$(basename "$piper")"
    for sh in $(ls -1d "$parent"/shard_* 2>/dev/null | sort); do
      if [[ -f "$sh/$base_piper_name" ]]; then cp -f "$sh/$base_piper_name" "$parent/$base_piper_name"; break; fi
    done
  } || true

  # Optional summary workbook across shards (short name)
  # Post-processing: consolidate shards into top-level folders before any report
  if [[ "$CONSOLIDATE" -eq 1 ]]; then
    if [[ "$any_fail" -eq 0 ]]; then
      echo "[post] Consolidating shards into top-level outputs for $name/$gshort"
      if [[ "$KEEP_SHARDS" -eq 1 ]]; then
        bash "$BASE_DIR/scripts/consolidate_shards.sh" -p "$parent" --keep-shards || true
      else
        bash "$BASE_DIR/scripts/consolidate_shards.sh" -p "$parent" || true
      fi
    else
      echo "[post] Skipping consolidation due to shard failures; resume then consolidate"
    fi
  else
    echo "[post] Skipping consolidation (--no-consolidate)"
  fi

  # Optional summary workbook (after consolidation)
  # Optional per-document CSVs that match Clinical Concepts columns
  if [[ "$ENHANCED_CSV" -eq 1 && "$any_fail" -eq 0 ]]; then
    echo "[post] Building per-document Clinical Concepts CSVs into $ENH_CSV_DIR/"
    : # Enhanced per-doc CSV now produced in-pipeline via ClinicalConceptCsvWriter
  fi

  # Optional summary workbook (after consolidation)
  if [[ "$MAKE_REPORTS" -eq 1 ]]; then
    local rpt="$parent/ctakes-${name}-${gshort}.xml"
    echo "[report] Building per-pipeline report (sync, summary): $rpt"
    bash "$BASE_DIR/scripts/build_xlsx_report.sh" -o "$parent" -w "$rpt" -M summary || true
    echo "- Report: $rpt"
  elif [[ "$MAKE_REPORTS" -eq 2 ]]; then
    local rpt="$parent/ctakes-${name}-${gshort}.xml"
    echo "[report] Building per-pipeline report (async, summary): $rpt"
    ( bash "$BASE_DIR/scripts/build_xlsx_report.sh" -o "$parent" -w "$rpt" -M summary || true ) &
    REPORT_PIDS+=($!)
  fi
  if [[ "$any_fail" -eq 1 ]]; then
    echo "Completed $name on group $gshort with warnings -> $parent" >&2
  else
    echo "Completed $name on group $gshort -> $parent"
  fi
}

for key in "${keys[@]}"; do
  piper="${SETS[$key]}"; [[ -f "$piper" ]] || { echo "Missing pipeline: $piper" >&2; continue; }
for grp in "${INPUT_GROUPS[@]}"; do
    run_pipeline_sharded "$key" "$piper" "$grp" "$OUT"
  done
done

if [[ "$SKIP_PARENT" -eq 0 ]]; then
  STAMP="$(date +%Y%m%d-%H%M%S)"
  PARENT_REPORT="$OUT/ctakes-report-compare-${STAMP}.xlsx"
  bash "$BASE_DIR/scripts/build_xlsx_report.sh" -o "$OUT" -w "$PARENT_REPORT" -M summary || true
  echo "- Summary: $PARENT_REPORT"
else
  echo "[report] Skipping parent compare summary (--no-parent-report)"
fi

if [[ "$MAKE_REPORTS" -eq 2 && ${#REPORT_PIDS[@]} -gt 0 ]]; then
  echo "[report] Waiting for async per-pipeline reports to finish (${#REPORT_PIDS[@]} jobs) ..."
  for rpid in "${REPORT_PIDS[@]}"; do
    wait "$rpid" || true
  done
  echo "[report] All async reports completed."
fi
