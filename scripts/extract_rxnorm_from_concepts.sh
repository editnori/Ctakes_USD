#!/usr/bin/env bash
set -euo pipefail

# Wrapper to extract RXNORM-coded rows from csv_table_concepts into a single CSV
# Usage: bash scripts/extract_rxnorm_from_concepts.sh -p <run_dir> [-o <out_csv>]

RUN_DIR=""; OUT=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    -p|--path) RUN_DIR="$2"; shift 2;;
    -o|--out) OUT="$2"; shift 2;;
    -h|--help)
      echo "Usage: $0 -p <run_dir> [-o <out_csv>]"; exit 0;;
    *) echo "Unknown arg: $1" >&2; exit 2;;
  esac
done

[[ -z "$RUN_DIR" ]] && { echo "-p|--path is required" >&2; exit 2; }

OUT_ARG=()
[[ -n "${OUT:-}" ]] && OUT_ARG=(-o "$OUT")

exec python3 "$(dirname "$0")/extract_rxnorm_from_concepts.py" -p "$RUN_DIR" ${OUT_ARG[@]:-}

