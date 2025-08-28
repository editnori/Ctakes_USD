#!/usr/bin/env bash
set -euo pipefail

# Create a self-contained bundle of your current cTAKES instance and
# customizations so others can install it via scripts/install_bundle.sh.
#
# This packs the apache-ctakes-6.0.0-bin/ tree relative to repo root and
# writes CtakesBun-bundle.tgz alongside.
#
# Usage:
#   scripts/make_bundle.sh

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
SRC_DIR="$BASE_DIR/apache-ctakes-6.0.0-bin"
OUT_TGZ="$BASE_DIR/CtakesBun-bundle.tgz"

[[ -d "$SRC_DIR" ]] || { echo "Missing $SRC_DIR. Put your exact cTAKES install there." >&2; exit 2; }

TMP_META="$BASE_DIR/.bundle_meta"
mkdir -p "$TMP_META"
CTAKES_VER_FILE="$TMP_META/VERSION.txt"
{
  echo "Bundle built: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "cTAKES tree: $SRC_DIR"
  if [[ -f "$SRC_DIR/apache-ctakes-6.0.0/NOTICE" ]]; then
    echo "NOTICE present."; fi
} > "$CTAKES_VER_FILE"

echo "Creating bundle: $OUT_TGZ"
tar -czf "$OUT_TGZ" -C "$BASE_DIR" \
  apache-ctakes-6.0.0-bin \
  .bundle_meta/VERSION.txt

echo "Bundle ready: $OUT_TGZ"
echo "SHA256: $(sha256sum "$OUT_TGZ" | awk '{print $1}')"

