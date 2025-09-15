#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: scripts/validate.sh -i <input_dir> -o <output_dir> [options]
Options:
  --pipeline <core|sectioned|smoke|drug>   Pipeline to exercise (default: sectioned)
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

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
RUNNER="${BASE_DIR}/scripts/run_pipeline.sh"
if [[ ! -x "${RUNNER}" ]]; then
  echo "[validate] Missing run_pipeline.sh helper" >&2
  exit 1
fi

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
ARGS=("${RUNNER}" -i "${PIPE_INPUT}" -o "${OUT_DIR}" --pipeline "${PIPELINE_KEY}")
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

if [[ ${STATUS} -eq 0 && -n "${MANIFEST}" ]]; then
  TMP_MANIFEST=$(mktemp)
  find "${OUT_DIR}" -type f \( -path "${OUT_DIR}/concepts/*.csv" -o -path "${OUT_DIR}/cui_count/*.bsv" -o -path "${OUT_DIR}/rxnorm/*.csv" \) \
    | sort \
    | while read -r file; do
        rel="${file#${OUT_DIR}/}"
        sha=$(sha256sum "${file}" | awk '{print $1}')
        printf "%s  %s\n" "$sha" "$rel"
      done > "${TMP_MANIFEST}"
  if [[ -f "${MANIFEST}" ]]; then
    if cmp -s "${MANIFEST}" "${TMP_MANIFEST}"; then
      echo "[validate] Manifest matches ${MANIFEST}"
    else
      echo "[validate] Manifest differs from ${MANIFEST}" >&2
      diff -u "${MANIFEST}" "${TMP_MANIFEST}" || true
      STATUS=1
    fi
    rm -f "${TMP_MANIFEST}"
  else
    mkdir -p "$(dirname "${MANIFEST}")"
    mv "${TMP_MANIFEST}" "${MANIFEST}"
    echo "[validate] Baseline manifest saved to ${MANIFEST}"
  fi
fi

exit ${STATUS}
