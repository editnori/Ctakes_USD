#!/usr/bin/env bash
set -euo pipefail

# Convenience wrapper to preview only the main pipelines (Sectioned Core/Relation/Smoking)
# Usage: bash scripts/status_main.sh -i <input_dir> [-o <output_base>] [other status.sh args]

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"

exec bash "$BASE_DIR/scripts/status.sh" --only "S_core_rel_smoke" "$@"
