#!/usr/bin/env bash
set -euo pipefail

RELEASE_TAG="${1:-bundle}"
ASSET_NAME="CtakesBun-bundle.tgz"
BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BUNDLE_DIR="${BASE_DIR}/CtakesBun-bundle/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0"

if [[ -d "${BUNDLE_DIR}" ]]; then
  echo "[get_bundle] Found existing bundle at ${BUNDLE_DIR}"
  exit 0
fi

DOWNLOAD_URL="https://github.com/editnori/Ctakes_USD/releases/download/${RELEASE_TAG}/${ASSET_NAME}"
TMP_FILE="${TMPDIR:-/tmp}/${ASSET_NAME}.download"

echo "[get_bundle] Downloading ${ASSET_NAME} from ${DOWNLOAD_URL}"
curl -L --fail --progress-bar -o "${TMP_FILE}" "${DOWNLOAD_URL}"

echo "[get_bundle] Extracting into ${BASE_DIR}"
tar -xzf "${TMP_FILE}" -C "${BASE_DIR}"
rm -f "${TMP_FILE}"

if [[ -d "${BUNDLE_DIR}" ]]; then
  echo "[get_bundle] Bundle ready at ${BUNDLE_DIR}"
else
  echo "[get_bundle] Extraction completed but ${BUNDLE_DIR} not found" >&2
  exit 1
fi
