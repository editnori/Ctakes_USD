#!/usr/bin/env bash
set -euo pipefail

# Flight checks for cTAKES run scripts.
# Verifies jars, dictionary DB, temporal model, piper files, and dry-run sanitized XML.
#
# Usage:
#   scripts/flight_check.sh [--mode cluster|smoke] [--require-shared]
#                           [--only "S_core ..."]
# Env used:
#   CTAKES_HOME, DICT_SHARED (default 1), DICT_SHARED_PATH (default /dev/shm)

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
CTAKES_HOME="${CTAKES_HOME:-$BASE_DIR/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0}"
MODE="cluster"
REQUIRE_SHARED=0
ONLY_KEYS=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --mode) MODE="$2"; shift 2;;
    --require-shared) REQUIRE_SHARED=1; shift 1;;
    --only) ONLY_KEYS="$2"; shift 2;;
    *) echo "Unknown arg: $1" >&2; exit 2;;
  esac
done

DICT_SHARED="${DICT_SHARED:-1}"
DICT_SHARED_PATH="${DICT_SHARED_PATH:-/dev/shm}"

PROPS="$BASE_DIR/docs/builder_full_clinical.properties"
DICT_NAME=$(awk -F= '/^\s*dictionary.name\s*=/{print $2}' "$PROPS" 2>/dev/null | tr -d '\r' | xargs || true)
[[ -n "$DICT_NAME" ]] || DICT_NAME="FullClinical_AllTUIs"
DICT_XML="$CTAKES_HOME/resources/org/apache/ctakes/dictionary/lookup/fast/${DICT_NAME}.xml"
SRC_DB_DIR="$CTAKES_HOME/resources/org/apache/ctakes/dictionary/lookup/fast/$DICT_NAME"

pass() { echo "[OK]  $*"; }
warn() { echo "[WARN] $*"; }
fail() { echo "[FAIL] $*"; exit 1; }

echo "== Flight Checks (mode=$MODE, dict=$DICT_NAME) =="

# 1) HSQLDB jar present
if ls "$CTAKES_HOME/lib"/hsqldb-*.jar >/dev/null 2>&1; then
  pass "HSQLDB jar present under CTAKES_HOME/lib"
else
  fail "HSQLDB jar NOT found under CTAKES_HOME/lib (expected hsqldb-*.jar)"
fi

# 2) Dictionary XML + DB files present
[[ -f "$DICT_XML" ]] || fail "Dictionary XML not found: $DICT_XML"
pass "Dictionary XML found: $DICT_XML"

SCRIPT_FILE="$SRC_DB_DIR/$DICT_NAME.script"
PROPS_FILE="$SRC_DB_DIR/$DICT_NAME.properties"
[[ -f "$PROPS_FILE" ]] || fail "Dictionary properties missing: $PROPS_FILE"
[[ -s "$SCRIPT_FILE" ]] || fail "Dictionary script missing/empty: $SCRIPT_FILE"
size_b=$(wc -c < "$SCRIPT_FILE" 2>/dev/null || echo 0)
if (( size_b < 1000000 )); then
  warn "Dictionary script is smaller than 1MB (size=${size_b}B); ensure correct build"
else
  pass "Dictionary DB present (~$(numfmt --to=iec 2>/dev/null <<<"$size_b" || echo "$size_b") script)"
fi

# 3) Shared dict cache readiness (if enabled)
SHARED_PREFIX="${DICT_SHARED_PATH%/}/${DICT_NAME}_shared"
if [[ "$DICT_SHARED" -eq 1 ]]; then
  if [[ -f "${SHARED_PREFIX}.script" && -f "${SHARED_PREFIX}.properties" ]]; then
    if [[ -s "${SHARED_PREFIX}.script" ]]; then
      pass "Shared dict cache exists: ${SHARED_PREFIX}.(script|properties)"
    else
      fail "Shared dict cache script exists but empty: ${SHARED_PREFIX}.script"
    fi
  else
    if [[ "$REQUIRE_SHARED" -eq 1 ]]; then
      fail "Shared dict cache missing at ${SHARED_PREFIX}.(script|properties); run prepare_shared_dict.sh"
    else
      warn "Shared dict cache not found at ${SHARED_PREFIX}.(script|properties) (will copy per runner)"
    fi
  fi
  # Free space advisory
  need_b="$size_b"
  avail_b=$(df -PB1 "$DICT_SHARED_PATH" 2>/dev/null | awk 'NR==2{print $4}' || echo 0)
  if (( avail_b < need_b )); then
    warn "Low free space in $DICT_SHARED_PATH (avail=$(numfmt --to=iec <<<"$avail_b" 2>/dev/null || echo $avail_b) < need=$(numfmt --to=iec <<<"$need_b" 2>/dev/null || echo $need_b))"
  else
    pass "Sufficient free space in $DICT_SHARED_PATH for shared dict"
  fi
fi

# 4) Temporal model availability
HAS_TEMP_MODELS=0
if [[ -f "$CTAKES_HOME/resources/org/apache/ctakes/temporal/models/eventannotator/model.jar" ]]; then
  HAS_TEMP_MODELS=1
else
  for J in "$CTAKES_HOME"/lib/*.jar; do
    if jar tf "$J" 2>/dev/null | grep -q "org/apache/ctakes/temporal/models/eventannotator/model.jar"; then
      HAS_TEMP_MODELS=1; break
    fi
  done
fi
if [[ "$HAS_TEMP_MODELS" -eq 1 ]]; then
  pass "Temporal model available (models/eventannotator/model.jar)"
else
  warn "Temporal model NOT found; temporal pipelines will be skipped"
fi

# 5) Piper files exist and contain no embedded JDBC settings
declare -A SETS=(
  [S_core]="$BASE_DIR/pipelines/compare/TsSectionedFast_WSD_Compare.piper"
  [S_core_rel]="$BASE_DIR/pipelines/compare/TsSectionedRelation_WSD_Compare.piper"
  [S_core_temp]="$BASE_DIR/pipelines/compare/TsSectionedTemporal_WSD_Compare.piper"
  [S_core_temp_coref]="$BASE_DIR/pipelines/compare/TsSectionedTemporalCoref_WSD_Compare.piper"
  [S_core_temp_coref_smoke]="$BASE_DIR/pipelines/compare/TsSectionedTemporalCoref_WSD_Smoking_Compare.piper"
  [D_core_rel]="$BASE_DIR/pipelines/compare/TsDefaultRelation_WSD_Compare.piper"
  [D_core_temp]="$BASE_DIR/pipelines/compare/TsDefaultTemporal_WSD_Compare.piper"
  [D_core_temp_coref]="$BASE_DIR/pipelines/compare/TsDefaultTemporalCoref_WSD_Compare.piper"
  [D_core_temp_coref_smoke]="$BASE_DIR/pipelines/compare/TsDefaultTemporalCoref_WSD_Smoking_Compare.piper"
  [D_core_coref]="$BASE_DIR/pipelines/compare/TsDefaultCoref_WSD_Compare.piper"
)
keys=()
if [[ -n "$ONLY_KEYS" ]]; then
  # shellcheck disable=SC2206
  keys=($ONLY_KEYS)
else
  keys=("${!SETS[@]}")
fi
for k in "${keys[@]}"; do
  p="${SETS[$k]:-}"
  [[ -n "$p" && -f "$p" ]] || fail "Missing pipeline file for $k: $p"
  if command -v rg >/dev/null 2>&1; then
    if rg -n "jdbc(Url|Driver)" -S "$p" >/dev/null 2>&1; then
      fail "Pipeline $k embeds JDBC settings (should be only in sanitized XML): $(basename "$p")"
    fi
  else
    if grep -qE 'jdbc(Url|Driver)' "$p"; then
      fail "Pipeline $k embeds JDBC settings (should be only in sanitized XML): $(basename "$p")"
    fi
  fi
done
pass "Pipeline descriptors found and clean (no JDBC settings)"

CTAKES_SANITIZE_DICT="${CTAKES_SANITIZE_DICT:-0}"
if [[ "$CTAKES_SANITIZE_DICT" -eq 1 ]]; then
  # 6) Dry-run sanitize and verify driver + jdbcUrl flags
  tmpdir="$(mktemp -d)"; trap 'rm -rf "$tmpdir"' EXIT
  san="$tmpdir/${DICT_NAME}_local.xml"
  cp -f "$DICT_XML" "$san"
  # Replace impl + driver + strip umls props (same as runners)
  sed -i -E \
    -e 's#<implementationName>org.apache.ctakes.dictionary.lookup2.dictionary.UmlsJdbcRareWordDictionary</implementationName>#<implementationName>org.apache.ctakes.dictionary.lookup2.dictionary.JdbcRareWordDictionary</implementationName>#' \
    -e 's#<implementationName>org.apache.ctakes.dictionary.lookup2.concept.UmlsJdbcConceptFactory</implementationName>#<implementationName>org.apache.ctakes.dictionary.lookup2.concept.JdbcConceptFactory</implementationName>#' \
    -e 's#(key=\"jdbcDriver\" value)=\"[^\"]*\"#\1=\"org.hsqldb.jdbc.JDBCDriver\"#' \
    -e '/<property key=\"umlsUrl\"/d' -e '/<property key=\"umlsVendor\"/d' -e '/<property key=\"umlsUser\"/d' -e '/<property key=\"umlsPass\"/d' \
    "$san"

  if [[ "$MODE" == "cluster" ]]; then
    if [[ "$DICT_SHARED" -eq 1 ]]; then
      workdb="${DICT_SHARED_PATH%/}/${DICT_NAME}_shared"
    else
      workdb="/dev/shm/${DICT_NAME}_S_core_000"
    fi
    # Do NOT append flags; cTAKES 6.0.0 pre-validates <path>.script and flags break it
    sed -i -E "s#(key=\"jdbcUrl\" value)=\"[^\"]+\"#\1=\"jdbc:hsqldb:file:${workdb}\"#" "$san"
  else
    tmp_db="/tmp/ctakes_full/$DICT_NAME"
    sed -i -E "s#(key=\"jdbcUrl\" value)=\"[^\"]+\"#\1=\"jdbc:hsqldb:file:${tmp_db}\"#" "$san"
  fi

  if command -v rg >/dev/null 2>&1; then
    has_driver=$(rg -n "org.hsqldb.jdbc.JDBCDriver" -S "$san" >/dev/null && echo 1 || echo 0)
    has_url=$(rg -n "jdbc:hsqldb:file:" -S "$san" >/dev/null && echo 1 || echo 0)
  else
    has_driver=$(grep -qE 'org\.hsqldb\.jdbc\.JDBCDriver' "$san" && echo 1 || echo 0)
    has_url=$(grep -q 'jdbc:hsqldb:file:' "$san" >/dev/null && echo 1 || echo 0)
  fi
  if [[ "$has_driver" -eq 1 && "$has_url" -eq 1 ]]; then
    pass "Sanitized XML uses JDBCDriver + jdbcUrl (dry-run)"
  else
    fail "Sanitized XML missing expected driver or jdbcUrl (dry-run)"
  fi
else
  pass "Skipping sanitize dry-run (CTAKES_SANITIZE_DICT=0): using provided dictionary XML as-is"
fi

echo "== Flight checks complete."
