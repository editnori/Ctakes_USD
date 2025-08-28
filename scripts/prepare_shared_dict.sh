#!/usr/bin/env bash
set -euo pipefail

# Prepare a shared read-only HSQLDB dictionary copy for cTAKES runs.
# Usage: scripts/prepare_shared_dict.sh [-d <dict_name>] [-t <target_dir>]
# Defaults:
#   - dict_name from docs/builder_full_clinical.properties (dictionary.name)
#   - target_dir=/dev/shm

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
CTAKES_HOME="${CTAKES_HOME:-$BASE_DIR/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0}"

DICT_NAME=""; TARGET_DIR="/dev/shm"
while [[ $# -gt 0 ]]; do
  case "$1" in
    -d|--dict) DICT_NAME="$2"; shift 2;;
    -t|--target) TARGET_DIR="$2"; shift 2;;
    *) echo "Unknown arg: $1" >&2; exit 2;;
  esac
done

if [[ -z "$DICT_NAME" ]]; then
  PROPS="$BASE_DIR/docs/builder_full_clinical.properties"
  DICT_NAME=$(awk -F= '/^\s*dictionary.name\s*=/{print $2}' "$PROPS" | tr -d '\r' | xargs)
  [[ -n "$DICT_NAME" ]] || DICT_NAME="FullClinical_AllTUIs"
fi

SRC_DIR="$CTAKES_HOME/resources/org/apache/ctakes/dictionary/lookup/fast/$DICT_NAME"
[[ -f "$SRC_DIR/$DICT_NAME.script" ]] || { echo "Dictionary script not found: $SRC_DIR/$DICT_NAME.script" >&2; exit 1; }

mkdir -p "$TARGET_DIR"
TARGET_PREFIX="${TARGET_DIR%/}/${DICT_NAME}_shared"
cp -f "$SRC_DIR/$DICT_NAME.properties" "${TARGET_PREFIX}.properties"
cp -f "$SRC_DIR/$DICT_NAME.script" "${TARGET_PREFIX}.script"

echo "Prepared: ${TARGET_PREFIX}.(properties|script)"
echo "Export to reuse:"
echo "  export DICT_SHARED=1"
echo "  export DICT_SHARED_PATH=\"${TARGET_DIR%/}\""
echo "Runner will use: jdbc:hsqldb:file:${TARGET_PREFIX};readonly=true;hsqldb.lock_file=false"

