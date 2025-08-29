#!/usr/bin/env bash
set -euo pipefail

# Simple preflight for reporting tools: checks env, conflicts, compiles

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"
cd "$ROOT_DIR"

echo "[preflight] Repo: $ROOT_DIR"

# Java availability
if ! command -v java >/dev/null 2>&1; then
  echo "[preflight] ERROR: 'java' not found in PATH" >&2
  exit 1
fi
if ! command -v javac >/dev/null 2>&1; then
  echo "[preflight] ERROR: 'javac' not found in PATH" >&2
  exit 1
fi
echo "[preflight] java  : $(java -version 2>&1 | head -n1)"
echo "[preflight] javac : $(javac -version 2>&1)"

# Conflict markers check
if rg -n "^(<<<<<<<|=======|>>>>>>>)" tools/reporting/ExcelXmlReport.java >/dev/null 2>&1; then
  echo "[preflight] ERROR: merge conflict markers present in tools/reporting/ExcelXmlReport.java" >&2
  rg -n "^(<<<<<<<|=======|>>>>>>>)" tools/reporting/ExcelXmlReport.java || true
  exit 1
fi
echo "[preflight] conflict markers: none"

# CTAKES_HOME (optional for compile of reporting tools)
CTAKES_HOME_DEFAULT="$ROOT_DIR/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0"
CTAKES_HOME="${CTAKES_HOME:-$CTAKES_HOME_DEFAULT}"
echo "[preflight] CTAKES_HOME: $CTAKES_HOME"
if [ -d "$CTAKES_HOME" ]; then
  echo "[preflight] ctakes dir present"
else
  echo "[preflight] note: $CTAKES_HOME not found; reporting tools compile without it"
fi

# Compile
mkdir -p .build_tools
find tools/reporting -name "*.java" -print0 | xargs -0 javac \
  -cp "$CTAKES_HOME/desc:$CTAKES_HOME/resources:$CTAKES_HOME/config:$CTAKES_HOME/config/*:$CTAKES_HOME/lib/*:$ROOT_DIR/.build_tools" \
  -d .build_tools

echo "[preflight] compile: OK (.build_tools ready)"
echo "[preflight] done"

