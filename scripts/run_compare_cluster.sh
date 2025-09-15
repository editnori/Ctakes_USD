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
#     [-n RUNNERS] [-m XMX_MB] [-t THREADS] [--reports|--reports-sync|--reports-async] \
#     [--no-consolidate|--keep-shards|--consolidate-async]

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
CTAKES_HOME="${CTAKES_HOME:-$BASE_DIR/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0}"
# Prepend repo overrides/resources before cTAKES resources so we can override default configs (e.g., DefaultListRegex.bsv)
JAVA_CP="$BASE_DIR/resources_override:$BASE_DIR/resources:$CTAKES_HOME/desc:$CTAKES_HOME/resources:$CTAKES_HOME/config:$CTAKES_HOME/config/*:$CTAKES_HOME/lib/*:$BASE_DIR/.build_tools"

IN=""; OUT=""; RUNNERS="${RUNNERS:-16}"; XMX_MB="${XMX_MB:-6144}"; THREADS="${THREADS:-6}"; MAKE_REPORTS=0
# Optional: limit concurrent pipeline-group executions (kept 1 by default for stability)
MAX_PIPELINES="${MAX_PIPELINES:-1}"
# Optional: autoscale runners/threads/xmx based on host cores/memory
AUTOSCALE=0
# Optional: build per-note-type workbooks after consolidation
NOTE_SPLITS=0
NOTE_TYPES_ARG="${NOTE_TYPES:-}"
# Global report extension default (used outside functions as well)
REPORT_EXT="${REPORT_EXT:-xlsx}"
# Control dictionary handling (default: no sanitization, use provided XML as-is)
CTAKES_SANITIZE_DICT="${CTAKES_SANITIZE_DICT:-0}"
DICT_XML_ARG="${DICT_XML:-}"  # allow DICT_XML env or --dict-xml flag
# Use a single shared read-only HSQLDB dictionary for all shards (reduces duplicate init)
DICT_SHARED="${DICT_SHARED:-1}"
# Directory to host the shared dictionary files (properties/script)
# Defaults to /dev/shm (tmpfs); set DICT_SHARED_PATH to persist across runs/hosts (e.g., /var/tmp/ctakes_dict_cache)
DICT_SHARED_PATH="${DICT_SHARED_PATH:-/dev/shm}"
PARENT_DIR=""; RESUME=0; ONLY=""; SKIP_PARENT=0; CONSOLIDATE=1; KEEP_SHARDS=0; SHARD_SEED="${SEED:-}"
CONSOLIDATE_ASYNC=1
declare -a CONSOLIDATE_PIDS=()
declare -a REPORT_PIDS=()
declare -a CHILD_PIDS=()
# Graceful shutdown: on INT/TERM, signal children and wait
_graceful_exit() {
  echo "[runner] Caught termination signal; attempting graceful shutdown..." >&2
  local p
  for p in "${CHILD_PIDS[@]}"; do
    if kill -0 "$p" 2>/dev/null; then kill "$p" 2>/dev/null || true; fi
  done
  # Give background jobs up to 20s to finish
  local end=$(( $(date +%s) + 20 ))
  while (( $(date +%s) < end )); do
    jobs -rp >/dev/null 2>&1 || break
    sleep 1
  done
  exit 143
}
trap _graceful_exit INT TERM
# UMLS key handling: default to env UMLS_KEY, fallback to project default if provided
UMLS_KEY="${UMLS_KEY:-6370dcdd-d438-47ab-8749-5a8fb9d013f2}"
while [[ $# -gt 0 ]]; do
  case "$1" in
    -i|--in) IN="$2"; shift 2;;
    -o|--out) OUT="$2"; shift 2;;
    -n|--runners) RUNNERS="$2"; shift 2;;
    -m|--xmx) XMX_MB="$2"; shift 2;;
    -t|--threads) THREADS="$2"; shift 2;;
    --reports) MAKE_REPORTS=2; shift 1;;
    --reports-sync) MAKE_REPORTS=1; shift 1;;
    --reports-async) MAKE_REPORTS=2; shift 1;;
    --parent) PARENT_DIR="$2"; shift 2;;
    --resume) RESUME=1; shift 1;;
    --only) ONLY="$2"; shift 2;;
    --max-pipelines) MAX_PIPELINES="$2"; shift 2;;
    --autoscale) AUTOSCALE=1; shift 1;;
    --note-type-splits) NOTE_SPLITS=1; shift 1;;
    --note-types) NOTE_TYPES_ARG="$2"; shift 2;;
    --no-parent-report) SKIP_PARENT=1; shift 1;;
    --no-consolidate) CONSOLIDATE=0; shift 1;;
    --keep-shards) KEEP_SHARDS=1; shift 1;;
    --seed) SHARD_SEED="$2"; shift 2;;
    --consolidate-async) CONSOLIDATE_ASYNC=1; shift 1;;
    -l|--dict-xml) DICT_XML_ARG="$2"; shift 2;;
    *) echo "Unknown arg: $1" >&2; exit 2;;
  esac
done

[[ -z "$IN" ]] && { echo "-i|--in is required" >&2; exit 2; }
OUT="${OUT:-$BASE_DIR/outputs/compare}"
mkdir -p "$OUT"

# Optional autoscale: derive sensible defaults
if [[ "$AUTOSCALE" -eq 1 ]]; then
  # Detect cores
  if command -v nproc >/dev/null 2>&1; then
    CORES=$(nproc)
  elif [[ -f /proc/cpuinfo ]]; then
    CORES=$(grep -c '^processor' /proc/cpuinfo 2>/dev/null || echo 1)
  else
    CORES=${NUMBER_OF_PROCESSORS:-1}
  fi
  [[ -z "$CORES" || "$CORES" -lt 1 ]] && CORES=1
  # Detect memory (MB)
  if [[ -f /proc/meminfo ]]; then
    MEM_MB=$(awk '/MemTotal:/ {printf "%d", $2/1024}' /proc/meminfo 2>/dev/null || echo 0)
  else
    MEM_MB=${MEM_MB:-0}
  fi
  # Allow override via env TARGET_MEM_FRAC (default 0.65)
  TARGET_MEM_FRAC=${TARGET_MEM_FRAC:-0.65}
  # Choose XMX per runner based on total memory
  if [[ "$MEM_MB" -ge 1048576 ]]; then       # >= 1 TB
    XMX_MB=${XMX_MB:-12288}
  elif [[ "$MEM_MB" -ge 524288 ]]; then       # >= 512 GB
    XMX_MB=${XMX_MB:-9216}
  else
    XMX_MB=${XMX_MB:-6144}
  fi
  # Threads per runner: default 4 on large-core boxes, else 6
  if [[ "$CORES" -ge 64 ]]; then
    THREADS=${THREADS:-4}
  else
    THREADS=${THREADS:-6}
  fi
  # Runners limited by cores and memory
  max_by_cpu=$(( CORES / (THREADS>0?THREADS:1) ))
  [[ "$max_by_cpu" -lt 1 ]] && max_by_cpu=1
  usable_mem_mb=$(awk -v m="$MEM_MB" -v f="$TARGET_MEM_FRAC" 'BEGIN{ printf "%d", (m*f) }')
  max_by_mem=$(( usable_mem_mb / (XMX_MB>0?XMX_MB:1) ))
  [[ "$max_by_mem" -lt 1 ]] && max_by_mem=1
  # Cap aggressively to avoid GC thrash, leave 20% headroom
  RUNNERS=$(( max_by_cpu < max_by_mem ? max_by_cpu : max_by_mem ))
  # Constrain to a reasonable ceiling unless explicitly overridden
  if [[ "$RUNNERS" -gt 192 ]]; then RUNNERS=192; fi
  echo "[autoscale] CORES=${CORES} MEM_MB=${MEM_MB} -> RUNNERS=${RUNNERS} THREADS=${THREADS} XMX_MB=${XMX_MB}" >&2
fi

# Pre-run flight checks (fail fast on missing deps / dict / models)
if [[ "$DICT_SHARED" -eq 1 ]]; then
  CTAKES_SANITIZE_DICT="$CTAKES_SANITIZE_DICT" bash "$BASE_DIR/scripts/flight_check.sh" --mode cluster --require-shared || exit 1
else
  CTAKES_SANITIZE_DICT="$CTAKES_SANITIZE_DICT" bash "$BASE_DIR/scripts/flight_check.sh" --mode cluster || exit 1
fi

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
if [[ -n "$DICT_XML_ARG" ]]; then
  DICT_XML="$DICT_XML_ARG"
else
  DICT_XML="$CTAKES_HOME/resources/org/apache/ctakes/dictionary/lookup/fast/${DICT_NAME}.xml"
fi
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
  [S_core_smoke]="$BASE_DIR/pipelines/compare/TsSectionedSmoking_WSD_Compare.piper"
  [S_core_rel_smoke]="$BASE_DIR/pipelines/compare/TsSectionedCoreRelSmoke_WSD_Compare.piper"
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
  # Default remains multi-pipeline for back-compat, but single-pass main wrapper will select S_core_rel_smoke.
  keys=(S_core S_core_rel D_core_rel D_core_coref)
  if [[ "$HAS_TEMP_MODELS" -eq 1 ]]; then
    keys+=(S_core_temp S_core_temp_coref D_core_temp D_core_temp_coref S_core_temp_coref_smoke D_core_temp_coref_smoke)
  fi
fi

sanitize_dict() {
  local in="$1"; local out="$2"; cp -f "$in" "$out"
  # In non-sanitize mode, this is just a copy; keeping hook for future use
}

hash_to_shard() { # file n seed -> shard index [0..n-1]
  local f="$1"; local n="$2"; local seed="${3:-}"
  local h
  if command -v sha1sum >/dev/null 2>&1; then
    h=$(printf "%s" "$f$seed" | sha1sum | awk '{print $1}')
    # take last 8 hex digits for speed
    h=$(( 0x${h:0:8} ))
  else
    h=$(printf "%s" "$f$seed" | cksum | awk '{print $1}')
  fi
  echo $(( h % n ))
}

make_shards() {
  local src="$1"; local shards_dir="$2"; local n="$3"; local seed="$4"
  mkdir -p "$shards_dir"
  # Stable file list
  while IFS= read -r -d '' f; do
    local g; g=$(hash_to_shard "$f" "$n" "$seed")
    local gd; gd=$(printf "%03d" "$g")
    mkdir -p "$shards_dir/$gd"
    ln "$f" "$shards_dir/$gd/"
  done < <(find "$src" -type f -name '*.txt' -print0 | sort -z)
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
    make_shards "$group_dir" "$shards_dir" "$RUNNERS" "$SHARD_SEED"
  fi

  # Ensure Temporal model path variant exists
  local event_src="$CTAKES_HOME/resources/org/apache/ctakes/temporal/models/eventannotator/model.jar"
  local event_dst="$CTAKES_HOME/resources/org/apache/ctakes/temporal/ae/eventannotator/model.jar"
  if [[ -f "$event_src" && ! -f "$event_dst" ]]; then mkdir -p "$(dirname "$event_dst")"; cp -f "$event_src" "$event_dst"; fi

  local -a pids=()
  # Prepare shared dictionary DB copy in /dev/shm if requested
  local shareddb="${DICT_SHARED_PATH%/}/${DICT_NAME}_shared"
  local using_shared_db=0
  if [[ "$DICT_SHARED" -eq 1 && -f "$SRC_DB_DIR/$DICT_NAME.script" ]]; then
    using_shared_db=1
    if [[ ! -f "${shareddb}.script" ]]; then
      echo "[dict] Creating shared read-only HSQLDB copy: ${shareddb}" >&2
      cp -f "$SRC_DB_DIR/$DICT_NAME.properties" "${shareddb}.properties"
      cp -f "$SRC_DB_DIR/$DICT_NAME.script" "${shareddb}.script"
      # Sanity: verify copy exists and is reasonably large
      if [[ ! -s "${shareddb}.script" ]]; then
        echo "[dict][fatal] Shared dictionary copy missing or empty: ${shareddb}.script" >&2
        echo "              Check DICT_SHARED_PATH=${DICT_SHARED_PATH} exists and has free space." >&2
        exit 1
      fi
    else
      echo "[dict] Using existing shared read-only HSQLDB: ${shareddb}" >&2
    fi
    # HSQLDB file databases cannot be opened by multiple JVMs concurrently.
    # If we will run more than one shard or more than one pipeline concurrently,
    # fall back to per-shard RAM copies to avoid lock acquisition failures.
    if [[ "${DICT_SHARED_FORCE:-0}" -ne 1 ]]; then
      if (( RUNNERS > 1 )) || (( ${MAX_PIPELINES:-1} > 1 )); then
        echo "[dict] Disabling shared DB for concurrency (RUNNERS=${RUNNERS}, MAX_PIPELINES=${MAX_PIPELINES:-1})." >&2
        echo "       Using per-shard copies under /dev/shm to avoid HSQLDB lock conflicts." >&2
        using_shared_db=0
      fi
    fi
    # Clear a stale lock file if present when using the shared DB and only 1 JVM will run
    if [[ "$using_shared_db" -eq 1 ]]; then
      rm -f "${shareddb}.lck" 2>/dev/null || true
    fi
  fi
  for i in $(seq -f "%03g" 0 $((RUNNERS-1))); do
    shard="$shards_dir/$i"; [[ -d "$shard" ]] || continue
    outdir="$parent/shard_$i"; mkdir -p "$outdir"
    # Build a pending set if resuming: include only notes that don't have XMI yet
    local in_dir="$shard"
    if [[ "$RESUME" -eq 1 ]]; then
      local pending; pending=$(mktemp -d -p "$parent" "pending_${i}_XXXXXX")
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
        rmdir "$pending" 2>/dev/null || true
        continue
      fi
      in_dir="$pending"
    fi
    # Resolve dictionary XML for this shard
    if [[ "$CTAKES_SANITIZE_DICT" -eq 1 ]]; then
      xml="$outdir/${DICT_NAME}_local.xml"; sanitize_dict "$DICT_XML" "$xml"
    else
      xml="$DICT_XML"
    fi
    # Create a per-run piper file with the requested thread count
    tuned_piper="$outdir/$(basename "$piper")"
    if grep -Eq "^[[:space:]]*threads[[:space:]]+[0-9]+" "$piper" 2>/dev/null; then
      sed -E "s#^[[:space:]]*threads[[:space:]]+[0-9]+#threads ${THREADS}#" "$piper" > "$tuned_piper"
    else
      { echo "threads ${THREADS}"; cat "$piper"; } > "$tuned_piper"
    fi
    # Rewrite relative includes to absolute repo paths so includes resolve from any location
    if command -v sed >/dev/null 2>&1; then
      # Rewrite lines like: "load ../../pipelines/..." or "include ../../pipelines/..."
      # Anchor at start with optional leading spaces to avoid accidental matches in comments.
      sed -i -E "s#^[[:space:]]*(load|include)[[:space:]]+\.\.\/\.\.\/pipelines/#\\1 $BASE_DIR/pipelines/#g" "$tuned_piper" || true
    fi

    # Ensure TimingEndAE writes a per-shard timing CSV to accelerate reporting
    timing_file="$outdir/timing_csv/timing.csv"; mkdir -p "$(dirname "$timing_file")"
    if ! grep -Eq "TimingEndAE.*TimingFile=" "$tuned_piper" 2>/dev/null; then
      # Append TimingFile to any TimingEndAE add line
      sed -i -E "/^[[:space:]]*add[[:space:]]+tools\\.timing\\.TimingEndAE([[:space:]]|$)/ s|$| TimingFile=\"$timing_file\"|" "$tuned_piper" || true
    fi
    if [[ "$CTAKES_SANITIZE_DICT" -eq 1 && -f "$SRC_DB_DIR/$DICT_NAME.script" ]]; then
      if [[ "$using_shared_db" -eq 1 ]]; then
        workdb="$shareddb"
      else
        workdb="/dev/shm/${DICT_NAME}_${name}_$i"; mkdir -p "$(dirname "$workdb")"
        cp -f "$SRC_DB_DIR/$DICT_NAME.properties" "$workdb.properties"
        cp -f "$SRC_DB_DIR/$DICT_NAME.script" "$workdb.script"
        if [[ ! -s "$workdb.script" ]]; then
          echo "[${name}_$i][fatal] Per-shard dictionary copy missing or empty: $workdb.script" | tee -a "$outdir/run.log" >&2
          exit 1
        fi
      fi
      # Point JDBC to the shared/per-shard DB. Do NOT append HSQL flags here.
      # cTAKES 6.0.0 validates the URL by resolving <path>.script; flags in the URL break that check.
      sed -i -E "s#(key=\"jdbcUrl\" value)=\"[^\"]+\"#\1=\"jdbc:hsqldb:file:${workdb}\"#" "$xml"
      # Debug: record resolved DB path and JDBC URL for this shard
      echo "[${name}_$i][dict] workdb=${workdb}" | tee -a "$outdir/run.log" >&2
      if command -v rg >/dev/null 2>&1; then
        rg -n "jdbcUrl" -S "$xml" | sed -u "s/^/[${name}_$i][dict] /" | tee -a "$outdir/run.log" >&2 || true
      else
        grep -n "jdbcUrl" "$xml" | sed -u "s/^/[${name}_$i][dict] /" | tee -a "$outdir/run.log" >&2 || true
      fi
    fi
    (
      set +e
      cd "$BASE_DIR" >/dev/null
      # Ensure per-shard temp directory exists for -Djava.io.tmpdir
      mkdir -p "$outdir/tmp" || true
      attempt=1
      last_ec=0
  while (( attempt <= 3 )); do
        stdbuf -oL -eL java -Xms${XMX_MB}m -Xmx${XMX_MB}m \
          -XX:+UseG1GC -XX:+ParallelRefProcEnabled -XX:+UseStringDeduplication -XX:MaxGCPauseMillis=200 \
          -XX:+AlwaysPreTouch -XX:+ExitOnOutOfMemoryError ${JVM_OPTS:-} \
          -Djava.io.tmpdir="$outdir/tmp" \
          ${UMLS_KEY:+-Dctakes.umls_apikey=$UMLS_KEY} \
          -Dorg.slf4j.simpleLogger.defaultLogLevel=info \
          -Dorg.slf4j.simpleLogger.log.org.apache.ctakes.dictionary=warn \
          -Dorg.slf4j.simpleLogger.log.org.apache.ctakes.dictionary.lookup2=warn \
          -Dorg.slf4j.simpleLogger.log.org.apache.uima=warn \
          -Dorg.slf4j.simpleLogger.log.de.tudarmstadt.ukp=warn \
          -Dorg.slf4j.simpleLogger.log.org.cleartk=warn \
          -Dorg.slf4j.simpleLogger.log.opennlp=warn \
          -Dorg.slf4j.simpleLogger.log.org.apache.ctakes.core.ae.RegexSpanFinder=warn \
          -Dorg.slf4j.simpleLogger.log.org.apache.uima.cas.impl.XmiCasSerializer=${XMI_LOG_LEVEL:-warn} \
          -cp "$JAVA_CP" \
          org.apache.ctakes.core.pipeline.PiperFileRunner \
          -p "$tuned_piper" -i "$in_dir" -o "$outdir" -l "$xml" ${UMLS_KEY:+--key $UMLS_KEY} \
          | sed -u "s/^/[${name}_$i] /" | tee -a "$outdir/run.log"
        ec=${PIPESTATUS[0]}
        last_ec=$ec
        if (( ec == 0 )); then
          break
        fi
        echo "[${name}_$i] attempt $attempt failed (exit=$ec). Retrying..." | tee -a "$outdir/run.log"
        sleep_sec=$(( 2 ** (attempt - 1) ))
        sleep "$sleep_sec"
        attempt=$(( attempt + 1 ))
      done
      set -e
      # Clean temporary pending dir if created
      if [[ "${pending:-}" =~ ^$parent/pending_ ]]; then rm -rf "$pending" 2>/dev/null || true; fi
      exit $last_ec
    ) &
    pids+=($!)
    CHILD_PIDS+=($!)
  done
  # Wait for shards but do not abort the whole script if one fails
  local any_fail=0
  for pid in "${pids[@]}"; do
    if ! wait "$pid"; then
      any_fail=1
      echo "WARN: shard PID $pid failed in $name/$gshort" >&2
    fi
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

  # Post-processing: consolidate shards, optionally async; then optionally build per-pipeline report
  # use global REPORT_EXT
  local rpt="$parent/ctakes-${name}-${gshort}.${REPORT_EXT}"
  if [[ "$CONSOLIDATE" -eq 1 ]]; then
    if [[ "$any_fail" -eq 0 ]]; then
      if [[ "$CONSOLIDATE_ASYNC" -eq 1 ]]; then
        echo "[post] Queueing consolidation for $name/$gshort (async)"
        if [[ "$KEEP_SHARDS" -eq 1 ]]; then
          if [[ "$MAKE_REPORTS" -gt 0 ]]; then
            ( bash "$BASE_DIR/scripts/consolidate_shards.sh" -p "$parent" --keep-shards && \
              bash "$BASE_DIR/scripts/build_xlsx_report.sh" -o "$parent" -w "$rpt" -M csv || true; \
              if [[ "$NOTE_SPLITS" -eq 1 ]]; then \
                if [[ -n "$NOTE_TYPES_ARG" ]]; then bash "$BASE_DIR/scripts/build_split_reports.sh" -p "$parent" -M csv --types "$NOTE_TYPES_ARG" || true; \
                else bash "$BASE_DIR/scripts/build_split_reports.sh" -p "$parent" -M csv || true; fi; \
              fi ) &
          else
            ( bash "$BASE_DIR/scripts/consolidate_shards.sh" -p "$parent" --keep-shards; \
              if [[ "$NOTE_SPLITS" -eq 1 ]]; then \
                if [[ -n "$NOTE_TYPES_ARG" ]]; then bash "$BASE_DIR/scripts/build_split_reports.sh" -p "$parent" -M csv --types "$NOTE_TYPES_ARG" || true; \
                else bash "$BASE_DIR/scripts/build_split_reports.sh" -p "$parent" -M csv || true; fi; \
              fi ) &
          fi
        else
          if [[ "$MAKE_REPORTS" -gt 0 ]]; then
            ( bash "$BASE_DIR/scripts/consolidate_shards.sh" -p "$parent" && \
              bash "$BASE_DIR/scripts/build_xlsx_report.sh" -o "$parent" -w "$rpt" -M csv || true; \
              if [[ "$NOTE_SPLITS" -eq 1 ]]; then \
                if [[ -n "$NOTE_TYPES_ARG" ]]; then bash "$BASE_DIR/scripts/build_split_reports.sh" -p "$parent" -M csv --types "$NOTE_TYPES_ARG" || true; \
                else bash "$BASE_DIR/scripts/build_split_reports.sh" -p "$parent" -M csv || true; fi; \
              fi ) &
          else
            ( bash "$BASE_DIR/scripts/consolidate_shards.sh" -p "$parent"; \
              if [[ "$NOTE_SPLITS" -eq 1 ]]; then \
                if [[ -n "$NOTE_TYPES_ARG" ]]; then bash "$BASE_DIR/scripts/build_split_reports.sh" -p "$parent" -M csv --types "$NOTE_TYPES_ARG" || true; \
                else bash "$BASE_DIR/scripts/build_split_reports.sh" -p "$parent" -M csv || true; fi; \
              fi ) &
          fi
        fi
        CONSOLIDATE_PIDS+=($!)
      else
        echo "[post] Consolidating shards into top-level outputs for $name/$gshort"
        if [[ "$KEEP_SHARDS" -eq 1 ]]; then
          bash "$BASE_DIR/scripts/consolidate_shards.sh" -p "$parent" --keep-shards || true
        else
          bash "$BASE_DIR/scripts/consolidate_shards.sh" -p "$parent" || true
        fi
        # Build report synchronously if requested
        if [[ "$MAKE_REPORTS" -eq 1 ]]; then
          echo "[report] Building per-pipeline report (sync, csv): $rpt"
          bash "$BASE_DIR/scripts/build_xlsx_report.sh" -o "$parent" -w "$rpt" -M csv || true
          echo "- Report: $rpt"
        elif [[ "$MAKE_REPORTS" -eq 2 ]]; then
          echo "[report] Building per-pipeline report (async, csv): $rpt"
          ( bash "$BASE_DIR/scripts/build_xlsx_report.sh" -o "$parent" -w "$rpt" -M csv || true ) &
          REPORT_PIDS+=($!)
        fi
        # Build per-note-type workbooks if requested
        if [[ "$NOTE_SPLITS" -eq 1 ]]; then
          if [[ -n "$NOTE_TYPES_ARG" ]]; then
            bash "$BASE_DIR/scripts/build_split_reports.sh" -p "$parent" -M csv --types "$NOTE_TYPES_ARG" || true
          else
            bash "$BASE_DIR/scripts/build_split_reports.sh" -p "$parent" -M csv || true
          fi
        fi
      fi
    else
      echo "[post] Skipping consolidation due to shard failures; resume then consolidate"
    fi
  else
    echo "[post] Skipping consolidation (--no-consolidate)"
  fi
  if [[ "$any_fail" -eq 1 ]]; then
    echo "Completed $name on group $gshort with warnings -> $parent" >&2
    # Collect failed shard logs for quick triage
    mkdir -p "$parent/errors"
    for sh in $(ls -1d "$parent"/shard_* 2>/dev/null | sort); do
      if [[ -f "$sh/run.log" ]]; then
        # Consider failure if no xmi produced in shard or last line contains attempt failed
        if ! find "$sh/xmi" -maxdepth 1 -type f -name '*.xmi' -print -quit | grep -q . || \
           tail -n 5 "$sh/run.log" | grep -qi 'attempt .* failed'; then
          cp -f "$sh/run.log" "$parent/errors/$(basename "$sh").log" || true
        fi
      fi
    done
  else
    echo "Completed $name on group $gshort -> $parent"
  fi
}

# Execute tasks with top-level concurrency control
declare -a TOP_PIDS=()
for key in "${keys[@]}"; do
  piper="${SETS[$key]}"; [[ -f "$piper" ]] || { echo "Missing pipeline: $piper" >&2; continue; }
  for grp in "${INPUT_GROUPS[@]}"; do
    # throttle top-level task fan-out
    while (( $(jobs -rp | wc -l) >= ${MAX_PIPELINES:-1} )); do sleep 1; done
    run_pipeline_sharded "$key" "$piper" "$grp" "$OUT" &
    TOP_PIDS+=($!)
  done
done
# Wait for all top-level tasks
top_any_fail=0
for pid in "${TOP_PIDS[@]}"; do
  if ! wait "$pid"; then top_any_fail=1; fi
done
if (( top_any_fail )); then echo "[warn] One or more pipeline-group tasks reported failures" >&2; fi

if [[ "$MAKE_REPORTS" -eq 2 && ${#REPORT_PIDS[@]} -gt 0 ]]; then
  echo "[report] Waiting for async per-pipeline reports to finish (${#REPORT_PIDS[@]} jobs) ..."
  for rpid in "${REPORT_PIDS[@]}"; do
    wait "$rpid" || true
  done
  echo "[report] All async reports completed."
fi

if [[ "$CONSOLIDATE_ASYNC" -eq 1 && ${#CONSOLIDATE_PIDS[@]} -gt 0 ]]; then
  echo "[post] Waiting for async consolidation/report jobs to finish (${#CONSOLIDATE_PIDS[@]} jobs) ..."
  for cpid in "${CONSOLIDATE_PIDS[@]}"; do
    wait "$cpid" || true
  done
  echo "[post] All async consolidation/report jobs completed."
fi

if [[ "$SKIP_PARENT" -eq 0 ]]; then
  STAMP="$(date +%Y%m%d-%H%M%S)"
  PARENT_REPORT="$OUT/ctakes-report-compare-${STAMP}.${REPORT_EXT}"
  # Default to csv mode to avoid XMI parse for parent; caller can override via REPORT_ALLOW_XMI=1
  bash "$BASE_DIR/scripts/build_xlsx_report.sh" -o "$OUT" -w "$PARENT_REPORT" -M "${REPORT_MODE:-csv}" || true
  echo "- Summary: $PARENT_REPORT"
else
  echo "[report] Skipping parent compare summary (--no-parent-report)"
fi



