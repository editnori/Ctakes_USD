#!/usr/bin/env bash
set -euo pipefail

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
ENV_FILE="${BASE_DIR}/.ctakes_env"
if [[ -f "${ENV_FILE}" ]]; then
  # shellcheck disable=SC1090
  source "${ENV_FILE}"
fi

DEFAULT_FLIGHT_UMLS_KEY="6370dcdd-d438-47ab-8749-5a8fb9d013f2"

write_env_var() {
  local var="$1"
  local value="$2"
  local tmp="${ENV_FILE}.tmp"
  if [[ -f "${ENV_FILE}" ]]; then
    awk -v var="$var" '
      /bin/bash ~ "^[[:space:]]*export[[:space:]]+" var "=" { next }
      { print }
    ' "${ENV_FILE}" > "${tmp}"
  else
    : > "${tmp}"
  fi
  printf 'export %s=%q
' "$var" "$value" >> "${tmp}"
  mv "${tmp}" "${ENV_FILE}"
  echo "[flight_check] Persisted ${var} to ${ENV_FILE}"
}

maybe_prompt_env_var() {
  local var="$1"
  local value="$2"
  local prompt="$3"
  local desired existing
  desired=$(printf %q "$value")
  if [[ -f "${ENV_FILE}" ]]; then
    existing=$(grep -E "^[[:space:]]*export[[:space:]]+${var}=" "${ENV_FILE}" | tail -n1 2>/dev/null || true)
    if [[ "$existing" == "export ${var}=${desired}" ]]; then
      return
    fi
  fi
  if [[ ! -t 0 || ! -t 1 ]]; then
    return
  fi
  read -r -p "${prompt}" reply
  if [[ "$reply" =~ ^[Yy] ]]; then
    write_env_var "$var" "$value"
    export "$var"="$value"
  fi
}

ensure_default_umls_key() {
  if [[ -n "${UMLS_KEY:-}" ]]; then
    return
  fi
  if [[ -f "${ENV_FILE}" ]]; then
    if grep -qE "^[[:space:]]*export[[:space:]]+UMLS_KEY=" "${ENV_FILE}"; then
      return
    fi
  fi
  write_env_var UMLS_KEY "${DEFAULT_FLIGHT_UMLS_KEY}"
  export UMLS_KEY="${DEFAULT_FLIGHT_UMLS_KEY}"
}

ISSUES=0

BASH_BIN="${BASH:-bash}"
RUN_PIPELINE="${BASE_DIR}/scripts/run_pipeline.sh"
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
BUNDLED_CTAKES_ALT="${BASE_DIR}/Ctakes_USD_clean/CtakesBun-bundle/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0"
CTAKES_ROOT="${CTAKES_HOME:-}"
if [[ -z "${CTAKES_ROOT}" ]]; then
  if [[ -d "${BUNDLED_CTAKES}" ]]; then
    note_ok "CTAKES_HOME not set; defaulting to bundled ${BUNDLED_CTAKES} (persist CTAKES_HOME to silence this message)."
    CTAKES_ROOT="${BUNDLED_CTAKES}"
  elif [[ -d "${BUNDLED_CTAKES_ALT}" ]]; then
    note_ok "CTAKES_HOME not set; defaulting to bundled ${BUNDLED_CTAKES_ALT} (persist CTAKES_HOME to silence this message)."
    CTAKES_ROOT="${BUNDLED_CTAKES_ALT}"
  else
    note_fail "CTAKES_HOME not set and no bundled distribution found at ${BUNDLED_CTAKES} (run scripts/get_bundle.sh)."
  fi
fi

if [[ -n "${CTAKES_ROOT}" ]]; then
  maybe_prompt_env_var CTAKES_HOME "${CTAKES_ROOT}" "Persist CTAKES_HOME=${CTAKES_ROOT} to ${ENV_FILE} for future runs? [y/N] "
  if [[ -d "${CTAKES_ROOT}" ]]; then
    note_ok "Using CTAKES_HOME=${CTAKES_ROOT}"
    [[ -d "${CTAKES_ROOT}/lib" ]] || note_fail "${CTAKES_ROOT} does not contain a lib/ directory."
  else
    note_fail "CTAKES_HOME=${CTAKES_ROOT} does not exist"
  fi
fi

ensure_default_umls_key

# Pipeline sanity ------------------------------------------------------------
for key in core sectioned smoke core_sectioned_smoke drug; do
  case "$key" in
    core_sectioned_smoke) p="${BASE_DIR}/pipelines/combined/"*; friendly="pipelines/combined";;
    *) p="${BASE_DIR}/pipelines/${key}/"*; friendly="pipelines/${key}";;
  esac
  if compgen -G "$p" >/dev/null 2>&1; then
    note_ok "Pipeline files present for ${key}"
  else
    note_fail "Missing pipeline definitions under ${friendly}"
  fi
done

# Tools check ----------------------------------------------------------------
if [[ -d "${BASE_DIR}/tools" ]] && find "${BASE_DIR}/tools" -type f -name "*.java" -print -quit >/dev/null 2>&1; then
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
  if [[ ! -f "${RUN_PIPELINE}" ]]; then
    note_warn "scripts/run_pipeline.sh missing; skipping dry run"
  else
    DRY_RUN_OUTPUT=$("${BASH_BIN}" "${RUN_PIPELINE}" --dry-run --pipeline sectioned --input "${SAMPLES_DIR}" --output "${BASE_DIR}/outputs/flight_check" 2>&1)
    if [[ $? -eq 0 ]]; then
      note_ok "run_pipeline.sh dry run succeeded"
      if [[ -n "${DRY_RUN_OUTPUT}" ]]; then
        while IFS= read -r line; do
          case "${line}" in
            "[pipeline] Autoscale recommends "*|"[async] autoscale -> "*)
              echo "    ${line}"
              ;;
          esac
        done <<< "${DRY_RUN_OUTPUT}"
      fi
    else
      note_warn "run_pipeline.sh dry run could not execute (ensure the default dictionary exists)"
      if [[ -n "${DRY_RUN_OUTPUT}" ]]; then
        echo "${DRY_RUN_OUTPUT}" >&2
      fi
    fi
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


