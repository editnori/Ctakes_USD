#!/usr/bin/env bash
set -euo pipefail

# One-shot setup using the published cTAKES bundle.
# - Installs deps (Linux/macOS/WSL) and downloads the bundle
# - Leaves cTAKES under apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0
# - Prints the export line for CTAKES_HOME

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$BASE_DIR"

REL_URL=${REL_URL:-"https://github.com/editnori/Ctakes_USD/releases/download/bundle/CtakesBun-bundle.tgz"}
REL_SHA256=${REL_SHA256:-"0aae08a684ee5332aac0136e057cac0ee4fc29b34f2d5e3c3e763dc12f59e825"}

echo "[setup] Installing bundle and dependencies (sudo may prompt on Linux)"
bash scripts/install_bundle.sh --deps -u "$REL_URL" -s "$REL_SHA256"

CTAKES_HOME_GUESS="$BASE_DIR/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0"
echo
echo "[setup] Done. Set CTAKES_HOME in this shell:"
echo "  export CTAKES_HOME=\"$CTAKES_HOME_GUESS\""
echo
echo "Windows note: run scripts in Git Bash or WSL."

