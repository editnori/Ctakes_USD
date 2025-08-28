#!/usr/bin/env bash
set -euo pipefail

# Run all pipelines in docs/stress_test_plan.csv (Ts-only), resolving to local XMI-only
# smoke pipelines. Writes results CSV with elapsed seconds and XMI counts.

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
CTAKES_HOME="${CTAKES_HOME:-$BASE_DIR/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0}"
PLAN="$BASE_DIR/docs/stress_test_plan.csv"
# Default dictionary xml under cTAKES resources; override by passing DICT_XML as 1st arg
PROPS="$BASE_DIR/docs/builder_full_clinical.properties"
DICT_NAME=$(awk -F= '/^\s*dictionary.name\s*=/{print $2}' "$PROPS" | tr -d '\r' | xargs)
DICT_XML_DEFAULT="$CTAKES_HOME/resources/org/apache/ctakes/dictionary/lookup/fast/${DICT_NAME}.xml"
DICT_XML="${1:-$DICT_XML_DEFAULT}"

# UMLS key is not required for offline/local dictionaries; if provided, we pass it through.
UMLS_KEY="${UMLS_KEY:-}"
IN_DIR="${2:-$BASE_DIR/samples/input}"
OUT_BASE="${3:-$BASE_DIR/outputs/stress_full}"

mkdir -p "$OUT_BASE"
RESULTS="$OUT_BASE/results_$(date +%Y%m%d_%H%M%S).csv"
echo "pipeline,resolved_pipeline,order,elapsed_sec,xmi_count,out_dir" > "$RESULTS"

sanitize_dict() { cp -f "$1" "$2" 2>/dev/null || true; echo "${2:-$1}"; }

resolve_pipeline() {
  local path="$1"
  case "$path" in
    org/apache/ctakes/clinical/pipeline/TsDefaultFastPipeline.piper)
      echo "$BASE_DIR/pipelines/smoke/TsDefaultFast_XmiOnly.piper";;
    org/apache/ctakes/relation/extractor/pipeline/TsDefaultRelationPipeline.piper)
      echo "$BASE_DIR/pipelines/smoke/TsDefaultRelation_XmiOnly.piper";;
    org/apache/ctakes/temporal/pipeline/TsDefaultTemporalPipeline.piper)
      echo "$BASE_DIR/pipelines/smoke/TsDefaultTemporal_XmiOnly.piper";;
    org/apache/ctakes/coreference/pipeline/TsDefaultTemporalCorefPipeline.piper)
      echo "$BASE_DIR/pipelines/smoke/TsDefaultTemporalCoref_XmiOnly.piper";;
    *)
      echo "$path";;
  esac
}

# Skip header
IFS=,
{ read -r _; while read -r Pipeline Type Jar Path rest; do
  [[ "$Type" != "full" ]] && continue
  p="$Path"
  name=$(basename "$p" .piper)
  outdir="$OUT_BASE/$name"
  mkdir -p "$outdir"

  resolved=$(resolve_pipeline "$p")
  XML_ARG="$DICT_XML"
  # Use provided XML as-is (no sanitization/relocation by default)
  start=$(date +%s)
  echo "==> $name"
  if [[ -f "$resolved" ]]; then
    # Local piper file
    java -Xms2g -Xmx6g ${UMLS_KEY:+-Dctakes.umls_apikey=$UMLS_KEY} \
      -cp "$CTAKES_HOME/desc:$CTAKES_HOME/resources:$CTAKES_HOME/config:$CTAKES_HOME/config/*:$CTAKES_HOME/lib/*:$BASE_DIR/.build_tools" \
      org.apache.ctakes.core.pipeline.PiperFileRunner \
      -p "$resolved" -i "$IN_DIR" -o "$outdir" -l "$XML_ARG" ${UMLS_KEY:+--key $UMLS_KEY} || true
  else
    # Classpath piper
    java -Xms2g -Xmx6g ${UMLS_KEY:+-Dctakes.umls_apikey=$UMLS_KEY} \
      -cp "$CTAKES_HOME/desc:$CTAKES_HOME/resources:$CTAKES_HOME/config:$CTAKES_HOME/config/*:$CTAKES_HOME/lib/*:$BASE_DIR/.build_tools" \
      org.apache.ctakes.core.pipeline.PiperFileRunner \
      -p "$p" -i "$IN_DIR" -o "$outdir" -l "$XML_ARG" ${UMLS_KEY:+--key $UMLS_KEY} || true
  fi
  end=$(date +%s)
  elapsed=$((end-start))
  xmi_count=$(find "$outdir" -type f -name "*.xmi" | wc -l | xargs)
  echo "$Pipeline,$resolved,,$elapsed,$xmi_count,$outdir" >> "$RESULTS"

done; } < "$PLAN"

echo "Results written to $RESULTS"
