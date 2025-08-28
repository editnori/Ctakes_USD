#!/usr/bin/env bash
set -euo pipefail

# Build an Excel-compatible XML workbook (multi-sheet) consolidating outputs for a run.
# Usage: scripts/build_xlsx_report.sh -o <run_output_dir> [-p <pipeline.piper>] [-l <run.log>] [-d <dict.xml>] [-w <workbook.xml>] [-M <mode>]
#
# Notes:
# - mode: "summary" (fast, default for in-run reports), "full" (heavier, parses XMI), or "csv" (use per-doc CSVs, no XMI parse)

OUT_DIR=""
PIPER=""
RUN_LOG=""
DICT_XML=""
WORKBOOK=""
MODE=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    -o|--out) OUT_DIR="$2"; shift 2;;
    -p|--piper) PIPER="$2"; shift 2;;
    -l|--log) RUN_LOG="$2"; shift 2;;
    -d|--dict) DICT_XML="$2"; shift 2;;
    -w|--workbook) WORKBOOK="$2"; shift 2;;
    -M|--mode) MODE="$2"; shift 2;;
    *) echo "Unknown arg: $1" >&2; exit 1;;
  esac
done

if [[ -z "$OUT_DIR" ]]; then echo "-o/--out is required" >&2; exit 2; fi
BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
CTAKES_HOME="${CTAKES_HOME:-$BASE_DIR/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0}"
JAVA_CP="$CTAKES_HOME/desc:$CTAKES_HOME/resources:$CTAKES_HOME/config:$CTAKES_HOME/config/*:$CTAKES_HOME/lib/*:$BASE_DIR/.build_tools"

if [[ -d "$OUT_DIR" ]]; then
  OUT_ABS="$(cd "$OUT_DIR" && pwd)"
elif [[ -d "$BASE_DIR/$OUT_DIR" ]]; then
  OUT_ABS="$(cd "$BASE_DIR/$OUT_DIR" && pwd)"
else
  echo "Output dir not found: $OUT_DIR" >&2; exit 2
fi
if [[ -z "$WORKBOOK" ]]; then
  BASE_NAME="$(basename "$OUT_ABS")"
  PIPE_NAME=""
  DICT_NAME=""
  if [[ -n "$PIPER" && -f "$PIPER" ]]; then
    PIPE_NAME="$(basename "$PIPER" .piper)"
  fi
  if [[ -n "$DICT_XML" && -f "$DICT_XML" ]]; then
    DICT_NAME="$(basename "$DICT_XML" .xml)"
  fi
  TS="$(date +%Y%m%d-%H%M%S)"
  # Compose: ctakes-report-<out>-<pipeline>-<dict>-<timestamp>.xlsx (omit missing parts)
  NAME_PARTS=("ctakes-report" "$BASE_NAME")
  [[ -n "$PIPE_NAME" ]] && NAME_PARTS+=("$PIPE_NAME")
  [[ -n "$DICT_NAME" ]] && NAME_PARTS+=("$DICT_NAME")
  NAME_PARTS+=("$TS")
  WORKBOOK="$OUT_ABS/$(IFS='-'; echo "${NAME_PARTS[*]}").xlsx"
fi

# Ensure WORKBOOK is absolute so path isn't affected by pushd into CTAKES_HOME
case "$WORKBOOK" in
  /*) : ;;
  *) WORKBOOK="$(cd "$BASE_DIR" && pwd)/$WORKBOOK" ;;
esac

# Compile reporting tools only (skip unrelated tools and Jupyter checkpoints)
find "$BASE_DIR/tools/reporting" -type f -name "*.java" ! -path "*/.ipynb_checkpoints/*" -print0 | \
  xargs -0 javac -cp "$JAVA_CP" -d "$BASE_DIR/.build_tools"

args=( tools.reporting.ExcelXmlReport -o "$OUT_ABS" -w "$WORKBOOK" )
[[ -n "$PIPER" ]] && args+=( -p "$PIPER" )
[[ -n "$RUN_LOG" ]] && args+=( -l "$RUN_LOG" )
[[ -n "$DICT_XML" ]] && args+=( -d "$DICT_XML" )
[[ -n "$MODE" ]] && args+=( -M "$MODE" )

pushd "$CTAKES_HOME" >/dev/null
# Allow caller to control report JVM heap via REPORT_XMX_MB (e.g., export REPORT_XMX_MB=8192)
JAVA_HEAP_OPTS=""
if [[ -n "${REPORT_XMX_MB:-}" ]]; then
  JAVA_HEAP_OPTS="-Xmx${REPORT_XMX_MB}m"
fi
java $JAVA_HEAP_OPTS -cp "$JAVA_CP" "${args[@]}"
popd >/dev/null

echo "Excel workbook: $WORKBOOK"
