#!/usr/bin/env bash
set -euo pipefail

# Run all "full" pipelines from docs/ctakes_catalog.csv.
# Prefers local WSD variants when available and writes XMI + CSV/BSV/HTML (where pipeline includes writers).
# Dictionary XML is used as-is (no sanitization). You can pass a key via --key.
# Usage: scripts/run_catalog_smoke.sh [-i samples/input] [-o outputs/catalog_smoke] [--key <UMLS_API_KEY>] [DICT_XML]

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
CTAKES_HOME="${CTAKES_HOME:-$BASE_DIR/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0}"
CAT="$BASE_DIR/docs/ctakes_catalog.csv"
IN_DIR="samples/input"
OUT_DIR="outputs/catalog_smoke"
DICT_XML=""
UMLS_KEY="${UMLS_KEY:-}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    -i|--in) IN_DIR="$2"; shift 2;;
    -o|--out) OUT_DIR="$2"; shift 2;;
    --key) UMLS_KEY="$2"; shift 2;;
    *) DICT_XML="$1"; shift;;
  esac
done

PROPS="$BASE_DIR/docs/builder_full_clinical.properties"
DICT_NAME=$(awk -F= '/^\s*dictionary.name\s*=/{print $2}' "$PROPS" | tr -d '\r' | xargs)
DICT_XML_DEFAULT="$CTAKES_HOME/resources/org/apache/ctakes/dictionary/lookup/fast/${DICT_NAME}.xml"
if [[ -z "$DICT_XML" ]]; then DICT_XML="$DICT_XML_DEFAULT"; fi

JAVA_CP="$CTAKES_HOME/desc:$CTAKES_HOME/resources:$CTAKES_HOME/config:$CTAKES_HOME/config/*:$CTAKES_HOME/lib/*:$BASE_DIR/.build_tools"
IN_ABS="$(cd "$IN_DIR" && pwd)"
OUT_ABS="$(mkdir -p "$OUT_DIR" && cd "$OUT_DIR" && pwd)"
RESULTS="$OUT_ABS/results_$(date +%Y%m%d_%H%M%S).csv"
echo "pipeline,resolved_pipeline,elapsed_sec,xmi_count,out_dir" > "$RESULTS"

sanitize_dict() { cp -f "$1" "$2"; } # no-op (kept for compatibility)

resolve_pipeline() {
  local p="$1"
  case "$p" in
    org/apache/ctakes/clinical/pipeline/DefaultFastPipeline.piper)
      echo "$BASE_DIR/pipelines/wsd/DefaultFastPipeline_WSD.piper";;
    org/apache/ctakes/clinical/pipeline/SectionedFastPipeline.piper)
      echo "$BASE_DIR/pipelines/wsd/SectionedFastPipeline_WSD.piper";;
    org/apache/ctakes/clinical/pipeline/TsDefaultFastPipeline.piper)
      echo "$BASE_DIR/pipelines/wsd/TsDefaultFastPipeline_WSD.piper";;
    org/apache/ctakes/clinical/pipeline/TsSectionedFastPipeline.piper)
      echo "$BASE_DIR/pipelines/wsd/TsSectionedFastPipeline_WSD.piper";;
    org/apache/ctakes/temporal/pipeline/DefaultTemporalPipeline.piper)
      echo "$BASE_DIR/pipelines/wsd/DefaultTemporalPipeline_WSD.piper";;
    org/apache/ctakes/temporal/pipeline/SectionedTemporalPipeline.piper)
      echo "$BASE_DIR/pipelines/wsd/SectionedTemporalPipeline_WSD.piper";;
    org/apache/ctakes/temporal/pipeline/TsDefaultTemporalPipeline.piper)
      echo "$BASE_DIR/pipelines/wsd/TsDefaultTemporalPipeline_WSD.piper";;
    org/apache/ctakes/temporal/pipeline/TsSectionedTemporalPipeline.piper)
      echo "$BASE_DIR/pipelines/wsd/TsSectionedTemporalPipeline_WSD.piper";;
    org/apache/ctakes/coreference/pipeline/DefaultTemporalCorefPipeline.piper)
      echo "$BASE_DIR/pipelines/wsd/DefaultTemporalCorefPipeline_WSD.piper";;
    org/apache/ctakes/coreference/pipeline/SectionedTemporalCorefPipeline.piper)
      echo "$BASE_DIR/pipelines/wsd/SectionedTemporalCorefPipeline_WSD.piper";;
    org/apache/ctakes/coreference/pipeline/TsDefaultTemporalCorefPipeline.piper)
      echo "$BASE_DIR/pipelines/wsd/TsDefaultTemporalCorefPipeline_WSD.piper";;
    org/apache/ctakes/coreference/pipeline/TsSectionedTemporalCorefPipeline.piper)
      echo "$BASE_DIR/pipelines/wsd/TsSectionedTemporalCorefPipeline_WSD.piper";;
    org/apache/ctakes/relation/extractor/pipeline/DefaultRelationPipeline.piper)
      echo "$BASE_DIR/pipelines/wsd/DefaultRelationPipeline_WSD.piper";;
    org/apache/ctakes/relation/extractor/pipeline/SectionedRelationPipeline.piper)
      echo "$BASE_DIR/pipelines/wsd/SectionedRelationPipeline_WSD.piper";;
    org/apache/ctakes/relation/extractor/pipeline/TsDefaultRelationPipeline.piper)
      echo "$BASE_DIR/pipelines/wsd/TsDefaultRelationPipeline_WSD.piper";;
    org/apache/ctakes/relation/extractor/pipeline/TsSectionedRelationPipeline.piper)
      echo "$BASE_DIR/pipelines/wsd/TsSectionedRelationPipeline_WSD.piper";;
    *) echo "$p";;
  esac
}

# compile tools
find "$BASE_DIR/tools" -name "*.java" -print0 | xargs -0 javac -cp "$JAVA_CP" -d "$BASE_DIR/.build_tools"

# run full pipelines from catalog
LIMIT="${LIMIT:-}"
COUNT=0
awk -F, 'NR>1 && $2=="full" {print $4}' "$CAT" | while read -r P; do
  COUNT=$((COUNT+1))
  if [[ -n "$LIMIT" && $COUNT -gt $LIMIT ]]; then break; fi
  name=$(basename "$P" .piper)
  out="$OUT_ABS/$name"; mkdir -p "$out"
  resolved=$(resolve_pipeline "$P")
  XML_ARG="$DICT_XML"
  start=$(date +%s)
  if [[ -f "$resolved" ]]; then
    java -Xms2g -Xmx6g ${UMLS_KEY:+-Dctakes.umls_apikey=$UMLS_KEY} -cp "$JAVA_CP" org.apache.ctakes.core.pipeline.PiperFileRunner -p "$resolved" -i "$IN_ABS" -o "$out" -l "$XML_ARG" ${UMLS_KEY:+--key $UMLS_KEY} || true
  else
    java -Xms2g -Xmx6g ${UMLS_KEY:+-Dctakes.umls_apikey=$UMLS_KEY} -cp "$JAVA_CP" org.apache.ctakes.core.pipeline.PiperFileRunner -p "$P" -i "$IN_ABS" -o "$out" -l "$XML_ARG" ${UMLS_KEY:+--key $UMLS_KEY} || true
  fi
  end=$(date +%s)
  elapsed=$((end-start))
  xmi_count=$(find "$out" -type f -name '*.xmi' | wc -l | xargs)
  echo "$P,$resolved,$elapsed,$xmi_count,$out" >> "$RESULTS"
done

echo "Results: $RESULTS"
