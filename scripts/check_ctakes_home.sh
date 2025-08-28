#!/usr/bin/env bash
set -euo pipefail

# Quick diagnostics for CTAKES_HOME tree and key resources

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
CTAKES_HOME="${CTAKES_HOME:-$BASE_DIR/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0}"

echo "CTAKES_HOME: $CTAKES_HOME"
[[ -d "$CTAKES_HOME" ]] || { echo "ERROR: CTAKES_HOME not found: $CTAKES_HOME" >&2; exit 1; }

ok=1

check_file() {
  local p="$1"; local label="$2"; if [[ -f "$p" ]]; then echo "[OK] $label: $p"; else echo "[MISS] $label: $p"; ok=0; fi
}

check_dir() {
  local p="$1"; local label="$2"; if [[ -d "$p" ]]; then echo "[OK] $label: $p"; else echo "[MISS] $label: $p"; ok=0; fi
}

echo "\nCore libs present?"
for jar in ctakes-core-6.0.0.jar ctakes-type-system-6.0.0.jar ctakes-context-6.0.0.jar; do
  if ls "$CTAKES_HOME/lib/$jar" >/dev/null 2>&1; then echo "[OK] $jar"; else echo "[MISS] $jar"; ok=0; fi
done

echo "\nTemporal models present?"
check_file "$CTAKES_HOME/resources/org/apache/ctakes/temporal/models/eventannotator/model.jar" "THYME event model"

echo "\nCoreference present?"
if ls "$CTAKES_HOME/lib"/ctakes-coreference-* >/dev/null 2>&1; then echo "[OK] ctakes-coreference"; else echo "[MISS] ctakes-coreference"; ok=0; fi

echo "\nSmoking status present?"
if ls "$CTAKES_HOME/lib"/ctakes-smoking-status-* >/dev/null 2>&1; then echo "[OK] ctakes-smoking-status"; else echo "[MISS] ctakes-smoking-status"; fi

echo "\nDictionary (HSQL) present?"
DICT_DIR="$CTAKES_HOME/resources/org/apache/ctakes/dictionary/lookup/fast/FullClinical_AllTUIs"
check_file "$DICT_DIR/FullClinical_AllTUIs.script" "HSQL script"
check_file "$DICT_DIR/FullClinical_AllTUIs.properties" "HSQL properties"

echo "\nDependency model present?"
check_file "$CTAKES_HOME/resources/org/apache/ctakes/dependency/parser/models/dependency/mayo-en-dep-1.3.0.jar" "Dependency parser model"

echo "\nICU + LVG present?"
if ls "$CTAKES_HOME/lib"/icu4j-* >/dev/null 2>&1; then echo "[OK] icu4j"; else echo "[MISS] icu4j"; fi
if ls "$CTAKES_HOME/lib"/lvgdist-* >/dev/null 2>&1; then echo "[OK] lvgdist"; else echo "[MISS] lvgdist"; fi

echo "\nTemporal resource pack present?"
check_dir "$CTAKES_HOME/resources/org/apache/ctakes/temporal" "Temporal resources"

echo
if [[ $ok -eq 1 ]]; then
  echo "Diagnostics complete: no obvious misses."
else
  echo "Diagnostics complete: some resources missing (see [MISS] above)." >&2
  exit 2
fi

