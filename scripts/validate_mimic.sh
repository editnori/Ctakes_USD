#!/usr/bin/env bash
set -euo pipefail

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
DEFAULT_INPUT="${BASE_DIR}/samples/mimic"
DEFAULT_OUTPUT="${BASE_DIR}/outputs/validate_mimic"
DEFAULT_MANIFEST="${BASE_DIR}/samples/mimic_manifest.txt"

usage() {
  cat <<'USAGE'
Usage: scripts/validate_mimic.sh [options]
Options:
  -i, --input <dir>    Source notes directory (default: samples/mimic)
  -o, --output <dir>   Output directory (default: outputs/validate_mimic)
  --limit <N>          Override sample size (default: 100)
  --pipeline <key>     Pipeline key passed to validate.sh (default: smoke)
  --with-temporal      Add temporal module
  --with-coref         Add coref module
  --manifest <file>    Override manifest path (default: samples/mimic_manifest.txt)
  --dry-run            Print the commands without executing
  -h, --help           Show this help text

Runs scripts/validate.sh with defaults suited for the shipped MIMIC sample (100 notes).
USAGE
}

IN_DIR="${DEFAULT_INPUT}"
OUT_DIR="${DEFAULT_OUTPUT}"
LIMIT=100
PIPELINE_KEY="smoke"
WITH_TEMPORAL=0
WITH_COREF=0
DRY_RUN=0
MANIFEST="${DEFAULT_MANIFEST}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    -i|--input) IN_DIR="$2"; shift 2;;
    -o|--output) OUT_DIR="$2"; shift 2;;
    --limit) LIMIT="$2"; shift 2;;
    --pipeline) PIPELINE_KEY="$2"; shift 2;;
    --with-temporal) WITH_TEMPORAL=1; shift 1;;
    --with-coref) WITH_COREF=1; shift 1;;
    --manifest) MANIFEST="$2"; shift 2;;
    --dry-run) DRY_RUN=1; shift 1;;
    -h|--help) usage; exit 0;;
    *) echo "Unknown option: $1" >&2; usage >&2; exit 1;;
  esac

done

if [[ ! -d "${IN_DIR}" ]]; then
  echo "[validate_mimic] Input directory not found: ${IN_DIR}" >&2
  echo "Copy sample notes into ${IN_DIR} or pass --input." >&2
  exit 1
fi

mkdir -p "${OUT_DIR}"

CMD=("${BASE_DIR}/scripts/validate.sh" -i "${IN_DIR}" -o "${OUT_DIR}" --pipeline "${PIPELINE_KEY}" --limit "${LIMIT}" --manifest "${MANIFEST}")
[[ ${WITH_TEMPORAL} -eq 1 ]] && CMD+=(--with-temporal)
[[ ${WITH_COREF} -eq 1 ]] && CMD+=(--with-coref)
[[ ${DRY_RUN} -eq 1 ]] && CMD+=(--dry-run)

if [[ ${DRY_RUN} -eq 1 ]]; then
  printf '[validate_mimic] %q ' "${CMD[@]}"
  printf '\n'
  exit 0
fi

exec "${CMD[0]}" "${CMD[@]:1}"
