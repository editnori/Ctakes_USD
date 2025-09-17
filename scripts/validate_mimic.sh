#!/usr/bin/env bash
set -euo pipefail

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
DEFAULT_INPUT="${BASE_DIR}/samples/mimic"
DEFAULT_OUTPUT="${BASE_DIR}/outputs/validate_mimic"
DEFAULT_MANIFEST_BASE="${BASE_DIR}/samples/mimic_manifest"
DEFAULT_MANIFEST="${DEFAULT_MANIFEST_BASE}.txt"

ENV_FILE="${BASE_DIR}/.ctakes_env"
if [[ -f "${ENV_FILE}" ]]; then
  # shellcheck disable=SC1090
  source "${ENV_FILE}"
fi


VALIDATE_SCRIPT="${BASE_DIR}/scripts/validate.sh"
if [[ ! -f "${VALIDATE_SCRIPT}" ]]; then
  echo "[validate_mimic] Missing validate.sh helper" >&2
  exit 1
fi

VALIDATE_CMD=("${BASH:-bash}" "${VALIDATE_SCRIPT}")

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
PIPELINE_KEY="core_sectioned_smoke"
PIPELINE_SET=0
PIPELINE_RUNS=()
WITH_TEMPORAL=0
WITH_COREF=0
DRY_RUN=0
MANIFEST="${DEFAULT_MANIFEST}"
MANIFEST_PROVIDED=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    -i|--input) IN_DIR="$2"; shift 2;;
    -o|--output) OUT_DIR="$2"; shift 2;;
    --limit) LIMIT="$2"; shift 2;;
    --pipeline) PIPELINE_KEY="$2"; PIPELINE_SET=1; PIPELINE_RUNS=("${PIPELINE_KEY}"); shift 2;;
    --with-temporal) WITH_TEMPORAL=1; shift 1;;
    --with-coref) WITH_COREF=1; shift 1;;
    --manifest) MANIFEST="$2"; MANIFEST_PROVIDED=1; shift 2;;
    --dry-run) DRY_RUN=1; shift 1;;
    -h|--help) usage; exit 0;;
    *) echo "Unknown option: $1" >&2; usage >&2; exit 1;;
  esac

done

if [[ ${PIPELINE_SET} -eq 0 && -t 0 && -t 1 ]]; then
  echo "Select validation option:"
  echo "  1) Core + Sectioned + Smoke (combined default)"
  echo "  2) Core + Sectioned"
  echo "  3) Core only"
  echo "  4) Drug only"
  read -r -p "Selection [1-4]: " __choice
  case "${__choice}" in
    ""|1) PIPELINE_RUNS=(core_sectioned_smoke);;
    2) PIPELINE_RUNS=(core sectioned);;
    3) PIPELINE_RUNS=(core);;
    4) PIPELINE_RUNS=(drug);;
    *) echo "[validate_mimic] Unknown selection '${__choice}'; defaulting to Core + Sectioned + Smoke combo."; PIPELINE_RUNS=(core_sectioned_smoke);;
  esac
fi

if [[ ${PIPELINE_SET} -eq 1 ]]; then
  PIPELINE_RUNS=("${PIPELINE_KEY}")
elif [[ ${#PIPELINE_RUNS[@]} -eq 0 ]]; then
  PIPELINE_RUNS=(core_sectioned_smoke)
fi

if [[ ! -d "${IN_DIR}" ]]; then
  echo "[validate_mimic] Input directory not found: ${IN_DIR}" >&2
  echo "Copy sample notes into ${IN_DIR} or pass --input." >&2
  exit 1
fi

mkdir -p "${OUT_DIR}"

if [[ ${#PIPELINE_RUNS[@]} -eq 0 ]]; then
  PIPELINE_RUNS=("${PIPELINE_KEY}")
fi

STATUS=0
for pipeline in "${PIPELINE_RUNS[@]}"; do
  OUT_DIR_PIPE="${OUT_DIR%/}/${pipeline}"
  mkdir -p "${OUT_DIR_PIPE}"
  MANIFEST_USE="${MANIFEST}"
  if [[ ${MANIFEST_PROVIDED} -eq 0 ]]; then
    MANIFEST_USE="${DEFAULT_MANIFEST_BASE}_${pipeline}.txt"
  fi
  CMD=("${VALIDATE_CMD[@]}" -i "${IN_DIR}" -o "${OUT_DIR_PIPE}" --pipeline "${pipeline}" --limit "${LIMIT}" --manifest "${MANIFEST_USE}")
  [[ ${WITH_TEMPORAL} -eq 1 ]] && CMD+=(--with-temporal)
  [[ ${WITH_COREF} -eq 1 ]] && CMD+=(--with-coref)
  [[ ${DRY_RUN} -eq 1 ]] && CMD+=(--dry-run)
  echo "[validate_mimic] Running ${pipeline} pipeline -> ${OUT_DIR_PIPE}"
  if ! "${CMD[0]}" "${CMD[@]:1}"; then
    STATUS=1
  fi
done

exit ${STATUS}

