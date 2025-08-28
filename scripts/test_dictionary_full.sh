#!/usr/bin/env bash
set -euo pipefail

# Test DefaultFastPipeline against an input directory using the built FullClinical_AllTUIs dictionary.
# Usage: scripts/test_dictionary_full.sh -i ./samples/input -o ./outputs/test_full

IN=""
OUT=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    -i|--in) IN="$2"; shift 2;;
    -o|--out) OUT="$2"; shift 2;;
    *) echo "Unknown arg: $1" >&2; exit 1;;
  esac
done

if [[ -z "$IN" || -z "$OUT" ]]; then
  echo "Usage: $0 -i <input_dir> -o <output_dir>" >&2
  exit 1
fi

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
CTAKES_HOME="${CTAKES_HOME:-$BASE_DIR/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0}"

# Make absolute input/output paths to survive cwd changes
IN_ABS="$(cd "$IN" && pwd)"
OUT_ABS="$(mkdir -p "$OUT" && cd "$OUT" && pwd)"
# Prefer dictionary xml under cTAKES resources produced by our headless builder
PROPS="$BASE_DIR/docs/builder_full_clinical.properties"
DICT_NAME=$(awk -F= '/^\s*dictionary.name\s*=/{print $2}' "$PROPS" | tr -d '\r' | xargs)
DICT_XML_RES="$CTAKES_HOME/resources/org/apache/ctakes/dictionary/lookup/fast/${DICT_NAME}.xml"
DICT_XML_LEG="$BASE_DIR/dictionaries/FullClinical_AllTUIs/dictionary.xml"
if [[ -f "$DICT_XML_RES" ]]; then
  DICT_XML="$DICT_XML_RES"
elif [[ -f "$DICT_XML_LEG" ]]; then
  DICT_XML="$DICT_XML_LEG"
else
  DICT_XML="$DICT_XML_RES"
fi

mkdir -p "$OUT_ABS/logs"
LOG="$OUT_ABS/logs/run_$(date +%Y%m%d_%H%M%S).log"

if [[ ! -f "$DICT_XML" ]]; then
  echo "Dictionary not found: $DICT_XML" >&2
  echo "Run scripts/build_dictionary_full.sh first." >&2
  exit 1
fi

echo "CTAKES_HOME: $CTAKES_HOME"
echo "IN:          $IN_ABS"
echo "OUT:         $OUT_ABS"
echo "DICT_XML:    $DICT_XML"
echo "Log:         $LOG"

# Default UMLS key (user provided). Override by exporting UMLS_KEY.
UMLS_KEY="${UMLS_KEY:-6370dcdd-d438-47ab-8749-5a8fb9d013f2}"

set -x
JAVA_CP="$CTAKES_HOME/desc:$CTAKES_HOME/resources:$CTAKES_HOME/config:$CTAKES_HOME/config/*:$CTAKES_HOME/lib/*:$BASE_DIR/.build_tools"
# If using a file path for dict xml, write a local variant that doesn't require UMLS auth
XML_ARG="$DICT_XML"
if [[ -f "$DICT_XML" ]]; then
  tmpxml="$OUT_ABS/$(basename "$DICT_XML" .xml)_local.xml"
  sed -e 's#<implementationName>org.apache.ctakes.dictionary.lookup2.dictionary.UmlsJdbcRareWordDictionary</implementationName>#<implementationName>org.apache.ctakes.dictionary.lookup2.dictionary.JdbcRareWordDictionary</implementationName>#' \
      -e 's#<property name=\"jdbcDriver\" value=\"[^\"]*\"#<property name=\"jdbcDriver\" value=\"org.hsqldb.jdbc.JDBCDriver\"#' \
      -e '/<property name=\"umlsUrl\"/d' \
      -e '/<property name=\"umlsVendor\"/d' \
      -e '/<property name=\"umlsUser\"/d' \
      -e '/<property name=\"umlsPass\"/d' \
      "$DICT_XML" > "$tmpxml"
  # Relocate DB to a space-free path to avoid HSQL URL encoding issues
  src_db="$CTAKES_HOME/resources/org/apache/ctakes/dictionary/lookup/fast/FullClinical_AllTUIs/FullClinical_AllTUIs"
  # Prefer tmp to avoid any parent dirs with spaces
  if [[ -d "/tmp" ]]; then
    dst_db="/tmp/ctakes_full/FullClinical_AllTUIs"
  else
    dst_db="$BASE_DIR/dictionaries/FullClinical_AllTUIs/build_local/FullClinical_AllTUIs"
  fi
  mkdir -p "$(dirname "$dst_db")"
  if [[ ! -f "$dst_db.script" ]]; then
    echo "Copying dictionary DB to: $dst_db.*" >&2
    cp -f "$src_db.properties" "$dst_db.properties"
    cp -f "$src_db.script" "$dst_db.script"
  fi
  # Rewrite jdbcUrl to absolute, space-free location
  dst_url="jdbc:hsqldb:file:${dst_db}"
  sed -i -E "s#(key=\"jdbcUrl\" value=)\"[^\"]+\"#\\1\"${dst_url}\"#" "$tmpxml"
  XML_ARG="$tmpxml"
fi

# Run from CTAKES_HOME so the HSQL jdbcUrl relative path resolves correctly
pushd "$CTAKES_HOME" >/dev/null
java -Xms2g -Xmx6g ${UMLS_KEY:+-Dctakes.umls_apikey=$UMLS_KEY} \
  -cp "$JAVA_CP" \
org.apache.ctakes.core.pipeline.PiperFileRunner \
  -p "$BASE_DIR/pipelines/smoke/DefaultFastWithXmi.piper" \
  -i "$IN_ABS" -o "$OUT_ABS" -l "$XML_ARG" ${UMLS_KEY:+--key $UMLS_KEY} |& tee "$LOG"
popd >/dev/null
set +x

echo "Done. Log: $LOG"
