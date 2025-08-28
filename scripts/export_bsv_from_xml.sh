#!/usr/bin/env bash
set -euo pipefail

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
CTAKES_HOME="${CTAKES_HOME:-$BASE_DIR/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0}"

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <dictionary.xml> [out_bsv]" >&2
  exit 1
fi

DICT_XML="$1"; shift || true
OUT_BSV="${1:-$BASE_DIR/dictionaries/FullClinical_AllTUIs/terms.bsv}"

WRAP_OUT="$BASE_DIR/.build_tools"
mkdir -p "$WRAP_OUT"

# Compile exporter if needed
if [[ ! -f "$WRAP_OUT/tools/ExportBsvFromHsql.class" ]]; then
  find "$BASE_DIR/tools" -name "*.java" -print0 | xargs -0 javac -cp "$CTAKES_HOME/desc:$CTAKES_HOME/resources:$CTAKES_HOME/config:$CTAKES_HOME/config/*:$CTAKES_HOME/lib/*" -d "$WRAP_OUT"
fi

JAVA_CP="$CTAKES_HOME/desc:$CTAKES_HOME/resources:$CTAKES_HOME/config:$CTAKES_HOME/config/*:$CTAKES_HOME/lib/*:$WRAP_OUT"

echo "Exporting BSV from: $DICT_XML" >&2
echo "Output BSV:        $OUT_BSV" >&2

java -Xms1g -Xmx2g -cp "$JAVA_CP" tools.ExportBsvFromHsql -l "$DICT_XML" -o "$OUT_BSV"
echo "Done. BSV at $OUT_BSV" >&2

