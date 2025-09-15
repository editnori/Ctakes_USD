**Fast Run (CSV-Only)**
- Goal: Speed up runs and avoid failure in relation extractor while keeping the core CSV outputs and timing.

- Use `scripts/run_compare_cluster.sh` flags:
  - `--csv-only`: drops XMI, HTML tables, BSV tables, and token tables; keeps `csv_table/`, `csv_table_concepts/`, `cui_list/`, `cui_count/`, and timing CSV.
  - `--skip-relations`: removes the relation extractor block to avoid occasional ClearTK null-feature errors and reduce runtime.
  - Optional: `--max-pipelines 2` to run two pipeline-groups concurrently.

- Example (main cluster):
  `XMI_LOG_LEVEL=error bash scripts/run_compare_cluster.sh -i inputs/main -o outputs/main --only S_core_rel_smoke --csv-only --skip-relations --max-pipelines 2`

- Where to read:
  - Per-run timing: `.../shard_XXX/timing_csv/timing.csv` and consolidated `.../timing_csv/timing.csv` under the run parent.
  - Concepts CSV: `.../csv_table/*.CSV` and `.../csv_table_concepts/*.csv`.
  - CUI lists/counts: `.../cui_list/*.bsv`, `.../cui_count/*.bsv`.

- Reproduce average seconds per doc for a run:
  `awk -F, 'NR>1{sum+=$5;n++} END{if(n) printf "%.2f\n", sum/n}' /path/to/run/timing_csv/timing.csv`

- Notes:
  - `--csv-only` is non-destructive; defaults remain unchanged unless flags are passed.
  - `--skip-relations` disables relation AEs only for the tuned piper used in that run; base piper files are untouched.
  - To keep tokens while still minimal, drop `--csv-only` and pass fine-grained flags: `--no-xmi --no-html --no-bsv`.

