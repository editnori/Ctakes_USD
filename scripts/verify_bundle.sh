#!/usr/bin/env bash
set -euo pipefail

# Verify the prebuilt fast dictionary is present either in the working tree
# (apache-ctakes-6.0.0-bin) or inside a bundle tarball.
#
# Usage:
#   scripts/verify_bundle.sh            # checks working tree
#   scripts/verify_bundle.sh -f bundle.tgz  # checks inside tarball

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
CHECK_TAR=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    -f|--file) CHECK_TAR="$2"; shift 2;;
    *) echo "Unknown arg: $1" >&2; exit 2;;
  esac
done

DICT_DIR="apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0/resources/org/apache/ctakes/dictionary/lookup/fast/FullClinical_AllTUIs"
DICT_FILE="${DICT_DIR}/FullClinical_AllTUIs.script"

if [[ -n "$CHECK_TAR" ]]; then
  [[ -f "$CHECK_TAR" ]] || { echo "No such tar: $CHECK_TAR" >&2; exit 2; }
  echo "Checking dictionary inside: $CHECK_TAR"
  if tar -tzf "$CHECK_TAR" "$DICT_FILE" >/dev/null 2>&1; then
    echo "OK: Found $DICT_FILE in bundle."
  else
    echo "ERROR: Missing $DICT_FILE in bundle." >&2
    exit 1
  fi
  exit 0
fi

echo "Checking working tree under: $BASE_DIR"
if [[ -f "$BASE_DIR/$DICT_FILE" ]]; then
  SIZE=$(du -h "$BASE_DIR/$DICT_FILE" | awk '{print $1}')
  echo "OK: Found $DICT_FILE ($SIZE)"
else
  echo "ERROR: Missing $DICT_FILE in working tree." >&2
  exit 1
fi

