#!/usr/bin/env bash
set -euo pipefail

# Convenience wrapper to run only the Sectioned Core + Relations + Smoking pipeline.
# Defaults to fast, CSV-focused outputs and proper timing collection.
#
# Usage:
#   scripts/run_s_core_rel_smoke.sh -i <input_dir> -o <output_base> \
#     [--runners N] [--threads N] [--autoscale] [--with-xmi] [--with-html] [--with-bsv] [--with-tokens] [--skip-relations]
#
# Notes:
# - By default, runs with --csv-only (keeps csv_table, csv_table_concepts, cui_list, cui_count, timing_csv).
# - Pass --with-xmi/--with-html/--with-bsv/--with-tokens to re-enable specific artifacts.
# - Add --skip-relations to avoid occasional ClearTK NPEs in relation extraction.

BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"
RUNNERS=${RUNNERS:-16}
THREADS=${THREADS:-6}
XMX_MB=${XMX_MB:-6144}
IN=""; OUT=""; AUTOSCALE=0
WITH_XMI=0; WITH_HTML=0; WITH_BSV=0; WITH_TOKENS=0; SKIP_REL=0

usage() {
  echo "Usage: $0 -i <input_dir> -o <output_base> [--runners N] [--threads N] [--autoscale] [--with-xmi] [--with-html] [--with-bsv] [--with-tokens] [--skip-relations]" >&2
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    -i|--in) IN="$2"; shift 2;;
    -o|--out) OUT="$2"; shift 2;;
    --runners) RUNNERS="$2"; shift 2;;
    --threads) THREADS="$2"; shift 2;;
    --autoscale) AUTOSCALE=1; shift 1;;
    --with-xmi) WITH_XMI=1; shift 1;;
    --with-html) WITH_HTML=1; shift 1;;
    --with-bsv) WITH_BSV=1; shift 1;;
    --with-tokens) WITH_TOKENS=1; shift 1;;
    --skip-relations) SKIP_REL=1; shift 1;;
    -h|--help) usage; exit 0;;
    *) echo "Unknown arg: $1" >&2; usage; exit 2;;
  esac
done

[[ -z "$IN" || -z "$OUT" ]] && { usage; exit 2; }

flags=( -i "$IN" -o "$OUT" --only S_core_rel_smoke -n "$RUNNERS" -t "$THREADS" -m "$XMX_MB" )

if [[ "$AUTOSCALE" -eq 1 ]]; then flags+=( --autoscale ); fi

# Default: csv-only. Re-enable artifacts if requested.
if (( WITH_XMI==0 && WITH_HTML==0 && WITH_BSV==0 && WITH_TOKENS==0 )); then
  flags+=( --csv-only )
else
  (( WITH_XMI==0 )) && flags+=( --no-xmi )
  (( WITH_HTML==0 )) && flags+=( --no-html )
  (( WITH_BSV==0 )) && flags+=( --no-bsv )
  (( WITH_TOKENS==0 )) && flags+=( --no-tokens )
fi

(( SKIP_REL==1 )) && flags+=( --skip-relations )

# Reduce log noise from XMI serializer if user requested XMI
export XMI_LOG_LEVEL=${XMI_LOG_LEVEL:-error}

exec bash "$BASE_DIR/scripts/run_compare_cluster.sh" "${flags[@]}"

