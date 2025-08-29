#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/.. && pwd)"
cd "$ROOT_DIR"

# Recompile reporting tools
CTAKES_HOME_DEFAULT="$ROOT_DIR/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0"
CTAKES_HOME="${CTAKES_HOME:-$CTAKES_HOME_DEFAULT}"
mkdir -p .build_tools
find tools/reporting -name "*.java" -print0 | xargs -0 javac \
  -cp "$CTAKES_HOME/desc:$CTAKES_HOME/resources:$CTAKES_HOME/config:$CTAKES_HOME/config/*:$CTAKES_HOME/lib/*:$ROOT_DIR/.build_tools" \
  -d .build_tools

echo "[build] compile OK"

# Run ExcelXmlReport with user args
exec java -cp .build_tools tools.reporting.ExcelXmlReport "$@"

