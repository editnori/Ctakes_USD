#!/usr/bin/env bash
set -euo pipefail

# Validate cTAKES pipelines on a 100-note MIMIC sample.
# - Creates a sample subset (100 .txt) from samples/mimic/ (flat dir)
# - Runs compare pipelines with modest parallelism
# - Builds a lightweight manifest (hashes + counts) for regression checks
# - If samples/mimic_output/manifest.txt exists, compares against it; otherwise seeds it
#
# Usage:
#   scripts/validate_mimic.sh [-i <mimic_dir>] [-n <count>] [-o <out_dir>]
#                             [--runners N] [--threads N] [--xmx MB] [--seed VAL]
#                             [--consolidate-async]

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
CTAKES_HOME="${CTAKES_HOME:-$BASE_DIR/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0}"

IN_DIR="$BASE_DIR/samples/mimic"
COUNT=100
OUT_BASE="$BASE_DIR/outputs/validation_mimic"
# Subset handling: link (default hardlink), copy, or reuse (use IN_DIR directly when it already has COUNT files)
SUBSET_MODE="link"
_EXPLICIT_SUBSET_MODE=""
RUNNERS="${RUNNERS:-4}"
THREADS="${THREADS:-4}"
XMX_MB="${XMX_MB:-4096}"
SEED="${SEED:-42}"
CONSOLIDATE_ASYNC=0
ONLY=""

UPDATE_BASELINE=0
while [[ $# -gt 0 ]]; do
  case "$1" in
    -i|--in) IN_DIR="$2"; shift 2;;
    -n|--count) COUNT="$2"; shift 2;;
    -o|--out) OUT_BASE="$2"; shift 2;;
    --runners) RUNNERS="$2"; shift 2;;
    --threads) THREADS="$2"; shift 2;;
    --xmx) XMX_MB="$2"; shift 2;;
    --seed) SEED="$2"; shift 2;;
    --only) ONLY="$2"; shift 2;;
    --subset-mode) SUBSET_MODE="$2"; _EXPLICIT_SUBSET_MODE=1; shift 2;;
    --consolidate-async) CONSOLIDATE_ASYNC=1; shift 1;;
    --update-baseline) UPDATE_BASELINE=1; shift 1;;
    *) echo "Unknown arg: $1" >&2; exit 2;;
  esac
done

## Ensure UMLS key default (hardcoded fallback if env not set)
export UMLS_KEY="${UMLS_KEY:-6370dcdd-d438-47ab-8749-5a8fb9d013f2}"

## Auto-switch dictionary sharing based on runners unless user explicitly set DICT_SHARED
if [[ -z "${DICT_SHARED+x}" ]]; then
  if (( RUNNERS > 1 )); then
    export DICT_SHARED=0
  fi
fi

[[ -d "$IN_DIR" ]] || mkdir -p "$IN_DIR"
# Robust check for .txt files (case-insensitive), supports nested dirs
if ! ( compgen -G "$IN_DIR/*.txt" >/dev/null 2>&1 || compgen -G "$IN_DIR/*.TXT" >/dev/null 2>&1 \
       || find "$IN_DIR" -type f -iname '*.txt' -print -quit | grep -q . ); then
  cat >&2 <<EOF
No .txt notes found under: $IN_DIR
- Place ~100 synthetic/de-identified validation notes there, or pass -i <path>
Then re-run: scripts/validate_mimic.sh -i "$IN_DIR" [...]
EOF
  exit 2
fi

SUBSET_DIR="$BASE_DIR/samples/mimic_100"
# Auto-reuse when IN_DIR already contains exactly COUNT .txt files and --subset-mode not explicitly set
if [[ "$SUBSET_MODE" == "copy" || "$SUBSET_MODE" == "link" ]]; then
  TXT_COUNT=$(find "$IN_DIR" -maxdepth 1 -type f -name '*.txt' | wc -l | awk '{print $1}')
  if [[ "$TXT_COUNT" -eq "$COUNT" && -z "${_EXPLICIT_SUBSET_MODE:-}" ]]; then
    SUBSET_MODE="reuse"
  fi
fi

case "$SUBSET_MODE" in
  reuse)
    echo "Reusing input dir directly (no subset build): $IN_DIR"
    USE_DIR="$IN_DIR" ;;
  link)
    rm -rf "$SUBSET_DIR" && mkdir -p "$SUBSET_DIR"
    echo "Linking $COUNT notes into subset (hardlinks) at: $SUBSET_DIR"
    find "$IN_DIR" -type f -name '*.txt' | LC_ALL=C sort | head -n "$COUNT" | \
      nl -w3 -s _ | while IFS=_ read -r idx path; do ln "$path" "$SUBSET_DIR/$(printf "%03d" "$idx")_$(basename "$path")"; done
    USE_DIR="$SUBSET_DIR" ;;
  copy|*)
    rm -rf "$SUBSET_DIR" && mkdir -p "$SUBSET_DIR"
    echo "Copying $COUNT notes into subset at: $SUBSET_DIR"
    find "$IN_DIR" -type f -name '*.txt' | LC_ALL=C sort | head -n "$COUNT" | \
      nl -w3 -s _ | while IFS=_ read -r idx path; do cp "$path" "$SUBSET_DIR/$(printf "%03d" "$idx")_$(basename "$path")"; done
    USE_DIR="$SUBSET_DIR" ;;
esac

export RUNNERS THREADS XMX_MB SEED
EXTRA=""; [[ "$CONSOLIDATE_ASYNC" -eq 1 ]] && EXTRA="--consolidate-async"
echo "Running compare pipelines on subset (RUNNERS=$RUNNERS THREADS=$THREADS XMX=$XMX_MB)"
# Pass dictionary XML if provided in env and exists
DICT_FLAG=()
if [[ -n "${DICT_XML:-}" && -f "$DICT_XML" ]]; then DICT_FLAG=( -l "$DICT_XML" ); fi
if [[ -n "$ONLY" ]]; then
  bash "$BASE_DIR/scripts/run_compare_cluster.sh" -i "$USE_DIR" -o "$OUT_BASE" --only "$ONLY" --reports "${DICT_FLAG[@]}" $EXTRA || true
else
  bash "$BASE_DIR/scripts/run_compare_cluster.sh" -i "$USE_DIR" -o "$OUT_BASE" --reports "${DICT_FLAG[@]}" $EXTRA || true
fi

# Build manifest from outputs
MAN_OUT_DIR="$OUT_BASE"
STAMP=$(date +%Y%m%d-%H%M%S)
CUR_MAN="$BASE_DIR/samples/mimic_manifest_${STAMP}.txt"
echo "Building manifest: $CUR_MAN"
{
  echo "# cTAKES validation manifest"
  echo "# Date: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "# CTAKES_HOME: $CTAKES_HOME"
  for run in $(ls -1d "$MAN_OUT_DIR"/*/ 2>/dev/null | sort); do
    name=$(basename "$run")
    docs=$(find "$run/xmi" -type f -name '*.xmi' 2>/dev/null | wc -l | awk '{print $1}')
    c_hash=$( (find "$run/cui_count" -type f -name '*.bsv' -print0 2>/dev/null | xargs -0 cat 2>/dev/null | LC_ALL=C sort | sha256sum 2>/dev/null | awk '{print $1}') || true )
    b_hash=$( (find "$run/bsv_table" -type f -name '*.BSV' -print0 2>/dev/null | xargs -0 cat 2>/dev/null | LC_ALL=C sort | sha256sum 2>/dev/null | awk '{print $1}') || true )
    t_hash=$( (find "$run/bsv_tokens" -type f -name '*.BSV' -print0 2>/dev/null | xargs -0 cat 2>/dev/null | LC_ALL=C sort | sha256sum 2>/dev/null | awk '{print $1}') || true )
    echo "[$name] docs=$docs cui_count_hash=${c_hash:-NA} bsv_table_hash=${b_hash:-NA} tokens_hash=${t_hash:-NA}"
  done
} > "$CUR_MAN"

BASELINE_DIR="$BASE_DIR/samples/mimic_output"
BASELINE_MAN="$BASELINE_DIR/manifest.txt"
mkdir -p "$BASELINE_DIR"
if [[ -f "$BASELINE_MAN" ]]; then
  echo "Comparing against baseline: $BASELINE_MAN"
  # Normalize both sides: drop volatile Date header and strip trailing timestamp suffix from bracketed run names
  normalize() {
    sed -E '/^# Date:/d; s/\] /]/; s/^(\[[^]_]+)_[0-9]{8}-[0-9]{6}(\])/\1\2/' "$1"
  }
  if diff -u <(normalize "$BASELINE_MAN") <(normalize "$CUR_MAN") >/dev/null; then
    echo "VALIDATION OK: current outputs match baseline manifest (ignoring Date)."
  else
    echo "VALIDATION MISMATCH: differences found vs baseline manifest (ignoring Date):" >&2
    diff -u <(normalize "$BASELINE_MAN") <(normalize "$CUR_MAN") || true
    if [[ "$UPDATE_BASELINE" -eq 1 ]]; then
      echo "Updating baseline manifest (per --update-baseline): $BASELINE_MAN"
      cp -f "$CUR_MAN" "$BASELINE_MAN"
    else
      exit 1
    fi
  fi
else
  echo "Seeding baseline manifest at: $BASELINE_MAN"
  cp -f "$CUR_MAN" "$BASELINE_MAN"
  echo "Baseline created. Commit only if appropriate; do NOT add raw notes to git."
fi

echo "Done. Input used: $USE_DIR  Outputs: $OUT_BASE"
