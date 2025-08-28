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

sanitize_dict() {
  local in="$1"
  local out="$2"
  if [[ -f "$in" ]]; then
    sed -e 's#<implementationName>org.apache.ctakes.dictionary.lookup2.dictionary.UmlsJdbcRareWordDictionary</implementationName>#<implementationName>org.apache.ctakes.dictionary.lookup2.dictionary.JdbcRareWordDictionary</implementationName>#' \
        -e 's#<implementationName>org.apache.ctakes.dictionary.lookup2.concept.UmlsJdbcConceptFactory</implementationName>#<implementationName>org.apache.ctakes.dictionary.lookup2.concept.JdbcConceptFactory</implementationName>#' \
        -e 's#<property name=\"jdbcDriver\" value=\"[^\"]*\"#<property name=\"jdbcDriver\" value=\"org.hsqldb.jdbc.JDBCDriver\"#' \
        -e '/<property name=\"umlsUrl\"/d' \
        -e '/<property name=\"umlsVendor\"/d' \
        -e '/<property name=\"umlsUser\"/d' \
        -e '/<property name=\"umlsPass\"/d' \
        "$in" > "$out"
    echo "$out"
  else
    # Might be a classpath resource; just return as-is
    echo "$in"
  fi
}

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
  # Use a sanitized local XML that doesn't require UMLS auth when the path is a real file
  XML_ARG="$DICT_XML"
  if [[ -f "$DICT_XML" ]]; then
    tmpxml="$outdir/$(basename "$DICT_XML" .xml)_local.xml"
    XML_ARG=$(sanitize_dict "$DICT_XML" "$tmpxml")
    # If a source HSQL DB exists adjacent to the descriptor under resources, relocate to /tmp and rewrite jdbcUrl
    SRC_DB_DIR="$CTAKES_HOME/resources/org/apache/ctakes/dictionary/lookup/fast/$DICT_NAME"
    if [[ -f "$SRC_DB_DIR/$DICT_NAME.script" ]]; then
      TMP_DB="/tmp/ctakes_full/$DICT_NAME"
      mkdir -p "$(dirname "$TMP_DB")"
      cp -f "$SRC_DB_DIR/$DICT_NAME.properties" "$TMP_DB.properties"
      cp -f "$SRC_DB_DIR/$DICT_NAME.script" "$TMP_DB.script"
      sed -i -E "s#(key=\"jdbcUrl\" value=)\"[^\"]+\"#\1\"jdbc:hsqldb:file:${TMP_DB}\"#" "$XML_ARG"
      sed -i -E "s#(key=\"jdbcDriver\" value=)\"[^\"]+\"#\1\"org.hsqldb.jdbc.JDBCDriver\"#" "$XML_ARG"
    fi
  fi
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
