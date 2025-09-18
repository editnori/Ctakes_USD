#!/usr/bin/env bash
set -euo pipefail

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
ENV_FILE="${BASE_DIR}/.ctakes_env"
if [[ -f "${ENV_FILE}" ]]; then
  # shellcheck disable=SC1090
  source "${ENV_FILE}"
fi


RUN_PIPELINE_SCRIPT="${BASE_DIR}/scripts/run_pipeline.sh"
RUN_PIPELINE_CMD=("${BASH:-bash}" "${RUN_PIPELINE_SCRIPT}")

usage() {
  cat <<'USAGE'
Usage: scripts/run_async.sh -i <input_dir> -o <output_dir> [options]
Options:
  --pipeline <core|sectioned|smoke|drug|core_sectioned_smoke>   Pipeline key (default: sectioned)
  --with-relations                        Enable TsRelationSubPipe (core/smoke/drug only)
  --shards <N>                             Number of parallel runners (default: 1 or autoscale recommendation)
  --threads <N>                            Threads per runner (passed to run_pipeline.sh)
  --xmx <MB>                               Heap per runner in MB
  --autoscale                              Estimate shards/threads/heap from host resources (default)
  --no-autoscale                           Disable autoscale heuristics
  --dict <file.xml>                        Dictionary XML to pass through
  --umls-key <KEY>                         UMLS API key override
  --java-opts "..."                       Extra JVM options per runner
  --dry-run                                Print the planned commands then exit
  --help                                   Show this help text

Each shard runs scripts/run_pipeline.sh with its own temp input folder and output folder.
After all shards finish, outputs are consolidated into <output_dir>/<pipeline>/<timestamp>/...
USAGE
}

# Helper functions -----------------------------------------------------------
detect_cpus() {
  if command -v nproc >/dev/null 2>&1; then
    nproc
    return
  fi
  case "$(uname -s 2>/dev/null)" in
    Darwin) sysctl -n hw.ncpu ;;
    MINGW*|MSYS*|CYGWIN*) powershell.exe -NoProfile -Command "(Get-CimInstance -ClassName Win32_ComputerSystem).NumberOfLogicalProcessors" | tr -d '\r' ;;
    *) getconf _NPROCESSORS_ONLN 2>/dev/null || echo 1 ;;
  esac
}

detect_mem_mb() {
  case "$(uname -s 2>/dev/null)" in
    Linux) awk '/MemTotal/ { printf "%d", $2/1024 }' /proc/meminfo ;;
    Darwin) sysctl -n hw.memsize | awk '{ printf "%d", $1/1024/1024 }' ;;
    MINGW*|MSYS*|CYGWIN*) powershell.exe -NoProfile -Command "[math]::Round((Get-CimInstance -ClassName Win32_OperatingSystem).TotalVisibleMemorySize / 1024)" | tr -d '\r' ;;
    *) echo 4096 ;;
  esac
}

PIPELINE_KEY="sectioned"
WITH_RELATIONS=0
SHARDS=""
SHARDS_SET=0
THREADS=""
THREADS_SET=0
XMX=""
XMX_SET=0
AUTOSCALE=1
DICT_XML=""
UMLS_OVERRIDE=""
JAVA_OPTS_EXTRA=""
DRY_RUN=0
IN_DIR=""
OUT_DIR=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    -i|--input) IN_DIR="$2"; shift 2;;
    -o|--output) OUT_DIR="$2"; shift 2;;
    --pipeline) PIPELINE_KEY="$2"; shift 2;;
    --with-relations) WITH_RELATIONS=1; shift 1;;
    --shards) SHARDS="$2"; SHARDS_SET=1; shift 2;;
    --threads) THREADS="$2"; THREADS_SET=1; shift 2;;
    --xmx) XMX="$2"; XMX_SET=1; shift 2;;
    --autoscale) AUTOSCALE=1; shift 1;;
    --no-autoscale) AUTOSCALE=0; shift 1;;
    --dict) DICT_XML="$2"; shift 2;;
    --umls-key) UMLS_OVERRIDE="$2"; shift 2;;
    --java-opts) JAVA_OPTS_EXTRA="$2"; shift 2;;
    --dry-run) DRY_RUN=1; shift 1;;
    --help|-h) usage; exit 0;;
    *) echo "Unknown option: $1" >&2; usage >&2; exit 1;;
  esac

done

if [[ -z "${IN_DIR}" || -z "${OUT_DIR}" ]]; then
  echo "[async] --input and --output are required" >&2
  usage >&2
  exit 1
fi

[[ -d "${IN_DIR}" ]] || { echo "[async] Input directory not found: ${IN_DIR}" >&2; exit 1; }

if [[ ! -f "${RUN_PIPELINE_SCRIPT}" ]]; then
  echo "[async] Missing run_pipeline.sh helper" >&2
  exit 1
fi

mapfile -t FILES < <(find "$IN_DIR" -type f -name '*.txt' -print | sort)
if [[ ${#FILES[@]} -eq 0 ]]; then
  echo "[async] No .txt files found under ${IN_DIR}" >&2
  exit 1
fi
TOTAL_DOCS=${#FILES[@]}

if [[ ${AUTOSCALE} -eq 1 ]]; then
  cpus=$(detect_cpus); [[ -z "$cpus" || "$cpus" -lt 1 ]] && cpus=1
  mem=$(detect_mem_mb); [[ -z "$mem" || "$mem" -lt 1024 ]] && mem=4096
  max_workers=$(( cpus > 1 ? cpus - 1 : 1 ))
  if [[ ${SHARDS_SET} -eq 0 ]]; then
    if (( cpus >= 16 )); then
      SHARDS=4
    elif (( cpus >= 8 )); then
      SHARDS=3
    elif (( cpus >= 4 )); then
      SHARDS=2
    else
      SHARDS=1
    fi
  fi
  (( SHARDS < 1 )) && SHARDS=1
  if (( SHARDS > TOTAL_DOCS )); then SHARDS=$TOTAL_DOCS; fi
  if [[ ${THREADS_SET} -eq 0 ]]; then
    threads_rec=$(( max_workers / SHARDS ))
    (( threads_rec < 1 )) && threads_rec=1
    if (( cpus >= 4 && threads_rec < 2 )); then threads_rec=2; fi
    (( threads_rec > 12 )) && threads_rec=12
    THREADS=$threads_rec
  fi
  if [[ ${XMX_SET} -eq 0 ]]; then
    divisor=$SHARDS
    (( divisor < 1 )) && divisor=1
    per_runner=$(( (mem * 70 / 100) / divisor ))
    (( per_runner < 4096 )) && per_runner=4096
    (( per_runner > 32768 )) && per_runner=32768
    XMX=$per_runner
  fi
  echo "[async] autoscale -> shards=${SHARDS}, threads=${THREADS:-"-"}, Xmx=${XMX:-"-"}MB" >&2
fi

if [[ -z "${SHARDS}" ]]; then SHARDS=1; fi
if (( SHARDS < 1 )); then SHARDS=1; fi
if (( SHARDS > TOTAL_DOCS )); then SHARDS=$TOTAL_DOCS; fi
if [[ -z "${THREADS}" ]]; then THREADS=2; fi
(( THREADS < 1 )) && THREADS=1
if [[ -z "${XMX}" ]]; then XMX=4096; fi

TIMESTAMP=$(date +%Y%m%d-%H%M%S)
BASE_OUT="${OUT_DIR%/}/${PIPELINE_KEY}/${TIMESTAMP}"
SHARDS_DIR="${BASE_OUT}/shards"
mkdir -p "$SHARDS_DIR"
declare -a SHARD_COUNTS

echo "[async] ===== Starting async run for pipeline '${PIPELINE_KEY}' ====="
printf '[async] total docs=%d | shards=%d | threads=%s | Xmx=%sMB\n' "$TOTAL_DOCS" "$SHARDS" "${THREADS:-"-"}" "${XMX:-"-"}"
printf '[async] input=%s -> output=%s\n' "$IN_DIR" "$OUT_DIR"

for ((i=0;i<SHARDS;i++)); do
  shard_dir=$(printf "%s/shard-%03d/input" "$SHARDS_DIR" "$i")
  mkdir -p "$shard_dir"
  SHARD_COUNTS[$i]=0
done

for ((idx=0; idx<TOTAL_DOCS; idx++)); do
  shard=$(( idx % SHARDS ))
  dest=$(printf "%s/shard-%03d/input" "$SHARDS_DIR" "$shard")
  src="${FILES[$idx]}"
  SHARD_COUNTS[$shard]=$(( ${SHARD_COUNTS[$shard]:-0} + 1 ))
  ln "$src" "$dest/" 2>/dev/null || cp -f "$src" "$dest/"
done

COMMON_ARGS=(--pipeline "$PIPELINE_KEY")
if [[ $WITH_RELATIONS -eq 1 ]]; then
  COMMON_ARGS+=(--with-relations)
fi
[[ -n $DICT_XML ]] && COMMON_ARGS+=(--dict "$DICT_XML")
[[ -n $UMLS_OVERRIDE ]] && COMMON_ARGS+=(--umls-key "$UMLS_OVERRIDE")
[[ -n $JAVA_OPTS_EXTRA ]] && COMMON_ARGS+=(--java-opts "$JAVA_OPTS_EXTRA")
[[ -n $THREADS ]] && COMMON_ARGS+=(--threads "$THREADS")
[[ -n $XMX ]] && COMMON_ARGS+=(--xmx "$XMX")

if [[ $DRY_RUN -eq 1 ]]; then
  for ((i=0;i<SHARDS;i++)); do
    in_dir=$(printf "%s/shard-%03d/input" "$SHARDS_DIR" "$i")
    out_dir=$(printf "%s/shard-%03d/output" "$SHARDS_DIR" "$i")
    printf '%q ' "${RUN_PIPELINE_CMD[@]}" --input "$in_dir" --output "$out_dir"
    printf '%q ' "${COMMON_ARGS[@]}"
    printf '\n'
  done
  exit 0
fi

PIDS=()
STATUS=0
trap 'for pid in "${PIDS[@]}"; do kill "$pid" 2>/dev/null || true; done' INT TERM

for ((i=0;i<SHARDS;i++)); do
  in_dir=$(printf "%s/shard-%03d/input" "$SHARDS_DIR" "$i")
  out_dir=$(printf "%s/shard-%03d/output" "$SHARDS_DIR" "$i")
  mkdir -p "$out_dir"
  docs=${SHARD_COUNTS[$i]:-0}
  log_file=$(printf "%s/shard-%03d/output/run.log" "$SHARDS_DIR" "$i")
  printf '[async] shard-%03d | docs=%d | threads=%s | Xmx=%sMB -> %s\n' "$i" "$docs" "${THREADS:-"-"}" "${XMX:-"-"}" "$out_dir"
  (
    printf '[async] shard-%03d started at %s\n' "$i" "$(date '+%Y-%m-%d %H:%M:%S')"
    "${RUN_PIPELINE_CMD[@]}" --input "$in_dir" --output "$out_dir" "${COMMON_ARGS[@]}"
    status=$?
    printf '[async] shard-%03d finished at %s (exit=%d)\n' "$i" "$(date '+%Y-%m-%d %H:%M:%S')" "$status"
    exit $status
  ) >"$log_file" 2>&1 &
  PIDS+=($!)
done
for pid in "${PIDS[@]}"; do
  if ! wait "$pid"; then STATUS=1; fi
done

if (( STATUS != 0 )); then
  echo "[async] One or more shards failed." >&2
fi

mkdir -p "$BASE_OUT/xmi" "$BASE_OUT/concepts" "$BASE_OUT/cui_counts"
if [[ $PIPELINE_KEY == "drug" ]]; then
  mkdir -p "$BASE_OUT/rxnorm"
fi

for shard_out in "$SHARDS_DIR"/shard-*/output; do
  [[ -d "$shard_out" ]] || continue
  if [[ -d "$shard_out/xmi" ]]; then
    cp -f "$shard_out"/xmi/* "$BASE_OUT/xmi/" 2>/dev/null || true
  fi
  if [[ -d "$shard_out/concepts" ]]; then
    cp -f "$shard_out"/concepts/* "$BASE_OUT/concepts/" 2>/dev/null || true
  fi
  if [[ -d "$shard_out/cui_counts" ]]; then
    cp -f "$shard_out"/cui_counts/* "$BASE_OUT/cui_counts/" 2>/dev/null || true
  fi
  if [[ -d "$shard_out/rxnorm" ]]; then
    mkdir -p "$BASE_OUT/rxnorm"
    cp -f "$shard_out"/rxnorm/* "$BASE_OUT/rxnorm/" 2>/dev/null || true
  fi
  if [[ -f "$shard_out/run.log" ]]; then
    mkdir -p "$BASE_OUT/logs"
    cp -f "$shard_out/run.log" "$BASE_OUT/logs/$(basename "$(dirname "$shard_out")").log" || true
  fi
done

combine_delimited_dir() {
  local src_dir="$1"; local dest_file="$2"; local extension="$3"
  local first_written=0
  >"$dest_file"
  shopt -s nullglob
  for file in "$src_dir"/*."${extension}" "$src_dir"/*."${extension^^}"; do
    [[ -f "$file" ]] || continue
    if [[ $first_written -eq 0 ]]; then
      cat "$file" >> "$dest_file"
      first_written=1
    else
      tail -n +2 "$file" >> "$dest_file"
    fi
  done
  shopt -u nullglob
  if [[ $first_written -eq 0 ]]; then rm -f "$dest_file"; fi
}
if compgen -G "$BASE_OUT/concepts/*.csv" >/dev/null 2>&1; then
  combine_delimited_dir "$BASE_OUT/concepts" "$BASE_OUT/concepts_summary.csv" csv
fi
if [[ -d "$BASE_OUT/rxnorm" ]] && compgen -G "$BASE_OUT/rxnorm/*.csv" >/dev/null 2>&1; then
  combine_delimited_dir "$BASE_OUT/rxnorm" "$BASE_OUT/rxnorm_summary.csv" csv
fi

if compgen -G "$BASE_OUT/cui_counts/*.bsv" >/dev/null 2>&1; then
  combine_delimited_dir "$BASE_OUT/cui_counts" "$BASE_OUT/cui_counts_summary.bsv" bsv
fi

printf '[async] Outputs ready at %s\n' "$BASE_OUT"
exit $STATUS


