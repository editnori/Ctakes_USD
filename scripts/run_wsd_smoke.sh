#!/usr/bin/env bash
set -euo pipefail

# Run the WSD-enabled DefaultFast pipeline with offline dictionary and write XMI + CSV/BSV/HTML tables.
# Also builds an Excel-compatible workbook (XML) summarizing the run.
# Usage:
#   scripts/run_wsd_smoke.sh [-i samples/input] [-o outputs/wsd_smoke] [-l <dictionary.xml>] [--no-report]

IN_DIR="samples/input"
OUT_DIR="outputs/wsd_smoke"
DICT_XML=""
REPORT=1
while [[ $# -gt 0 ]]; do
  case "$1" in
    -i|--in) IN_DIR="$2"; shift 2;;
    -o|--out) OUT_DIR="$2"; shift 2;;
    -l|--lookup) DICT_XML="$2"; shift 2;;
    --no-report) REPORT=0; shift 1;;
    *) echo "Unknown arg: $1" >&2; exit 1;;
  esac
done

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
CTAKES_HOME="${CTAKES_HOME:-$BASE_DIR/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0}"
JAVA_CP="$CTAKES_HOME/desc:$CTAKES_HOME/resources:$CTAKES_HOME/config:$CTAKES_HOME/config/*:$CTAKES_HOME/lib/*:$BASE_DIR/.build_tools"

mkdir -p "$OUT_DIR/logs"
LOG="$OUT_DIR/logs/run_$(date +%Y%m%d_%H%M%S).log"

# Prefer the full dictionary built under cTAKES resources
if [[ -z "$DICT_XML" ]]; then
  DICT_XML="$CTAKES_HOME/resources/org/apache/ctakes/dictionary/lookup/fast/FullClinical_AllTUIs.xml"
fi

if [[ ! -f "$DICT_XML" ]]; then
  echo "Dictionary XML not found: $DICT_XML" >&2
  exit 1
fi

# Compile local tools (including our SimpleWsdDisambiguatorAnnotator)
find "$BASE_DIR/tools" -name "*.java" -print0 | xargs -0 javac -cp "$JAVA_CP" -d "$BASE_DIR/.build_tools"

# Create an offline+portable dictionary xml copy for this run
SAN_XML="$OUT_DIR/$(basename "$DICT_XML" .xml)_wsd_local.xml"
cp -f "$DICT_XML" "$SAN_XML"
sed -i -E \
  -e 's#<implementationName>org.apache.ctakes.dictionary.lookup2.dictionary.UmlsJdbcRareWordDictionary</implementationName>#<implementationName>org.apache.ctakes.dictionary.lookup2.dictionary.JdbcRareWordDictionary</implementationName>#' \
  -e 's#<implementationName>org.apache.ctakes.dictionary.lookup2.concept.UmlsJdbcConceptFactory</implementationName>#<implementationName>org.apache.ctakes.dictionary.lookup2.concept.JdbcConceptFactory</implementationName>#' \
  -e 's#<property key=\"jdbcDriver\" value=\"[^\"]*\"#<property key=\"jdbcDriver\" value=\"org.hsqldb.jdbc.JDBCDriver\"#' \
  -e '/<property key=\"umlsUrl\"/d' \
  -e '/<property key=\"umlsVendor\"/d' \
  -e '/<property key=\"umlsUser\"/d' \
  -e '/<property key=\"umlsPass\"/d' \
  "$SAN_XML"

# Ensure HSQL db path has no spaces; copy to /tmp if needed
SRC_DB="$CTAKES_HOME/resources/org/apache/ctakes/dictionary/lookup/fast/FullClinical_AllTUIs/FullClinical_AllTUIs"
if [[ -f "$SRC_DB.script" ]]; then
  TMP_DB="/tmp/ctakes_full/FullClinical_AllTUIs"
  mkdir -p "$(dirname "$TMP_DB")"
  cp -f "$SRC_DB.properties" "$TMP_DB.properties"
  cp -f "$SRC_DB.script" "$TMP_DB.script"
  sed -i -E "s#(key=\"jdbcUrl\" value=)\"[^\"]+\"#\1\"jdbc:hsqldb:file:${TMP_DB}\"#" "$SAN_XML"
fi

IN_ABS="$(cd "$IN_DIR" && pwd)"
OUT_ABS="$(mkdir -p "$OUT_DIR" && cd "$OUT_DIR" && pwd)"
mkdir -p "$OUT_ABS/logs"
LOG="$OUT_ABS/logs/run_$(date +%Y%m%d_%H%M%S).log"

echo "CTAKES_HOME: $CTAKES_HOME"
echo "IN:          $IN_ABS"
echo "OUT:         $OUT_ABS"
echo "DICT_XML:    $SAN_XML"
echo "Log:         $LOG"

pushd "$CTAKES_HOME" >/dev/null
java -Xms2g -Xmx6g \
  -cp "$JAVA_CP" \
  org.apache.ctakes.core.pipeline.PiperFileRunner \
  -p "$(pwd -P)/../..//pipelines/wsd/TsDefaultFastPipeline_WSD.piper" \
  -i "$IN_ABS" \
  -o "$OUT_ABS" \
  -l "$SAN_XML" |& tee "$LOG"
popd >/dev/null

echo "Outputs:"
echo "- XMI:        $OUT_ABS/xmi"
echo "- BSV:        $OUT_ABS/bsv_table"
echo "- CSV:        $OUT_ABS/csv_table"
echo "- HTML:       $OUT_ABS/html_table"
echo "- CUI List:   $OUT_ABS/cui_list"
echo "- CUI Count:  $OUT_ABS/cui_count"
echo "- Tokens:     $OUT_ABS/bsv_tokens"

# Build Excel-compatible XML workbook summary unless disabled
if [[ "$REPORT" -eq 1 ]]; then
  PIPELINE_PIPER="$BASE_DIR/pipelines/wsd/TsDefaultFastPipeline_WSD.piper"
  # Build report with a descriptive default name if -w not provided
  # Short report name to avoid Windows path limits
  REPORT_NAME_BASE="ctakes-wsd-$(date +%Y%m%d-%H%M%S).xml"
  REPORT_PATH="$OUT_ABS/$REPORT_NAME_BASE"
  bash "$BASE_DIR/scripts/build_xlsx_report.sh" \
    -o "$OUT_ABS" \
    -p "$PIPELINE_PIPER" \
    -l "$LOG" \
    -d "$SAN_XML" \
    -w "$REPORT_PATH" || echo "WARN: report build failed (see logs)."
  echo "- Report:     $REPORT_PATH"
fi
