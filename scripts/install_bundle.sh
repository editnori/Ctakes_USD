#!/usr/bin/env bash
set -euo pipefail

# Install the exact cTAKES bundle for this repo.
#
# Looks for a local CtakesBun-bundle.tgz at repo root first, otherwise
# downloads from a release URL. Extracts into the repo so that
# CTAKES_HOME defaults work without extra config.
#
# Usage:
#   scripts/install_bundle.sh [-f <bundle.tgz>] [-u <url>] [-s <sha256>] [--deps]

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BUNDLE_FILE="${BASE_DIR}/CtakesBun-bundle.tgz"
BUNDLE_URL="${BUNDLE_URL:-https://github.com/editnori/Ctakes_USD/releases/download/bundle/CtakesBun-bundle.tgz}"
EXPECT_SHA=""
INSTALL_DEPS=0

need_tool() {
  local t="$1"; command -v "$t" >/dev/null 2>&1 || return 1
}

install_deps() {
  if ! command -v apt-get >/dev/null 2>&1; then
    echo "apt-get not found. --deps is only supported on Debian/Ubuntu." >&2
    return 1
  fi
  local SUDO=""
  if [[ "$(id -u)" -ne 0 ]]; then
    if command -v sudo >/dev/null 2>&1; then SUDO="sudo "; else
      echo "Need root privileges to install packages. Re-run with sudo or as root." >&2
      return 1
    fi
  fi
  echo "Installing prerequisites via apt-get (Java 17 JDK + CLI tools)..."
  ${SUDO}apt-get update -y
  ${SUDO}apt-get install -y openjdk-17-jdk curl coreutils findutils gawk sed grep tar
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    -f|--file) BUNDLE_FILE="$2"; shift 2;;
    -u|--url) BUNDLE_URL="$2"; shift 2;;
    -s|--sha256) EXPECT_SHA="$2"; shift 2;;
    --deps) INSTALL_DEPS=1; shift 1;;
    *) echo "Unknown arg: $1" >&2; exit 2;;
  esac
done

if [[ "$INSTALL_DEPS" -eq 1 ]]; then
  install_deps || { echo "Dependency install failed." >&2; exit 2; }
fi

# Verify basic tools and Java
MISSING=()
for t in curl tar sha256sum sed awk grep find xargs ln; do need_tool "$t" || MISSING+=("$t"); done
if [[ ${#MISSING[@]} -gt 0 ]]; then
  echo "Missing required tools: ${MISSING[*]}" >&2
  echo "Run again with --deps on Ubuntu/Debian to auto-install, or install manually." >&2
  exit 2
fi

if ! command -v java >/dev/null 2>&1 || ! command -v javac >/dev/null 2>&1 || ! command -v jar >/dev/null 2>&1; then
  echo "Java 17 JDK (java, javac, jar) not found in PATH." >&2
  if [[ "$INSTALL_DEPS" -eq 0 ]]; then
    echo "Re-run with --deps on Ubuntu/Debian to install openjdk-17-jdk." >&2
  fi
  exit 2
fi

if [[ ! -f "$BUNDLE_FILE" ]]; then
  echo "Bundle not found locally at $BUNDLE_FILE"
  echo "Downloading from: $BUNDLE_URL"
  curl -L "$BUNDLE_URL" -o "$BUNDLE_FILE"
fi

if [[ -n "$EXPECT_SHA" ]]; then
  echo "$EXPECT_SHA  $BUNDLE_FILE" | sha256sum -c -
fi

echo "Extracting bundle: $BUNDLE_FILE"
tar -xzf "$BUNDLE_FILE" -C "$BASE_DIR"

echo "Bundle extracted. CTAKES_HOME default should now resolve to:"
echo "  $BASE_DIR/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0"
echo "Run: scripts/run_compare_cluster.sh -i <in> -o <out> --reports"

echo
echo "java -version:"
java -version || true
echo "javac -version:"
javac -version || true
echo "jar --version:"
jar --version || true
