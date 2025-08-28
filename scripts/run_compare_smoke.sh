#!/usr/bin/env bash
set -euo pipefail

# Run 5 comparison pipelines (WSD-enabled Ts variants) on a single note or input dir.
# Outputs go under outputs/compare/<combo>/.

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
CTAKES_HOME="${CTAKES_HOME:-$BASE_DIR/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0}"
# Prepend repo overrides/resources before cTAKES resources so we can override default configs (e.g., DefaultListRegex.bsv)
JAVA_CP="$BASE_DIR/resources_override:$BASE_DIR/resources:$CTAKES_HOME/desc:$CTAKES_HOME/resources:$CTAKES_HOME/config:$CTAKES_HOME/config/*:$CTAKES_HOME/lib/*:$BASE_DIR/.build_tools"

# Defaults
IN_PATH_RAW="$BASE_DIR/samples/input/note1.txt"
OUT_BASE_RAW="$BASE_DIR/outputs/compare"
DICT_XML_ARG="" # explicit dictionary xml (no sanitization)
UMLS_KEY_OPT="" # --key value if provided

# Support positional or flags (-i/--in, -o/--out)
args=("$@")
idx=0
if [[ ${#args[@]} -gt 0 && "${args[0]}" != -* ]]; then IN_PATH_RAW="${args[0]}"; idx=$((idx+1)); fi
if [[ ${#args[@]} -gt $idx && "${args[$idx]}" != -* ]]; then OUT_BASE_RAW="${args[$idx]}"; idx=$((idx+1)); fi
while [[ $idx -lt ${#args[@]} ]]; do
  case "${args[$idx]}" in
    -i|--in) IN_PATH_RAW="${args[$((idx+1))]:-}"; idx=$((idx+2));;
    -o|--out) OUT_BASE_RAW="${args[$((idx+1))]:-}"; idx=$((idx+2));;
    -l|--dict-xml) DICT_XML_ARG="${args[$((idx+1))]:-}"; idx=$((idx+2));;
    --key) UMLS_KEY_OPT="${args[$((idx+1))]:-}"; idx=$((idx+2));;
    *) echo "Unknown arg: ${args[$idx]}" >&2; exit 1;;
  esac
done

# Resolve absolute paths
if [[ -d "$IN_PATH_RAW" ]]; then IN_PATH="$(cd "$IN_PATH_RAW" && pwd)"; else IN_PATH="$(cd "$(dirname "$IN_PATH_RAW")" && pwd)/$(basename "$IN_PATH_RAW")"; fi
OUT_BASE="$(mkdir -p "$OUT_BASE_RAW" && cd "$OUT_BASE_RAW" && pwd)"

# Flight checks (smoke mode) — do not require sanitization
CTAKES_SANITIZE_DICT=0 bash "$BASE_DIR/scripts/flight_check.sh" --mode smoke || exit 1

# Detect availability of Temporal models (EventAnnotator requires model.jar on classpath)
HAS_TEMP_MODELS=0
# Check resources directory first (packaged in distribution)
if [[ -f "$CTAKES_HOME/resources/org/apache/ctakes/temporal/models/eventannotator/model.jar" ]]; then
  HAS_TEMP_MODELS=1
else
  # Fallback: check any lib jar contains the resource
  for J in "$CTAKES_HOME"/lib/*.jar; do
    if jar tf "$J" 2>/dev/null | grep -q "org/apache/ctakes/temporal/models/eventannotator/model.jar"; then
      HAS_TEMP_MODELS=1; break
    fi
  done
fi
if [[ "$HAS_TEMP_MODELS" -ne 1 ]]; then
  echo "[warn] Temporal model not found; temporal pipelines will be skipped."
  echo "       Expected resource: CTAKES_HOME/resources/org/apache/ctakes/temporal/models/eventannotator/model.jar"
fi

# Dictionary descriptor
PROPS="$BASE_DIR/docs/builder_full_clinical.properties"
DICT_NAME=$(awk -F= '/^\s*dictionary.name\s*=/{print $2}' "$PROPS" | tr -d '\r' | xargs)
if [[ -z "$DICT_NAME" ]]; then DICT_NAME="FullClinical_AllTUIs"; fi
if [[ -n "$DICT_XML_ARG" ]]; then
  DICT_XML="$DICT_XML_ARG"
else
  DICT_XML="$CTAKES_HOME/resources/org/apache/ctakes/dictionary/lookup/fast/${DICT_NAME}.xml"
fi
if [[ ! -f "$DICT_XML" ]]; then
  echo "Dictionary XML not found: $DICT_XML" >&2
  exit 1
fi

# Compile local tools (WSD AE)
find "$BASE_DIR/tools" -name "*.java" -print0 | xargs -0 javac -cp "$JAVA_CP" -d "$BASE_DIR/.build_tools"

prep_smoking_desc() {
  local jar="$CTAKES_HOME/lib/ctakes-smoking-status-6.0.0.jar"
  local dst="$CTAKES_HOME/desc/org/apache/ctakes/smoking/status/analysis_engine"
  mkdir -p "$dst"
  # Extract required descriptors if missing
  for f in \
    ProductionPostSentenceAggregate_step1.xml \
    ProductionPostSentenceAggregate_step2_libsvm.xml \
    KuRuleBasedClassifierAnnotator.xml \
    PcsClassifierAnnotator_libsvm.xml \
    SentenceAdjuster.xml \
    ArtificialSentenceAnnotator.xml \
    SmokingStatusDictionaryLookupAnnotator.xml; do
    if [[ ! -f "$dst/$f" ]]; then
      (cd "$CTAKES_HOME/desc" && jar xf "$jar" org/apache/ctakes/smoking/status/analysis_engine/$f)
    fi
  done
}

# Use explicit dictionary XML as-is; no sanitization by default. You may still enable
# sanitization by exporting CTAKES_SANITIZE_DICT=1 before running, but the default is pass-through.
resolve_xml() {
  local outdir="$1"
  if [[ "${CTAKES_SANITIZE_DICT:-0}" == "1" ]]; then
    local san="$outdir/${DICT_NAME}_local.xml"
    cp -f "$DICT_XML" "$san"
    echo "$san"
  else
    echo "$DICT_XML"
  fi
}

run() {
  local name="$1"; local piper="$2"; local in="$3"; local out="$4"
  mkdir -p "$out"
  out="$(cd "$out" && pwd)"
  local xml
  xml=$(resolve_xml "$out")
  # Ensure EventAnnotator default model path is available (some releases package under models/, not ae/)
  local event_src="$CTAKES_HOME/resources/org/apache/ctakes/temporal/models/eventannotator/model.jar"
  local event_dst="$CTAKES_HOME/resources/org/apache/ctakes/temporal/ae/eventannotator/model.jar"
  if [[ -f "$event_src" && ! -f "$event_dst" ]]; then
    mkdir -p "$(dirname "$event_dst")"
    cp -f "$event_src" "$event_dst"
  fi
  if [[ -d "$in" ]]; then in=$(cd "$in" && pwd); else in=$(cd "$(dirname "$in")" && pwd)/"$(basename "$in")"; fi
  pushd "$CTAKES_HOME" >/dev/null
  java -Xms2g -Xmx6g ${UMLS_KEY_OPT:+-Dctakes.umls_apikey=$UMLS_KEY_OPT} -cp "$JAVA_CP" \
    org.apache.ctakes.core.pipeline.PiperFileRunner \
    -p "$piper" -i "$in" -o "$out" -l "$xml" ${UMLS_KEY_OPT:+--key $UMLS_KEY_OPT} |& tee "$out/run.log"
  popd >/dev/null

  # Build per-pipeline Excel XML workbook with a short name to avoid Windows path limits
  local stamp="$(date +%Y%m%d-%H%M%S)"
  local report_name="ctakes-${name}-${stamp}.xml"
  if grep -q "ResourceInitializationException" "$out/run.log"; then
    echo "WARN: $name failed to initialize one or more AEs; skipping report. See $out/run.log"
  else
    bash "$BASE_DIR/scripts/build_xlsx_report.sh" -o "$out" -p "$piper" -l "$out/run.log" -d "$xml" -w "$out/$report_name" || \
      echo "WARN: report build failed for $name (see logs)."
    echo "- Report:     $out/$report_name"
  fi
}

# Clear previous compare outputs
rm -rf "$OUT_BASE"/*

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
if [[ "$HAS_TEMP_MODELS" -eq 1 ]]; then
  keys+=(S_core_temp S_core_temp_coref D_core_temp D_core_temp_coref S_core_temp_coref_smoke D_core_temp_coref_smoke)
fi

for key in "${keys[@]}"; do
  # Ensure smoking descriptors exist before running smoking pipelines
  if [[ "$key" == *"smoke"* ]]; then prep_smoking_desc; fi
  run "$key" "${SETS[$key]}" "$IN_PATH" "$OUT_BASE/$key"
done

echo "Compare outputs in: $OUT_BASE"

# Build a combined Pipelines Summary workbook at the parent level
STAMP="$(date +%Y%m%d-%H%M%S)"
PARENT_REPORT="$OUT_BASE/ctakes-report-compare-${STAMP}.xml"
bash "$BASE_DIR/scripts/build_xlsx_report.sh" -o "$OUT_BASE" -w "$PARENT_REPORT" || \
  echo "WARN: parent summary report build failed (see logs)."
echo "- Summary:    $PARENT_REPORT"
