# cTAKES Compare Runs — Fast, Clean, Repeatable

This repo runs Apache cTAKES at scale, writes the right per-note artifacts during the run, then builds a modern Excel workbook in one step. No post-processing shims. No mystery flags.

The output stays consistent:
- XMI: always kept as the full, canonical record.
- Per-note Clinical Concepts CSVs: written in-pipeline (`csv_table_concepts/`).
- Workbook: `.xlsx` with Overview, Pipeline Map, Processing Metrics, Clinical Concepts, CUI Counts, Tokens.

## Prerequisites

- Java 17+
- cTAKES 6.0.0. Set `CTAKES_HOME` to `apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0` (or your install).
- Input notes under one directory. `.txt` files only.

## Quick Start

Run compare across multiple pipelines, shard inputs, and generate a workbook per run.

1) Run the cluster job (detached is optional):

```
scripts/run_detached.sh scripts/run_compare_cluster.sh -i <input_dir> -o <output_base> --reports
```

2) Consolidate shards and build the workbook (fast path):

```
scripts/consolidate_shards.sh -p <run_dir> -W  # builds <run_dir>/ctakes-report-<run>-<ts>.xlsx
```

That’s it. The workbook pulls from per-note CSVs and BSV outputs. No XMI parse.

## What the run writes (per pipeline)

You’ll see these folders under each run directory:
- `xmi/` — full CAS. Source of truth.
- `csv_table_concepts/` — per-note Clinical Concepts (produced in-pipeline).
- `bsv_tokens/` — tokens for span checks.
- `cui_count/` — per-note CUI counts; we aggregate to totals.
- `bsv_table/`, `csv_table/`, `html_table/` — built-in semantic tables (kept for completeness).

Consolidation moves shard outputs to top-level, restores the tuned `.piper` and a combined `run.log`, then removes shards.

## Resume and stability

Runs that stop mid-way resume cleanly:
- `--resume` links only missing notes per shard (checks `xmi/`).
- Sharding is deterministic (modulo number-of-runners), so reruns do not jump shards.
- Logs capture start/finish lines per document. The workbook’s Processing Metrics and average seconds per note come from `run.log`.

Detach long jobs when a terminal might close:

```
scripts/run_detached.sh scripts/run_compare_cluster.sh -i <input_dir> -o <output_base> --reports
tail -f logs/run_compare_cluster.<timestamp>.nohup.log
```

## Multi-run summary (review “note types” across runs)

When you finish 10 runs and want one view:

```
scripts/build_multi_run_summary.sh -o <combined_dir> <run_dir1> ... <run_dir10>
```

This links each run’s pipeline folders into `<combined_dir>` and builds `ctakes-runs-summary-<ts>.xlsx` in summary mode. Name your run folders clearly; they become prefixes in the sheet.

## Commands I actually use

- Run compare: `scripts/run_compare_cluster.sh -i SD5000_1 -o outputs/compare --reports`
- Consolidate + workbook: `scripts/consolidate_shards.sh -p outputs/compare/S_core_<...> -W`
- Detached run: `scripts/run_detached.sh scripts/run_compare_cluster.sh -i SD5000_1 -o outputs/compare --reports`
- Multi-run workbook: `scripts/build_multi_run_summary.sh -o outputs/combined outputs/22 outputs/23 ...`

## Why this is fast

We write the normalized per-note CSVs during the run, not after. The workbook builder reads CSV/BSV and the piper/log directly. No XMI parsing on the critical path.

## Notes on options

Default choices are safe. If you need to tweak:
- `-n/--runners`, `-t/--threads`, `-m/--xmx` control parallelism and heap. Use whole numbers. Watch memory.
- `--resume` picks up where you left off.
- `--no-consolidate` leaves shards in place for inspection (I rarely use this).

## Repository layout

```
scripts/           # runners, consolidation, detached helper, multi-run summary
pipelines/         # compare pipelines and shared writer includes
tools/reporting/   # Excel workbook builder, CSV aggregator
tools/reporting/uima/ClinicalConceptCsvWriter.java  # in-pipeline per-note CSV writer
```

I ignore the cTAKES distro and runtime outputs in git. Keep those local.

## Writing style

I use direct sentences. Active voice. Parallel lists when they clarify a process. Numbers over adjectives. See `docs/WRITING_STYLE.md` if you need to match my tone in docs or PRs.
