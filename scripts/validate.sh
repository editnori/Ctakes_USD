#!/usr/bin/env bash
set -euo pipefail

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
ENV_FILE="${BASE_DIR}/.ctakes_env"
if [[ -f "${ENV_FILE}" ]]; then
  # shellcheck disable=SC1090
  source "${ENV_FILE}"
fi


usage() {
  cat <<'USAGE'
Usage: scripts/validate.sh -i <input_dir> -o <output_dir> [options]
Options:
  --pipeline <core|sectioned|smoke|drug|core_sectioned_smoke>   Pipeline to exercise (default: sectioned)
  --limit <N>                              Copy the first N files into a temp dir before running (default: all)
  --with-temporal                          Run with TsTemporalSubPipe enabled
  --with-coref                             Run with TsCorefSubPipe enabled
  --manifest <file>                        Compare outputs against a saved manifest (creates baseline if missing)
  --dry-run                                Print the pipeline command instead of executing
  --help                                   Show this help text

Runs scripts/run_pipeline.sh with sensible defaults. Use --limit to perform a quick
validation pass on a small sample of notes.
USAGE
}

PIPELINE_KEY="sectioned"
IN_DIR=""
OUT_DIR=""
LIMIT=0
WITH_TEMPORAL=0
WITH_COREF=0
DRY_RUN=0
MANIFEST=""
STATUS=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    -i|--input) IN_DIR="$2"; shift 2;;
    -o|--output) OUT_DIR="$2"; shift 2;;
    --pipeline) PIPELINE_KEY="$2"; shift 2;;
    --limit) LIMIT="$2"; shift 2;;
    --with-temporal) WITH_TEMPORAL=1; shift 1;;
    --with-coref) WITH_COREF=1; shift 1;;
    --manifest) MANIFEST="$2"; shift 2;;
    --dry-run) DRY_RUN=1; shift 1;;
    --help|-h) usage; exit 0;;
    *) echo "Unknown option: $1" >&2; usage >&2; exit 1;;
  esac

done

if [[ -z "${IN_DIR}" || -z "${OUT_DIR}" ]]; then
  echo "[validate] --input and --output are required" >&2
  usage >&2
  exit 1
fi

if ! [[ "${LIMIT}" =~ ^[0-9]+$ ]]; then
  echo "[validate] --limit must be a non-negative integer" >&2
  exit 1
fi

if [[ ! -d "${IN_DIR}" ]]; then
  echo "[validate] Input directory does not exist: ${IN_DIR}" >&2
  exit 1
fi

RUNNER="${BASE_DIR}/scripts/run_pipeline.sh"
if [[ ! -f "${RUNNER}" ]]; then
  echo "[validate] Missing run_pipeline.sh helper" >&2
  exit 1
fi

RUNNER_CMD=("${BASH:-bash}" "${RUNNER}")

PIPE_INPUT="${IN_DIR}"
TMP_ROOT=""
if [[ ${LIMIT} -gt 0 ]]; then
  TMP_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/ctakes_validate.XXXXXX")"
  trap '[[ -n "${TMP_ROOT}" ]] && rm -rf "${TMP_ROOT}"' EXIT
  PIPE_INPUT="${TMP_ROOT}/input"
  mkdir -p "${PIPE_INPUT}"
  PYTHON=$(command -v python3 || command -v python || true)
  if [[ -z "${PYTHON}" ]]; then
    echo "[validate] --limit requires python (python3 or python) to copy samples" >&2
    exit 1
  fi
  "${PYTHON}" <<PY
import os, shutil
src = os.path.abspath(${IN_DIR@Q})
dest = os.path.abspath(${PIPE_INPUT@Q})
limit = int(${LIMIT@Q})
allowed = {'.txt', '.xmi', '.xml'}
count = 0
for root, _, files in os.walk(src):
    rel_root = os.path.relpath(root, src)
    for name in sorted(files):
        if limit and count >= limit:
            break
        ext = os.path.splitext(name)[1].lower()
        if allowed and ext and ext not in allowed:
            continue
        src_path = os.path.join(root, name)
        rel_path = name if rel_root == os.curdir else os.path.join(rel_root, name)
        dest_path = os.path.join(dest, rel_path)
        os.makedirs(os.path.dirname(dest_path), exist_ok=True)
        shutil.copy2(src_path, dest_path)
        count += 1
    if limit and count >= limit:
        break
if count == 0:
    raise SystemExit('No files copied from %s (supported extensions: %s)' % (src, ', '.join(sorted(allowed))))
print('Copied %d file(s) into %s' % (count, dest))
PY
fi

mkdir -p "${OUT_DIR}"
ARGS=("${RUNNER_CMD[@]}" -i "${PIPE_INPUT}" -o "${OUT_DIR}" --pipeline "${PIPELINE_KEY}")
[[ ${WITH_TEMPORAL} -eq 1 ]] && ARGS+=(--with-temporal)
[[ ${WITH_COREF} -eq 1 ]] && ARGS+=(--with-coref)
[[ ${DRY_RUN} -eq 1 ]] && ARGS+=(--dry-run)

if [[ ${DRY_RUN} -eq 1 ]]; then
  printf '[validate] '
  printf '%q ' "${ARGS[@]}"
  printf '\n'
  exit 0
fi

if ! "${ARGS[0]}" "${ARGS[@]:1}"; then
  STATUS=1
fi

files_processed=0
if [[ -d "${OUT_DIR}/concepts" ]]; then
  files_processed=$(find "${OUT_DIR}/concepts" -type f -name '*.csv' | wc -l | awk '{print $1}')
fi
base_manifest_count=0
if [[ -f "${MANIFEST}" ]]; then
  base_manifest_count=$(wc -l < "${MANIFEST}" | awk '{print $1}')
fi

if [[ ${STATUS} -eq 0 && -n "${MANIFEST}" ]]; then
  TMP_MANIFEST=$(mktemp)
  find "${OUT_DIR}" -type f \( -path "${OUT_DIR}/concepts/*.csv" -o -path "${OUT_DIR}/cui_counts/*.bsv" -o -path "${OUT_DIR}/rxnorm/*.csv" \) \
    | sort \
    | while read -r file; do
        rel="${file#${OUT_DIR}/}"
        sha=$(sha256sum "${file}" | awk '{print $1}')
        printf "%s  %s\n" "$sha" "$rel"
      done > "${TMP_MANIFEST}"
  timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  COUNTS_OUTPUT=$(python - <<'PY' "${TMP_MANIFEST}" "${MANIFEST}"
import os
import sys
cats = ("concepts", "cui_counts", "rxnorm")
def counts(path):
    totals = [0, 0, 0]
    if not path or not os.path.isfile(path):
        return totals
    with open(path, encoding="utf-8") as handle:
        for line in handle:
            parts = line.strip().split(None, 1)
            if len(parts) != 2:
                continue
            rel = parts[1]
            for idx, cat in enumerate(cats):
                if rel.startswith(cat + "/"):
                    totals[idx] += 1
    return totals
actual = counts(sys.argv[1])
baseline = counts(sys.argv[2]) if len(sys.argv) > 2 and sys.argv[2] else [0, 0, 0]
print(*(actual + baseline))
PY
)
  read -r concepts_actual cui_actual rx_actual concepts_base cui_base rx_base <<< "${COUNTS_OUTPUT:-0 0 0 0 0 0}"
  summary_actual_text="concepts:${concepts_actual}, cui_counts:${cui_actual}, rxnorm:${rx_actual}"
  summary_compare="concepts: current ${concepts_actual} vs baseline ${concepts_base}; cui_counts: current ${cui_actual} vs baseline ${cui_base}; rxnorm: current ${rx_actual} vs baseline ${rx_base}"
  REPORT_FILE="${OUT_DIR%/}/validation_report.log"
  if [[ -f "${MANIFEST}" ]]; then
    if cmp -s "${MANIFEST}" "${TMP_MANIFEST}"; then
      echo "[validate] Manifest matches ${MANIFEST}"
      if [[ ${files_processed} -gt 0 ]]; then
        if [[ ${base_manifest_count} -gt 0 ]]; then
          echo "[validate] ${files_processed}/${base_manifest_count} files matched the baseline."
        else
          echo "[validate] Processed ${files_processed} files; no baseline manifest entries to compare."
        fi
      fi
      echo "[validate] All outputs validated at ${timestamp} (${summary_actual_text})."
      {
        echo "timestamp=${timestamp}"
        echo "status=match"
        echo "manifest=${MANIFEST}"
        echo "concepts=${concepts_actual}"
        echo "cui_counts=${cui_actual}"
        echo "rxnorm=${rx_actual}"
        echo
      } >> "${REPORT_FILE}"
    else
      echo "[validate] Manifest differs from ${MANIFEST}" >&2
      diff -u "${MANIFEST}" "${TMP_MANIFEST}" || true
      ACTUAL_COUNT=$(wc -l < "${TMP_MANIFEST}" | awk '{print $1}')
      BASELINE_COUNT=$(wc -l < "${MANIFEST}" | awk '{print $1}')
      echo "[validate] Baseline entries: ${BASELINE_COUNT}; current entries: ${ACTUAL_COUNT}." >&2
      echo "[validate] Validation mismatch at ${timestamp} (${summary_compare})." >&2
      {
        echo "timestamp=${timestamp}"
        echo "status=diff"
        echo "manifest=${MANIFEST}"
        echo "concepts-current=${concepts_actual}"
        echo "concepts-baseline=${concepts_base}"
        echo "cui_counts-current=${cui_actual}"
        echo "cui_counts-baseline=${cui_base}"
        echo "rxnorm-current=${rx_actual}"
        echo "rxnorm-baseline=${rx_base}"
        echo
      } >> "${REPORT_FILE}"
      STATUS=1
    fi
    rm -f "${TMP_MANIFEST}"
  else
    mkdir -p "$(dirname "${MANIFEST}")"
    mv "${TMP_MANIFEST}" "${MANIFEST}"
    echo "[validate] Baseline manifest saved to ${MANIFEST}"
    echo "[validate] Baseline captured at ${timestamp} (${summary_actual_text})."
    {
      echo "timestamp=${timestamp}"
      echo "status=baseline-created"
      echo "manifest=${MANIFEST}"
      echo "concepts=${concepts_actual}"
      echo "cui_counts=${cui_actual}"
      echo "rxnorm=${rx_actual}"
      echo
    } >> "${REPORT_FILE}"
  fi
fi

exit ${STATUS}


