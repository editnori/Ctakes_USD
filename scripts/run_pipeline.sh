#!/usr/bin/env bash
set -euo pipefail

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
ENV_FILE="${BASE_DIR}/.ctakes_env"
if [[ -f "${ENV_FILE}" ]]; then
  # shellcheck disable=SC1090
  source "${ENV_FILE}"
fi


DEFAULT_UMLS_KEY="6370dcdd-d438-47ab-8749-5a8fb9d013f2"

usage() {
  cat <<'USAGE'
Usage: scripts/run_pipeline.sh -i <input_dir> -o <output_dir> [options]
Options:
  --pipeline <core|sectioned|smoke|drug|core_sectioned_smoke>   Pipeline to execute (default: sectioned)
  --with-temporal                          Insert TsTemporalSubPipe before writers
  --with-coref                             Insert TsCorefSubPipe before writers
  --threads <N>                            Override the Piper "threads" clause
  --xmx <MB>                               Heap size per run (overrides autoscale)
  --java-opts "..."                       Extra JVM options to append
  --dict <file.xml>                        Dictionary Lookup XML (defaults to bundled FullClinical_AllTUIs[_local].xml)
  --umls-key <KEY>                         Override the UMLS API key for dictionary building
  --autoscale                              Recommend threads/heap based on host resources (default)
  --no-autoscale                           Disable autoscale heuristics
  --dry-run                                Print the Java command instead of executing
  --help                                   Show this message

Environment:
  CTAKES_HOME       Path to apache cTAKES install root. Defaults to bundled CtakesBun copy when present.
  CTAKES_JAVA_OPTS  Prepended to the JVM invocation; --java-opts and --xmx append to it.
  UMLS_KEY          Default UMLS key if --umls-key is omitted.
USAGE
}

# Platform helpers -----------------------------------------------------------
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

recommend_autoscale() {
  local cpus mem threads_rec xmx_rec
  cpus=$(detect_cpus); [[ -z "$cpus" || "$cpus" -lt 1 ]] && cpus=1
  mem=$(detect_mem_mb); [[ -z "$mem" || "$mem" -lt 1024 ]] && mem=4096
  threads_rec=$cpus
  if (( cpus >= 8 )); then
    threads_rec=$(( cpus - 2 ))
  elif (( cpus >= 4 )); then
    threads_rec=$(( cpus - 1 ))
  fi
  (( threads_rec < 1 )) && threads_rec=1
  (( threads_rec > 12 )) && threads_rec=12
  xmx_rec=$(( mem * 70 / 100 ))
  (( xmx_rec < 4096 )) && xmx_rec=4096
  (( xmx_rec > 32768 )) && xmx_rec=32768
  AUTOSCALE_THREADS="$threads_rec"
  AUTOSCALE_XMX="$xmx_rec"
  echo "[autoscale] CPU cores=${cpus}, RAM=${mem}MB -> threads=${threads_rec}, Xmx=${xmx_rec}MB" >&2
}

PIPELINE_KEY="sectioned"
WITH_TEMPORAL=0
WITH_COREF=0
THREAD_OVERRIDE=""
XMX_MB=""
JAVA_OPTS_EXTRA=""
DICT_XML=""
UMLS_KEY_OVERRIDE=""
AUTOSCALE=1
DRY_RUN=0
IN_DIR=""
OUT_DIR=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    -i|--input) IN_DIR="$2"; shift 2;;
    -o|--output) OUT_DIR="$2"; shift 2;;
    --pipeline) PIPELINE_KEY="$2"; shift 2;;
    --with-temporal) WITH_TEMPORAL=1; shift 1;;
    --with-coref) WITH_COREF=1; shift 1;;
    --threads) THREAD_OVERRIDE="$2"; shift 2;;
    --xmx) XMX_MB="$2"; shift 2;;
    --java-opts) JAVA_OPTS_EXTRA="$2"; shift 2;;
    --dict) DICT_XML="$2"; shift 2;;
    --umls-key) UMLS_KEY_OVERRIDE="$2"; shift 2;;
    --autoscale) AUTOSCALE=1; shift 1;;
    --no-autoscale) AUTOSCALE=0; shift 1;;
    --dry-run) DRY_RUN=1; shift 1;;
    --help|-h) usage; exit 0;;
    *) echo "Unknown option: $1" >&2; usage >&2; exit 1;;
  esac

done

if [[ -z "${IN_DIR}" || -z "${OUT_DIR}" ]]; then
  echo "[pipeline] --input and --output are required" >&2
  usage >&2
  exit 1
fi


if [[ -z "${CTAKES_HOME:-}" ]]; then
  bundle_home="${BASE_DIR}/CtakesBun-bundle/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0"
  bundle_home_alt="${BASE_DIR}/Ctakes_USD_clean/CtakesBun-bundle/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0"
  if [[ -d "${bundle_home}" ]]; then
    export CTAKES_HOME="${bundle_home}"
    echo "[pipeline] CTAKES_HOME not set; defaulting to bundled ${CTAKES_HOME}" >&2
  elif [[ -d "${bundle_home_alt}" ]]; then
    export CTAKES_HOME="${bundle_home_alt}"
    echo "[pipeline] CTAKES_HOME not set; found bundle under Ctakes_USD_clean/; using ${CTAKES_HOME}" >&2
  else
    echo "[pipeline] Set CTAKES_HOME or run scripts/get_bundle.sh to download the bundled distribution" >&2
    exit 1
  fi
fi

DATA_PATH_BASE="${BASE_DIR}/resources_override:${BASE_DIR}/resources"
if [[ -n "${CTAKES_HOME:-}" ]]; then
  DATA_PATH_BASE="${DATA_PATH_BASE}:${CTAKES_HOME}/resources:${CTAKES_HOME}/desc"
fi
if [[ -n "${UIMA_DATAPATH:-}" ]]; then
  export UIMA_DATAPATH="${DATA_PATH_BASE}:${UIMA_DATAPATH}"
else
  export UIMA_DATAPATH="${DATA_PATH_BASE}"
fi

case "${PIPELINE_KEY}" in
  core) PIPER="${BASE_DIR}/pipelines/core/core_wsd.piper";;
  sectioned) PIPER="${BASE_DIR}/pipelines/sectioned/sectioned_core_wsd.piper";;
  smoke) PIPER="${BASE_DIR}/pipelines/smoke/sectioned_smoke_status.piper";;
  core_sectioned_smoke) PIPER="${BASE_DIR}/pipelines/combined/core_sectioned_smoke.piper";;
  drug) PIPER="${BASE_DIR}/pipelines/drug/drug_ner_wsd.piper";;
  *) echo "[pipeline] Unknown pipeline key: ${PIPELINE_KEY}" >&2; exit 1;;
esac

[[ -f "${PIPER}" ]] || { echo "[pipeline] Missing pipeline file: ${PIPER}" >&2; exit 1; }
[[ -d "${IN_DIR}" ]] || { echo "[pipeline] Input directory does not exist: ${IN_DIR}" >&2; exit 1; }
mkdir -p "${OUT_DIR}"

if [[ ${AUTOSCALE} -eq 1 ]]; then
  recommend_autoscale
  if [[ -z "${THREAD_OVERRIDE}" ]]; then
    THREAD_OVERRIDE="${AUTOSCALE_THREADS}"
    echo "[pipeline] Autoscale recommends ${THREAD_OVERRIDE} thread(s)." >&2
  fi
  if [[ -z "${XMX_MB}" ]]; then
    XMX_MB="${AUTOSCALE_XMX}"
    echo "[pipeline] Autoscale recommends ${XMX_MB} MB heap." >&2
  fi
fi

[[ -z "${THREAD_OVERRIDE}" ]] && THREAD_OVERRIDE=2
[[ -z "${XMX_MB}" ]] && XMX_MB=4096

if [[ -z "${DICT_XML}" ]]; then
  default_dict="${CTAKES_HOME}/resources/org/apache/ctakes/dictionary/lookup/fast/FullClinical_AllTUIs_local.xml"
  if [[ -f "$default_dict" ]]; then
    DICT_XML="$default_dict"
  else
    default_dict="${CTAKES_HOME}/resources/org/apache/ctakes/dictionary/lookup/fast/FullClinical_AllTUIs.xml"
    if [[ -f "$default_dict" ]]; then
      DICT_XML="$default_dict"
    fi
  fi
fi

if [[ -z "${DICT_XML}" ]]; then
  echo "[pipeline] Could not locate a dictionary XML. Pass --dict <path>." >&2
  exit 1
fi

if [[ ! -f "${DICT_XML}" ]]; then
  echo "[pipeline] Dictionary XML not found: ${DICT_XML}" >&2
  exit 1
fi

BUILD_DIR="${BASE_DIR}/build/tools"
mkdir -p "$BUILD_DIR"
CLASSPATH_SEP=':'
case "$(uname -s 2>/dev/null)" in
  MINGW*|MSYS*|CYGWIN*) CLASSPATH_SEP=';';;
  *) CLASSPATH_SEP=':';;
esac

JAVA_CP_COMPILE="${CTAKES_HOME}/desc${CLASSPATH_SEP}${CTAKES_HOME}/resources${CLASSPATH_SEP}${CTAKES_HOME}/lib/*"
if command -v find >/dev/null 2>&1; then
  mapfile -t JAVA_SOURCES < <(find "${BASE_DIR}/tools" -type f -name '*.java')
else
  JAVA_SOURCES=()
fi
if [[ ${#JAVA_SOURCES[@]} -gt 0 ]]; then
  javac -cp "$JAVA_CP_COMPILE" -d "$BUILD_DIR" "${JAVA_SOURCES[@]}"
fi

TMP_PIPE="$(mktemp "${TMPDIR:-/tmp}/ctakes_pipeline.${PIPELINE_KEY}.XXXXXX.piper")"
cleanup() { rm -f "${TMP_PIPE}"; }
trap cleanup EXIT

OPTIONAL_LINES=()
[[ ${WITH_TEMPORAL} -eq 1 ]] && OPTIONAL_LINES+=("load TsTemporalSubPipe")
[[ ${WITH_COREF} -eq 1 ]] && OPTIONAL_LINES+=("load TsCorefSubPipe")

>"${TMP_PIPE}"
while IFS='' read -r line; do
  if [[ "${line}" == "// OPTIONAL_MODULES" ]]; then
    if [[ ${#OPTIONAL_LINES[@]} -gt 0 ]]; then
      for opt in "${OPTIONAL_LINES[@]}"; do
        printf '%s\n' "$opt" >> "${TMP_PIPE}"
      done
    fi
  elif [[ ${line} =~ ^[[:space:]]*threads[[:space:]]+ ]]; then
    printf 'threads %s\n' "${THREAD_OVERRIDE}" >> "${TMP_PIPE}"
  else
    printf '%s\n' "${line}" >> "${TMP_PIPE}"
  fi
done < "${PIPER}"

JAVA_CP_RUN="${BUILD_DIR}${CLASSPATH_SEP}${BASE_DIR}/resources_override${CLASSPATH_SEP}${BASE_DIR}/resources${CLASSPATH_SEP}${CTAKES_HOME}/desc${CLASSPATH_SEP}${CTAKES_HOME}/resources${CLASSPATH_SEP}${CTAKES_HOME}/config${CLASSPATH_SEP}${CTAKES_HOME}/config/*${CLASSPATH_SEP}${CTAKES_HOME}/lib/*"
JAVA_CMD=(java -cp "$JAVA_CP_RUN" -Xms${XMX_MB}m -Xmx${XMX_MB}m -XX:+UseG1GC -XX:+ParallelRefProcEnabled -XX:+UseStringDeduplication -XX:MaxGCPauseMillis=200)

if [[ -n "${CTAKES_JAVA_OPTS:-}" ]]; then
  EXTRA_OPTS=( ${CTAKES_JAVA_OPTS} )
  JAVA_CMD+=("${EXTRA_OPTS[@]}")
fi
if [[ -n "${JAVA_OPTS_EXTRA}" ]]; then
  EXTRA_OPTS=( ${JAVA_OPTS_EXTRA} )
  JAVA_CMD+=("${EXTRA_OPTS[@]}")
fi

if [[ -n "${UMLS_KEY_OVERRIDE}" ]]; then
  JAVA_CMD+=(-Dctakes.umls_apikey="${UMLS_KEY_OVERRIDE}")
elif [[ -n "${UMLS_KEY:-}" ]]; then
  JAVA_CMD+=(-Dctakes.umls_apikey="${UMLS_KEY}")
fi

if [[ -n "${UIMA_DATAPATH:-}" ]]; then
  JAVA_CMD+=(-Duima.datapath="${UIMA_DATAPATH}")
fi

JAVA_CMD+=(-Dorg.slf4j.simpleLogger.defaultLogLevel=info)
JAVA_CMD+=(-Dorg.slf4j.simpleLogger.log.org.apache.ctakes.dictionary=warn)
JAVA_CMD+=(-Dorg.slf4j.simpleLogger.log.org.apache.ctakes.dictionary.lookup2=warn)
JAVA_CMD+=(-Dorg.slf4j.simpleLogger.log.org.apache.uima=warn)
JAVA_CMD+=(-Dorg.slf4j.simpleLogger.log.org.cleartk=warn)
JAVA_CMD+=(-Dorg.slf4j.simpleLogger.log.opennlp=warn)
JAVA_CMD+=(-Dorg.slf4j.simpleLogger.log.org.apache.ctakes.core.ae.RegexSpanFinder=warn)
JAVA_CMD+=(-Dorg.slf4j.simpleLogger.log.org.apache.uima.cas.impl.XmiCasSerializer=${XMI_LOG_LEVEL:-warn})

JAVA_CMD+=(org.apache.ctakes.core.pipeline.PiperFileRunner -p "${TMP_PIPE}" -i "${IN_DIR}" -o "${OUT_DIR}" -l "${DICT_XML}")

if [[ ${DRY_RUN} -eq 1 ]]; then
  printf '[pipeline] '
  printf '%q ' "${JAVA_CMD[@]}"
  printf '\n'
  exit 0
fi

"${JAVA_CMD[@]}"
