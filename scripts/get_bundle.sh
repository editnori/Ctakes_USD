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
TMP_FILE="${TMPDIR:-/tmp}/${ASSET_NAME}.download"
TMP_DIR="${TMPDIR:-/tmp}/ctakes_bundle.$$"

mkdir -p "${TMP_DIR}"
trap 'rm -rf "${TMP_DIR}" "${TMP_FILE}"' EXIT

echo "[get_bundle] Downloading ${ASSET_NAME} from ${DOWNLOAD_URL}"
curl -L --fail --progress-bar -o "${TMP_FILE}" "${DOWNLOAD_URL}"

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
LOCAL_DICT="${DICT_DIR}/FullClinical_AllTUIs_local.xml"
DEFAULT_DICT="${DICT_DIR}/FullClinical_AllTUIs.xml"
if [[ -f "${LOCAL_DICT}" && -f "${DEFAULT_DICT}" ]]; then
  echo "[get_bundle] Bundle ready at ${EXPECTED_CTAKES}"
else
  echo "[get_bundle] Warning: dictionary files not found under ${DICT_DIR}" >&2
fi
