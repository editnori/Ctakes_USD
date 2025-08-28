#!/usr/bin/env bash
set -euo pipefail

# Install the exact cTAKES bundle for this repo.
#
# Looks for a local CtakesBun-bundle.tgz at repo root first, otherwise
# downloads from a release URL. Extracts into the repo so that
# CTAKES_HOME defaults work without extra config.
#
# Usage:
#   scripts/install_bundle.sh [-f <bundle.tgz>] [-u <url>] [-s <sha256>]

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BUNDLE_FILE="${BASE_DIR}/CtakesBun-bundle.tgz"
BUNDLE_URL="${BUNDLE_URL:-https://github.com/editnori/Ctakes_USD/releases/download/bundle/CtakesBun-bundle.tgz}"
EXPECT_SHA=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    -f|--file) BUNDLE_FILE="$2"; shift 2;;
    -u|--url) BUNDLE_URL="$2"; shift 2;;
    -s|--sha256) EXPECT_SHA="$2"; shift 2;;
    *) echo "Unknown arg: $1" >&2; exit 2;;
  esac
done

if [[ ! -f "$BUNDLE_FILE" ]]; then
  echo "Bundle not found locally at $BUNDLE_FILE"
  echo "Downloading from: $BUNDLE_URL"
  curl -L "$BUNDLE_URL" -o "$BUNDLE_FILE"
fi

if [[ -n "$EXPECT_SHA" ]]; then
  echo "$EXPECT_SHA  $BUNDLE_FILE" | sha256sum -c -
fi

echo "Extracting bundle: $BUNDLE_FILE"
tar -xzf "$BUNDLE_FILE" -C "$BASE_DIR"

echo "Bundle extracted. CTAKES_HOME default should now resolve to:"
echo "  $BASE_DIR/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0"
echo "Run: scripts/run_compare_cluster.sh -i <in> -o <out> --reports"

