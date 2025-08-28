#!/usr/bin/env bash
set -euo pipefail

# Run the WSD-enabled DefaultFast pipeline and write XMI + CSV/BSV/HTML tables.
# Uses the provided dictionary XML as-is (no sanitization by default).
# Usage:
#   scripts/run_wsd_smoke.sh [-i samples/input] [-o outputs/wsd_smoke] [-l <dictionary.xml>] [--no-report] [--key <UMLS_API_KEY>]

IN_DIR="samples/input"
OUT_DIR="outputs/wsd_smoke"
DICT_XML=""
UMLS_KEY="${UMLS_KEY:-}"
REPORT=1
while [[ $# -gt 0 ]]; do
  case "$1" in
    -i|--in) IN_DIR="$2"; shift 2;;
    -o|--out) OUT_DIR="$2"; shift 2;;
    -l|--lookup|-l|--dict-xml) DICT_XML="$2"; shift 2;;
    --key) UMLS_KEY="$2"; shift 2;;
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

XML_ARG="$DICT_XML"

IN_ABS="$(cd "$IN_DIR" && pwd)"
OUT_ABS="$(mkdir -p "$OUT_DIR" && cd "$OUT_DIR" && pwd)"
mkdir -p "$OUT_ABS/logs"
LOG="$OUT_ABS/logs/run_$(date +%Y%m%d_%H%M%S).log"

echo "CTAKES_HOME: $CTAKES_HOME"
echo "IN:          $IN_ABS"
echo "OUT:         $OUT_ABS"
echo "DICT_XML:    $XML_ARG"
echo "Log:         $LOG"

pushd "$CTAKES_HOME" >/dev/null
java -Xms2g -Xmx6g ${UMLS_KEY:+-Dctakes.umls_apikey=$UMLS_KEY} \
  -cp "$JAVA_CP" \
  org.apache.ctakes.core.pipeline.PiperFileRunner \
  -p "$(pwd -P)/../..//pipelines/wsd/TsDefaultFastPipeline_WSD.piper" \
  -i "$IN_ABS" \
  -o "$OUT_ABS" \
  -l "$XML_ARG" ${UMLS_KEY:+--key $UMLS_KEY} |& tee "$LOG"
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
  REPORT_NAME_BASE="ctakes-wsd-$(date +%Y%m%d-%H%M%S).xlsx"
  REPORT_PATH="$OUT_ABS/$REPORT_NAME_BASE"
  bash "$BASE_DIR/scripts/build_xlsx_report.sh" \
    -o "$OUT_ABS" \
    -p "$PIPELINE_PIPER" \
    -l "$LOG" \
    -d "$XML_ARG" \
    -w "$REPORT_PATH" || echo "WARN: report build failed (see logs)."
  echo "- Report:     $REPORT_PATH"
fi
