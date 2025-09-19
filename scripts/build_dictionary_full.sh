#!/usr/bin/env bash
set -euo pipefail

# Build the KidneyStone_SDOH dictionary (BSV + HSQL) from this repo.
# Logs land under dictionaries/<name>/logs/build_<timestamp>.log

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
CTAKES_HOME="${CTAKES_HOME:-$BASE_DIR/CtakesBun-bundle/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0}"
PROPS_REL="${PROPS_REL:-resources/dictionary_configs/kidney_sdoh.conf}"

PROPS="$BASE_DIR/$PROPS_REL"
if [[ ! -f "$PROPS" ]]; then
  echo "Properties file not found: $PROPS_REL" >&2
  exit 1
fi

# Extract umls.dir and output.dir from the properties
UMLS_DIR=$(awk -F= '/^\s*umls.dir\s*=/{print $2}' "$PROPS" | tr -d '\r' | xargs)
OUT_DIR=$(awk -F= '/^\s*output.dir\s*=/{print $2}' "$PROPS" | tr -d '\r' | xargs)

[[ "${UMLS_DIR:0:1}" != "/" ]] && UMLS_DIR="$BASE_DIR/$UMLS_DIR"
[[ "${OUT_DIR:0:1}" != "/" ]] && OUT_DIR="$BASE_DIR/$OUT_DIR"

mkdir -p "$OUT_DIR/logs"
LOG="$OUT_DIR/logs/build_$(date +%Y%m%d_%H%M%S).log"

export CTAKES_HOME
if [[ ! -d "$CTAKES_HOME/lib" ]]; then
  echo "Invalid CTAKES_HOME (lib missing): $CTAKES_HOME" >&2
  exit 1
fi

echo "CTAKES_HOME: $CTAKES_HOME"
echo "UMLS_DIR:    $UMLS_DIR"
echo "OUT_DIR:     $OUT_DIR"
echo "Logging to:  $LOG"

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
  echo "Required UMLS files missing. Update umls.dir in $PROPS_REL" >&2
  exit 1
fi

if [[ ! -f "$UMLS_DIR/META/MRCONSO.RRF" ]]; then
  if [[ -f "$UMLS_DIR/MRCONSO.RRF" ]]; then
    echo "Normalizing UMLS layout under $UMLS_DIR/META" | tee -a "$LOG"
    mkdir -p "$UMLS_DIR/META"
    for f in MRCONSO.RRF MRSTY.RRF MRSAB.RRF MRRANK.RRF MRXW_ENG.RRF MRREL.RRF MRHIER.RRF MRDEF.RRF MRMAP.RRF MRSMAP.RRF; do
      if [[ -f "$UMLS_DIR/$f" && ! -f "$UMLS_DIR/META/$f" ]]; then
        ln "$UMLS_DIR/$f" "$UMLS_DIR/META/$f" 2>/dev/null || cp -f "$UMLS_DIR/$f" "$UMLS_DIR/META/$f"
      fi
    done
  fi
fi

WRAP_OUT="$BASE_DIR/build/dictionary_tools"
mkdir -p "$WRAP_OUT"

echo "Compiling headless builder helpers..." | tee -a "$LOG"
set -x
javac -cp "$CTAKES_HOME/desc:$CTAKES_HOME/resources:$CTAKES_HOME/config:$CTAKES_HOME/config/*:$CTAKES_HOME/lib/*" \
  -d "$WRAP_OUT" \
  "$BASE_DIR/tools/HeadlessDictionaryCreator.java" \
  "$BASE_DIR/tools/HeadlessDictionaryBuilder.java" \
  "$BASE_DIR/tools/DictionaryRxnormAugmenter.java" |& tee -a "$LOG"

mkdir -p "$CTAKES_HOME/resources/org/apache/ctakes/dictionary/lookup/fast"

echo "Running discovery scan..." | tee -a "$LOG"
DISCOVER_PROPS="$OUT_DIR/_abs_builder.properties"
cp -f "$PROPS" "$DISCOVER_PROPS"
if grep -q '^\s*umls.dir\s*=' "$DISCOVER_PROPS"; then
  sed -i "s#^\s*umls.dir\s*=.*#umls.dir=$UMLS_DIR#" "$DISCOVER_PROPS"
else
  printf "\numls.dir=%s\n" "$UMLS_DIR" >> "$DISCOVER_PROPS"
fi
DISCOVER=$(java -Xms1g -Xmx2g \
  -cp "$CTAKES_HOME/desc:$CTAKES_HOME/resources:$CTAKES_HOME/config:$CTAKES_HOME/config/*:$CTAKES_HOME/lib/*:$WRAP_OUT" \
  tools.HeadlessDictionaryCreator -p "$DISCOVER_PROPS" 2>&1 | tee -a "$LOG")
rm -f "$DISCOVER_PROPS"

SABS=$(echo "$DISCOVER" | sed -n 's/^DISCOVERED_SABS=//p' | tail -n1)
TUIS=$(echo "$DISCOVER" | sed -n 's/^DISCOVERED_TUIS=//p' | tail -n1)

echo "Using SABs: ${SABS:-<configured>}" | tee -a "$LOG"

MERGED="$OUT_DIR/merged_builder.properties"
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

set +x

echo "Running dictionary build..." | tee -a "$LOG"
DICT_NAME=$(awk -F= '/^\s*dictionary.name\s*=/{print $2}' "$PROPS" | tr -d '\r' | xargs)
DICT_XML_DIR="$CTAKES_HOME/resources/org/apache/ctakes/dictionary/lookup/fast"
DB_DIR="$DICT_XML_DIR/$DICT_NAME"
rm -rf "$DB_DIR"
mkdir -p "$DICT_XML_DIR"
CP="$CTAKES_HOME/desc:$CTAKES_HOME/resources:$CTAKES_HOME/config:$CTAKES_HOME/config/*:$CTAKES_HOME/lib/*:$WRAP_OUT"
java -Xms2g -Xmx6g -Drepo.base="$BASE_DIR" -cp "$CP" tools.HeadlessDictionaryBuilder -p "$PROPS" | tee -a "$LOG"

if [[ -f "$OUT_DIR/terms.bsv" ]]; then
  echo "BSV rows: $(wc -l < "$OUT_DIR/terms.bsv")" | tee -a "$LOG"
  echo "terms.bsv size: $(du -h "$OUT_DIR/terms.bsv" | cut -f1)" | tee -a "$LOG"
fi
if [[ -d "$OUT_DIR/hsqldb" ]]; then
  echo "HSQL files:" | tee -a "$LOG"
  ls -lh "$OUT_DIR/hsqldb" | tee -a "$LOG"
fi

DICT_XML="$DICT_XML_DIR/${DICT_NAME}.xml"
if [[ -f "$DICT_XML" ]]; then
  echo "dictionary xml: $DICT_XML" | tee -a "$LOG"
  echo "Augmenting RxNorm codes..." | tee -a "$LOG"
  java -Xms1g -Xmx2g -cp "$CP" tools.DictionaryRxnormAugmenter -l "$DICT_XML" -u "$UMLS_DIR" | tee -a "$LOG"
  if ! grep -q 'rxnormTable' "$DICT_XML"; then
    perl -0pi -e 's/(<property key="prefTermTable" value="prefTerm"\s*\/>\n)/$1         <property key="rxnormTable" value="TEXT"\/>\n/' "$DICT_XML"
  fi
  DICT_XML_LOCAL="$DICT_XML_DIR/${DICT_NAME}_local.xml"
  echo "writing local variant: $DICT_XML_LOCAL" | tee -a "$LOG"
  sed -e 's#<implementationName>org.apache.ctakes.dictionary.lookup2.dictionary.UmlsJdbcRareWordDictionary</implementationName>#<implementationName>org.apache.ctakes.dictionary.lookup2.dictionary.JdbcRareWordDictionary</implementationName>#' \
      -e 's#<property key="jdbcDriver" value="[^"]*"#<property key="jdbcDriver" value="org.hsqldb.jdbcDriver"#' \
      -e '/<property key="umlsUrl"/d' \
      -e '/<property key="umlsVendor"/d' \
      -e '/<property key="umlsUser"/d' \
      -e '/<property key="umlsPass"/d' \
      "$DICT_XML" > "$DICT_XML_LOCAL"
  echo "dictionary local xml: $DICT_XML_LOCAL" | tee -a "$LOG"
else
  echo "dictionary xml not found at expected path: $DICT_XML" | tee -a "$LOG"
fi

echo "Done. Log: $LOG"
