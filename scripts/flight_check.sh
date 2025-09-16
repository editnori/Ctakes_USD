#!/usr/bin/env bash
set -euo pipefail

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
ISSUES=0
WARNINGS=0

note_ok()   { echo "[ok] $1"; }
note_warn() { echo "[warn] $1"; (( WARNINGS++ )) || true; }
note_fail() { echo "[fail] $1"; (( ISSUES++ )); }

# Java -----------------------------------------------------------------------
if command -v java >/dev/null 2>&1; then
  JAVA_VERSION_RAW=$(java -version 2>&1 | head -n1)
  JAVA_MAJOR=$(java -version 2>&1 | awk -F'[ "]+' 'NR==1 {print $3}' | awk -F'.' '{print $1}')
  if [[ -n "${JAVA_MAJOR}" && ${JAVA_MAJOR} -ge 11 ]]; then
    note_ok "Java ${JAVA_VERSION_RAW}"
  else
    note_warn "Java appears to be <11 (${JAVA_VERSION_RAW:-unknown})."
  fi
else
  note_fail "java not found on PATH. Install Java 11+ before running pipelines."
fi

# cTAKES home ----------------------------------------------------------------
BUNDLED_CTAKES="${BASE_DIR}/CtakesBun-bundle/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0"
CTAKES_ROOT="${CTAKES_HOME:-}"
if [[ -z "${CTAKES_ROOT}" ]]; then
  if [[ -d "${BUNDLED_CTAKES}" ]]; then
    note_warn "CTAKES_HOME not set; using bundled ${BUNDLED_CTAKES}. Export CTAKES_HOME for scripts."
    CTAKES_ROOT="${BUNDLED_CTAKES}"
  else
    note_fail "CTAKES_HOME not set and no bundled distribution found at ${BUNDLED_CTAKES}."
  fi
fi

if [[ -n "${CTAKES_ROOT}" ]]; then
  if [[ -d "${CTAKES_ROOT}" ]]; then
    note_ok "Using CTAKES_HOME=${CTAKES_ROOT}"
    [[ -d "${CTAKES_ROOT}/lib" ]] || note_fail "${CTAKES_ROOT} does not contain a lib/ directory."
  else
    note_fail "CTAKES_HOME=${CTAKES_ROOT} does not exist"
  fi
fi

# Pipeline sanity ------------------------------------------------------------
for key in core sectioned smoke drug; do
  p="${BASE_DIR}/pipelines/${key}/"*
  if compgen -G "$p" >/dev/null 2>&1; then
    note_ok "Pipeline files present for ${key}"
  else
    note_fail "Missing pipeline definitions under pipelines/${key}"
  fi
done

# Tools check ----------------------------------------------------------------
if ls "${BASE_DIR}/tools"/*.java >/dev/null 2>&1; then
  note_ok "Java tools present (dictionary + helpers)"
else
  note_warn "No Java tools found under tools/."
fi

# Samples --------------------------------------------------------------------
SAMPLES_DIR="${BASE_DIR}/samples/mimic"
if [[ -d "${SAMPLES_DIR}" ]]; then
  COUNT=$(find "${SAMPLES_DIR}" -type f -name '*.txt' | wc -l | awk '{print $1}')
  note_ok "Found ${COUNT} sample note(s) under samples/mimic"
else
  note_warn "samples/mimic missing. Restore the sample notes if you rely on the smoke test."
fi

# Dry run --------------------------------------------------------------------
if [[ ${ISSUES} -eq 0 && ${COUNT:-0} -gt 0 ]]; then
  if scripts/run_pipeline.sh --dry-run --pipeline sectioned --input "${SAMPLES_DIR}" --output "${BASE_DIR}/outputs/flight_check" >/dev/null 2>&1; then
    note_ok "run_pipeline.sh dry run succeeded"
  else
    note_warn "run_pipeline.sh dry run could not execute (ensure the default dictionary exists)"
  fi
fi

if [[ ${ISSUES} -eq 0 ]]; then
  if [[ ${WARNINGS} -eq 0 ]]; then
    echo "[summary] Flight check passed."
  else
    echo "[summary] Flight check completed with ${WARNINGS} warning(s)."
  fi
  exit 0
else
  echo "[summary] Flight check failed with ${ISSUES} issue(s)." >&2
  exit 1
fi
