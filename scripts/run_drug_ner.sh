#!/usr/bin/env bash
set -euo pipefail

# Side test: run cTAKES Drug NER only, write XMI + tables, and capture timing.
# - Builds a minimal Piper file on the fly with preprocessing + DrugMentionAnnotator
# - Uses the repo writers include for consistent outputs
# - Writes timing TSV to <run_dir>/timing_csv/timing.csv
#
# Usage:
#   bash scripts/run_drug_ner.sh -i <input_dir> -o <run_dir> [--threads N] [--xmx MB]
#

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
CTAKES_HOME="${CTAKES_HOME:-$BASE_DIR/apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0}"

IN=""; OUT=""; THREADS="${THREADS:-3}"; XMX_MB="${XMX_MB:-4096}"
while [[ $# -gt 0 ]]; do
  case "$1" in
    -i|--in) IN="$2"; shift 2;;
    -o|--out) OUT="$2"; shift 2;;
    --threads) THREADS="$2"; shift 2;;
    --xmx) XMX_MB="$2"; shift 2;;
    -h|--help)
      cat <<EOF
Drug NER Side Test
Runs preprocessing + DrugMentionAnnotator, writes XMI and tables, captures timing.

Examples:
  CTAKES_HOME=... bash scripts/run_drug_ner.sh -i samples/mimic -o outputs/drug_ner_test
  THREADS=4 XMX_MB=6144 bash scripts/run_drug_ner.sh -i <notes> -o outputs/drug_ner_test
EOF
      exit 0;;
    *) echo "Unknown arg: $1" >&2; exit 2;;
  esac
done

[[ -z "$IN" || -z "$OUT" ]] && { echo "-i and -o are required" >&2; exit 2; }
[[ -d "$CTAKES_HOME/desc" ]] || { echo "CTAKES_HOME not set or invalid: $CTAKES_HOME" >&2; exit 1; }

mkdir -p "$OUT" "$OUT/timing_csv"

# Compile local tools so writers/timing are available on classpath
mkdir -p "$BASE_DIR/.build_tools"
find "$BASE_DIR/tools" -type f -name "*.java" -print0 | \
  xargs -0 javac -cp "$BASE_DIR/resources_override:$BASE_DIR/resources:$CTAKES_HOME/desc:$CTAKES_HOME/resources:$CTAKES_HOME/config:$CTAKES_HOME/config/*:$CTAKES_HOME/lib/*" -d "$BASE_DIR/.build_tools"

# Build a temporary Piper file
piper="$OUT/run_drug_ner_$(date +%Y%m%d-%H%M%S).piper"
cat > "$piper" <<PIPER
// Minimal preprocessing + Drug NER + writers
threads ${THREADS}

// Timing at start
add tools.timing.TimingStartAE

// Preprocessing stack analogous to AggregatePlaintextProcessor
addDescription SimpleSegmentAnnotator
addDescription SentenceDetectorAnnotator
addDescription TokenizerAnnotator
addDescription LvgAnnotator
addDescription ContextDependentTokenizerAnnotator
addDescription POSTagger
addDescription Chunker
addDescription AdjustNounPhraseToIncludeFollowingNP
addDescription AdjustNounPhraseToIncludeFollowingPPNP

// Drug NER
add org.apache.ctakes.drugner.ae.DrugMentionAnnotator

// Writers (XMI + tables + concepts CSV)
load ${BASE_DIR}/pipelines/includes/Writers_Xmi_Table.piper
// Minimal RxNorm per-document CSVs
add tools.reporting.uima.DrugRxNormCsvWriter SubDirectory=rxnorm_min

// Append a timing file line to persist per-doc durations
add tools.timing.TimingEndAE TimingFile="${OUT}/timing_csv/timing.csv"
PIPER

echo "[drug-ner] Piper: $piper"

# Run
export JAVA_TOOL_OPTIONS="-Xmx${XMX_MB}m ${JAVA_TOOL_OPTIONS:-}"
export CLASSPATH="$BASE_DIR/.build_tools:$CTAKES_HOME/desc:$CTAKES_HOME/resources:$CTAKES_HOME/config:$CTAKES_HOME/config/*:$CTAKES_HOME/lib/*"
"$CTAKES_HOME/bin/runPiperFile.sh" -p "$piper" -i "$IN" -o "$OUT"

echo "[drug-ner] Outputs under: $OUT"
echo "[drug-ner] Timing CSV:     $OUT/timing_csv/timing.csv"
