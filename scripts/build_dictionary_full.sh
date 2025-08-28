#!/usr/bin/env bash
set -euo pipefail

# Build the comprehensive FullClinical_AllTUIs dictionary (BSV + HSQL) with relative paths.
# Logs to dictionaries/FullClinical_AllTUIs/logs/build_<timestamp>.log

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
CTAKES_HOME="${CTAKES_HOME:-$BASE_DIR/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0}"
PROPS_REL="docs/builder_full_clinical.properties"
# FAST_SMOKE=1 to use a tiny smoke config and tiny UMLS files under umls_smoke/
if [[ "${FAST_SMOKE:-0}" == "1" ]]; then
  PROPS_REL="docs/builder_smoke.properties"
fi
PROPS="$BASE_DIR/$PROPS_REL"

if [[ ! -f "$PROPS" ]]; then
  echo "Properties file not found: $PROPS_REL" >&2
  exit 1
fi

# Extract umls.dir and output.dir from properties (simple parser)
UMLS_DIR=$(awk -F= '/^\s*umls.dir\s*=/{print $2}' "$PROPS" | tr -d '\r' | xargs)
OUT_DIR=$(awk -F= '/^\s*output.dir\s*=/{print $2}' "$PROPS" | tr -d '\r' | xargs)

# Resolve relative to repo
[[ "${UMLS_DIR:0:1}" != "/" ]] && UMLS_DIR="$BASE_DIR/$UMLS_DIR"
[[ "${OUT_DIR:0:1}" != "/" ]] && OUT_DIR="$BASE_DIR/$OUT_DIR"

mkdir -p "$OUT_DIR/logs"
LOG="$OUT_DIR/logs/build_$(date +%Y%m%d_%H%M%S).log"

export CTAKES_HOME
echo "CTAKES_HOME: $CTAKES_HOME"
echo "UMLS_DIR:    $UMLS_DIR"
echo "OUT_DIR:     $OUT_DIR"
echo "Logging to:  $LOG"

# Minimal RRF presence check
missing=0
for f in MRCONSO.RRF MRSTY.RRF MRSAB.RRF; do
  if [[ -f "$UMLS_DIR/$f" || -f "$UMLS_DIR/META/$f" ]]; then
    :
  else
    echo "Missing $f in $UMLS_DIR (or $UMLS_DIR/META)" >&2
    missing=1
  fi
done
if [[ "$missing" -ne 0 ]]; then
  echo "Required UMLS files missing. Please place RRF files under $UMLS_DIR" >&2
  exit 1
fi

# Ensure cTAKES GUI builder expected layout: RRFs under UMLS_DIR/META
if [[ ! -f "$UMLS_DIR/META/MRCONSO.RRF" ]]; then
  if [[ -f "$UMLS_DIR/MRCONSO.RRF" ]]; then
    echo "Normalizing UMLS layout: creating META view under $UMLS_DIR/META" | tee -a "$LOG"
    mkdir -p "$UMLS_DIR/META"
    for f in MRCONSO.RRF MRSTY.RRF MRSAB.RRF MRRANK.RRF MRXW_ENG.RRF MRREL.RRF MRHIER.RRF MRDEF.RRF MRMAP.RRF MRSMAP.RRF; do
      if [[ -f "$UMLS_DIR/$f" && ! -f "$UMLS_DIR/META/$f" ]]; then
        # Try hardlink; fall back to copy if hardlink fails (e.g., cross-device)
        ln "$UMLS_DIR/$f" "$UMLS_DIR/META/$f" 2>/dev/null || cp -f "$UMLS_DIR/$f" "$UMLS_DIR/META/$f"
      fi
    done
  fi
fi

# Compile and run headless builder wrapper
WRAP_SRC="$BASE_DIR/tools/HeadlessDictionaryBuilder.java"
WRAP_OUT="$BASE_DIR/.build_tools"
mkdir -p "$WRAP_OUT"

echo "Compiling headless builder wrapper..." | tee -a "$LOG"
set -x
# Compile all helper classes (wrapper + HSQL compatibility shim)
find "$BASE_DIR/tools" -name "*.java" -print0 | xargs -0 javac -cp "$CTAKES_HOME/desc:$CTAKES_HOME/resources:$CTAKES_HOME/config:$CTAKES_HOME/config/*:$CTAKES_HOME/lib/*" -d "$WRAP_OUT" |& tee -a "$LOG"

echo "Ensuring cTAKES fast dictionary resources path exists..." | tee -a "$LOG"
mkdir -p "$CTAKES_HOME/resources/org/apache/ctakes/dictionary/lookup/fast"

echo "Running headless discovery scan..." | tee -a "$LOG"
DISCOVER=$(java -Xms1g -Xmx2g \
  -cp "$CTAKES_HOME/desc:$CTAKES_HOME/resources:$CTAKES_HOME/config:$CTAKES_HOME/config/*:$CTAKES_HOME/lib/*:$WRAP_OUT" \
  tools.HeadlessDictionaryCreator -p "$PROPS" 2>&1 | tee -a "$LOG")

# Extract discovered sets
SABS=$(echo "$DISCOVER" | sed -n 's/^DISCOVERED_SABS=//p' | tail -n1)
LATS=$(echo "$DISCOVER" | sed -n 's/^DISCOVERED_LANGUAGES=//p' | tail -n1)
TUIS=$(echo "$DISCOVER" | sed -n 's/^DISCOVERED_TUIS=//p' | tail -n1)
echo "Using SABs: ${SABS:-<as-is>}" | tee -a "$LOG"
echo "Found LANGs: ${LATS:-<unknown>} (keeping ENG only)" | tee -a "$LOG"

# Build a merged properties file overriding vocabularies and TUIs with discovered sets
MERGED="$OUT_DIR/merged_builder.properties"; mkdir -p "$(dirname "$MERGED")"
cp -f "$PROPS" "$MERGED"
if [[ -n "${SABS:-}" ]]; then
  if grep -q '^\s*vocabularies\s*=' "$MERGED"; then
    sed -i "s#^\s*vocabularies\s*=.*#vocabularies=$SABS#" "$MERGED"
  else
    printf "\nvocabularies=%s\n" "$SABS" >> "$MERGED"
  fi
fi
if [[ -n "${TUIS:-}" ]]; then
  if grep -q '^\s*semantic\.types\s*=' "$MERGED"; then
    sed -i "s#^\s*semantic\.types\s*=.*#semantic.types=$TUIS#" "$MERGED"
  else
    printf "\nsemantic.types=%s\n" "$TUIS" >> "$MERGED"
  fi
fi
PROPS="$MERGED"

echo "Running headless dictionary build..." | tee -a "$LOG"
# Pre-clean existing DB dir to avoid HSQL create-table collisions on rebuild
DICT_NAME=$(awk -F= '/^\s*dictionary.name\s*=/{print $2}' "$PROPS" | tr -d '\r' | xargs)
DICT_XML_DIR="$CTAKES_HOME/resources/org/apache/ctakes/dictionary/lookup/fast"
DB_DIR="$DICT_XML_DIR/$DICT_NAME"
rm -rf "$DB_DIR"
mkdir -p "$DICT_XML_DIR"
CP="$CTAKES_HOME/desc:$CTAKES_HOME/resources:$CTAKES_HOME/config:$CTAKES_HOME/config/*:$CTAKES_HOME/lib/*:$WRAP_OUT"
java -Xms2g -Xmx6g \
  -Drepo.base="$BASE_DIR" \
  -cp "$CP" \
  tools.HeadlessDictionaryBuilder \
  -p "$PROPS" |& tee -a "$LOG"
set +x

# Summarize outputs
if [[ -f "$OUT_DIR/terms.bsv" ]]; then
  echo "BSV rows: $(wc -l < "$OUT_DIR/terms.bsv")" | tee -a "$LOG"
  echo "terms.bsv size: $(du -h "$OUT_DIR/terms.bsv" | cut -f1)" | tee -a "$LOG"
fi
if [[ -d "$OUT_DIR/hsqldb" ]]; then
  echo "HSQL files:" | tee -a "$LOG"
  ls -lh "$OUT_DIR/hsqldb" | tee -a "$LOG"
fi
# Try to locate resulting XML under cTAKES resources
DICT_NAME=$(awk -F= '/^\s*dictionary.name\s*=/{print $2}' "$PROPS" | tr -d '\r' | xargs)
DICT_XML_DIR="$CTAKES_HOME/resources/org/apache/ctakes/dictionary/lookup/fast"
DICT_XML="$DICT_XML_DIR/${DICT_NAME}.xml"
if [[ -f "$DICT_XML" ]]; then
  echo "dictionary xml: $DICT_XML" | tee -a "$LOG"
  # Create a local-credentials-free variant that does not require UMLS approval
  DICT_XML_LOCAL="$DICT_XML_DIR/${DICT_NAME}_local.xml"
  echo "writing local variant: $DICT_XML_LOCAL" | tee -a "$LOG"
  sed -e 's#<implementationName>org.apache.ctakes.dictionary.lookup2.dictionary.UmlsJdbcRareWordDictionary</implementationName>#<implementationName>org.apache.ctakes.dictionary.lookup2.dictionary.JdbcRareWordDictionary</implementationName>#' \
      -e 's#<property name="jdbcDriver" value="[^"]*"#<property name="jdbcDriver" value="org.hsqldb.jdbc.JDBCDriver"#' \
      -e '/<property name="umlsUrl"/d' \
      -e '/<property name="umlsVendor"/d' \
      -e '/<property name="umlsUser"/d' \
      -e '/<property name="umlsPass"/d' \
      "$DICT_XML" > "$DICT_XML_LOCAL"
  echo "dictionary local xml: $DICT_XML_LOCAL" | tee -a "$LOG"
else
  echo "dictionary xml not found at expected path: $DICT_XML" | tee -a "$LOG"
fi

echo "Done. Log: $LOG"
