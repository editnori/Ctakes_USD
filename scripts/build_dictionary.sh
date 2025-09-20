#!/usr/bin/env bash
set -euo pipefail

# Build (and optionally run) the headless dictionary builder.
# Usage:
#   scripts/build_dictionary.sh --compile-only
#   scripts/build_dictionary.sh -- <tools.dictionary.HeadlessDictionaryBuilder args>
# Environment:
#   CTAKES_HOME - required path to the apache-ctakes installation.
#   BUILD_DIR   - optional custom output directory for compiled classes (defaults to build/dictionary).

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
CTAKES_HOME="${CTAKES_HOME:-}"
if [[ -z "${CTAKES_HOME}" ]]; then
  echo "[dict] Set CTAKES_HOME to your apache-ctakes install root" >&2
  exit 1
fi
BUILD_DIR="${BUILD_DIR:-$BASE_DIR/build/dictionary}"
mkdir -p "${BUILD_DIR}"

if [[ ! -d "${CTAKES_HOME}/lib" ]]; then
  echo "[dict] Expected CTAKES_HOME/lib to exist (found ${CTAKES_HOME})" >&2
  exit 1
fi

# Resolve classpath separator per platform
CLASSPATH_SEP=:
case "$(uname -s 2>/dev/null)" in
  MINGW*|MSYS*|CYGWIN*) CLASSPATH_SEP=';';;
  Darwin*) CLASSPATH_SEP=':';;
  Linux*) CLASSPATH_SEP=':';;
  *) if [[ "${OS:-}" == "Windows_NT" ]]; then CLASSPATH_SEP=';'; fi;;
esac

SRC=("${BASE_DIR}/tools/dictionary/HeadlessDictionaryCreator.java" "${BASE_DIR}/tools/dictionary/HeadlessDictionaryBuilder.java")
JAVAC_CP="${CTAKES_HOME}/lib/*"

# Compile when build dir is empty or sources newer than classes
javac -cp "${JAVAC_CP}" -d "${BUILD_DIR}" "${SRC[@]}"

if [[ "${1:-}" == "--compile-only" ]]; then
  echo "[dict] Compiled dictionary builder classes into ${BUILD_DIR}" >&2
  exit 0
fi

JAVA_CP="${BUILD_DIR}${CLASSPATH_SEP}${CTAKES_HOME}/lib/*"
java -cp "${JAVA_CP}" tools.dictionary.HeadlessDictionaryBuilder "$@"
