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
  --pipeline <core|sectioned|smoke|drug|core_sectioned_smoke|s_core_relations_smoke>   Pipeline to exercise (default: sectioned)
  --limit <N>                              Copy the first N files into a temp dir before running (default: all)
  --with-relations                        Run with TsRelationSubPipe enabled (core/smoke/drug only)
  --manifest <file>                        Compare outputs against a saved manifest (creates baseline if missing)
  --canonicalize                           Rewrite outputs into a stable order before manifesting (default)
  --no-canonicalize                        Skip canonical rewriting before manifesting
  --deterministic                          Force single-threaded pipeline for reproducibility (default)
  --no-deterministic                       Allow autoscale / multi-threaded pipeline
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
WITH_RELATIONS=0
DRY_RUN=0
MANIFEST=""
CANONICALIZE=1
DETERMINISTIC=0
STATUS=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    -i|--input) IN_DIR="$2"; shift 2;;
    -o|--output) OUT_DIR="$2"; shift 2;;
    --pipeline) PIPELINE_KEY="$2"; shift 2;;
    --limit) LIMIT="$2"; shift 2;;
    --with-relations) WITH_RELATIONS=1; shift 1;;
    --manifest) MANIFEST="$2"; shift 2;;
    --canonicalize) CANONICALIZE=1; shift 1;;
    --no-canonicalize) CANONICALIZE=0; shift 1;;
    --deterministic) DETERMINISTIC=1; shift 1;;
    --no-deterministic) DETERMINISTIC=0; shift 1;;
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
if [[ ${WITH_RELATIONS} -eq 1 ]]; then
  ARGS+=(--with-relations)
fi
[[ ${DRY_RUN} -eq 1 ]] && ARGS+=(--dry-run)
if [[ ${DETERMINISTIC} -eq 1 ]]; then
  ARGS+=(--no-autoscale --threads 1 --xmx 4096)
fi

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
if [[ ${STATUS} -eq 0 && ${CANONICALIZE} -eq 1 ]]; then
  CANON_PY=$(command -v python3 || command -v python || true)
  if [[ -z "${CANON_PY}" ]]; then
    echo "[validate] --canonicalize requires python (python3 or python)." >&2
    STATUS=1
  else
    "${CANON_PY}" - <<'PY' "${OUT_DIR}"
import csv
import pathlib
import sys

base = pathlib.Path(sys.argv[1]).resolve()

def int_or_zero(value):
    try:
        return int(value)
    except Exception:
        return 0

def rewrite_csv(path, sort_key):
    with path.open('r', newline='', encoding='utf-8') as src:
        reader = csv.reader(src)
        try:
            header = next(reader)
        except StopIteration:
            return
        rows = list(reader)
    if not rows:
        return
    rows.sort(key=lambda row: sort_key(header, row))
    with path.open('w', newline='', encoding='utf-8') as dst:
        writer = csv.writer(dst)
        writer.writerow(header)
        writer.writerows(rows)

def rewrite_bsv(path):
    with path.open('r', encoding='utf-8') as src:
        lines = src.readlines()
    if not lines:
        return
    header, *data = lines
    data = [line.rstrip('\n') for line in data if line.strip()]
    if not data:
        return
    data.sort()
    with path.open('w', encoding='utf-8') as dst:
        dst.write(header)
        for line in data:
            dst.write(line)
            dst.write('\n')

def concept_key(header, row):
    begin_idx = header.index('core:Begin') if 'core:Begin' in header else -1
    end_idx = header.index('core:End') if 'core:End' in header else -1
    cui_idx = header.index('core:CUI') if 'core:CUI' in header else -1
    return (
        int_or_zero(row[begin_idx]) if 0 <= begin_idx < len(row) else 0,
        int_or_zero(row[end_idx]) if 0 <= end_idx < len(row) else 0,
        row[cui_idx] if 0 <= cui_idx < len(row) else '',
        row,
    )

def rxnorm_key(header, row):
    begin_idx = header.index('Begin') if 'Begin' in header else -1
    end_idx = header.index('End') if 'End' in header else -1
    cui_idx = header.index('RxCUI') if 'RxCUI' in header else -1
    return (
        int_or_zero(row[begin_idx]) if 0 <= begin_idx < len(row) else 0,
        int_or_zero(row[end_idx]) if 0 <= end_idx < len(row) else 0,
        row[cui_idx] if 0 <= cui_idx < len(row) else '',
        row,
    )

concept_dir = base / 'concepts'
if concept_dir.is_dir():
    for csv_path in sorted(concept_dir.glob('*.csv')):
        rewrite_csv(csv_path, concept_key)

cui_dir = base / 'cui_counts'
if cui_dir.is_dir():
    for bsv_path in sorted(cui_dir.glob('*.bsv')):
        rewrite_bsv(bsv_path)

rx_dir = base / 'rxnorm'
if rx_dir.is_dir():
    for csv_path in sorted(rx_dir.glob('*.csv')):
        rewrite_csv(csv_path, rxnorm_key)
PY
  fi
fi

if [[ ${STATUS} -eq 0 && -n "${MANIFEST}" ]]; then
  PY_BIN=$(command -v python3 || command -v python || true)
  if [[ -z "${PY_BIN}" ]]; then
    echo "[validate] Manifest comparison requires python (python3 or python)." >&2
    STATUS=1
  else
    REPORT_FILE="${OUT_DIR%/}/validation_report.log"
    if ! "${PY_BIN}" "${BASE_DIR}/scripts/semantic_manifest.py" --outputs "${OUT_DIR}" --manifest "${MANIFEST}" --report "${REPORT_FILE}" --processed-count "${files_processed}"; then
      STATUS=1
    fi
  fi
fi

exit ${STATUS}
