#!/usr/bin/env bash
set -euo pipefail

RELEASE_TAG="${1:-bundle}"
ASSET_NAME="CtakesBun-bundle.tgz"
BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
EXPECTED_DIR="${BASE_DIR}/CtakesBun-bundle"
EXPECTED_CTAKES="${EXPECTED_DIR}/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0"

if [[ -d "${EXPECTED_CTAKES}" ]]; then
  echo "[get_bundle] Found existing bundle at ${EXPECTED_CTAKES}"
  exit 0
fi

DOWNLOAD_URL="https://github.com/editnori/Ctakes_USD/releases/download/${RELEASE_TAG}/${ASSET_NAME}"
CHECKSUM_URL="${DOWNLOAD_URL}.sha256"
TMP_FILE="${TMPDIR:-/tmp}/${ASSET_NAME}.download"
TMP_SUM="${TMP_FILE}.sha256"
TMP_DIR="${TMPDIR:-/tmp}/ctakes_bundle.$$"

mkdir -p "${TMP_DIR}"
trap 'rm -rf "${TMP_DIR}" "${TMP_FILE}" "${TMP_SUM}"' EXIT

echo "[get_bundle] Downloading ${ASSET_NAME} from ${DOWNLOAD_URL}"
curl -L --fail --progress-bar -o "${TMP_FILE}" "${DOWNLOAD_URL}"

if curl -L --fail --silent --show-error -o "${TMP_SUM}" "${CHECKSUM_URL}"; then
  PYTHON_BIN=$(command -v python3 || command -v python || true)
  if [[ -z "${PYTHON_BIN}" ]]; then
    echo "[get_bundle] Warning: python not available; skipping checksum verification" >&2
  else
    EXPECTED_SUM=$(head -n 1 "${TMP_SUM}" | awk '{print $1}')
    if [[ -z "${EXPECTED_SUM}" ]]; then
      echo "[get_bundle] Warning: checksum file ${CHECKSUM_URL} is empty; skipping verification" >&2
    else
      echo "[get_bundle] Verifying SHA-256 checksum"
      ACTUAL_SUM=$("${PYTHON_BIN}" - <<'PY' "${TMP_FILE}"
import hashlib
import sys
path = sys.argv[1]
h = hashlib.sha256()
with open(path, 'rb') as stream:
    for chunk in iter(lambda: stream.read(1024 * 1024), b''):
        h.update(chunk)
print(h.hexdigest())
PY
)
      if [[ "${ACTUAL_SUM,,}" != "${EXPECTED_SUM,,}" ]]; then
        echo "[get_bundle] Checksum mismatch for ${ASSET_NAME}" >&2
        exit 1
      fi
      echo "[get_bundle] Checksum verified"
    fi
  fi
else
  echo "[get_bundle] Warning: checksum file not found at ${CHECKSUM_URL}; skipping verification" >&2
fi

echo "[get_bundle] Extracting into temporary directory"
tar -xzf "${TMP_FILE}" -C "${TMP_DIR}"

if [[ -d "${TMP_DIR}/Ctakes_USD_clean/CtakesBun-bundle" ]]; then
  mv "${TMP_DIR}/Ctakes_USD_clean/CtakesBun-bundle" "${TMP_DIR}/CtakesBun-bundle"
fi

if [[ -d "${TMP_DIR}/CtakesBun-bundle" ]]; then
  rm -rf "${EXPECTED_DIR}"
  mv "${TMP_DIR}/CtakesBun-bundle" "${EXPECTED_DIR}"
else
  echo "[get_bundle] Unexpected bundle layout. Expected CtakesBun-bundle in archive." >&2
  exit 1
fi

if [[ ! -d "${EXPECTED_CTAKES}" ]]; then
  echo "[get_bundle] Extraction completed but ${EXPECTED_CTAKES} not found" >&2
  exit 1
fi

DICT_DIR="${EXPECTED_CTAKES}/resources/org/apache/ctakes/dictionary/lookup/fast"
LOCAL_DICT="${DICT_DIR}/KidneyStone_SDOH_local.xml"
DEFAULT_DICT="${DICT_DIR}/KidneyStone_SDOH.xml"
if [[ -f "${LOCAL_DICT}" && -f "${DEFAULT_DICT}" ]]; then
  echo "[get_bundle] Bundle ready at ${EXPECTED_CTAKES}"
else
  echo "[get_bundle] Warning: KidneyStone_SDOH dictionary not found under ${DICT_DIR}" >&2
fi

