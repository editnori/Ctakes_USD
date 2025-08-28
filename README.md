# cTAKES Compare Runs — Fast, Clean, Repeatable

This repo runs Apache cTAKES at scale, writes the right per‑note artifacts during the run, then builds a modern Excel workbook in one step.

What you get
- One `.xlsx` workbook per run with: Overview, Pipeline Map, Processing Metrics, Clinical Concepts, CUI Counts, and Tokens.
- Per‑note “Clinical Concepts” CSVs (written during the run) for quick spot‑checks.
- XMI for each note (full record) if you need to drill down.

Prerequisites
- Java 17+
- cTAKES 6.0.0. Set `CTAKES_HOME` to `apache-ctakes-6.0.0-bin/apache-ctakes-6.0.0` (or your install).
- Input notes under one directory (`.txt` files).

Quick start
1) Start the run (detached optional):
   - `scripts/run_detached.sh scripts/run_compare_cluster.sh -i <input_dir> -o <output_base> --reports`
2) Consolidate and build the workbook:
   - `scripts/consolidate_shards.sh -p <run_dir> -W`
3) Open the workbook in Excel.

Inputs (example)
Your input folder can contain note‑type subfolders. Example with ~30,000 notes:

```
SD5000_1/
  AdmissionNote/
  DischargeSummary/
  EmergencyDepartmentNote/
  InpatientNote/
  OutpatientNote/
  RadiologyReport/
```

Progress and resume
- Progress: `scripts/progress_compare_cluster.sh -i SD5000_1 -o outputs/compare`
- Resume after stop: add `--resume` to the run command. It links only missing documents per shard (checks top‑level `xmi/`).
- Stable sharding: use `--seed 42` (any number) with the same `--runners` to keep the shard assignment stable across retries.
- Long runs: use `scripts/run_detached.sh …` to keep the job running if a terminal closes.

Review multiple runs
When you finish 10 runs and want one view:
- `scripts/build_multi_run_summary.sh -o <combined_dir> <run_dir1> ... <run_dir10>`
This links each run’s pipeline folders into `<combined_dir>` and builds `ctakes-runs-summary-<ts>.xlsx` in summary mode.

What happens behind the scenes
- The pipelines write per‑note Clinical Concepts CSVs during the run.
- Consolidation moves shard outputs to the top level, restores the tuned `.piper` and a combined `run.log`, then removes shards.
- The workbook builder reads the CSVs, CUI counts, tokens, `.piper`, and `run.log`. It does not parse XMI for the fast path.

Options you might change
- `-n/--runners`, `-t/--threads`, `-m/--xmx`: parallelism and heap (watch memory).
- `--resume`: continue only missing documents.
- `--seed <val>`: keep shard assignment stable across runs (with the same `--runners`).

Repository layout
```
scripts/           # runners, consolidation, detached helper, multi-run summary
pipelines/         # compare pipelines and shared writer includes
tools/reporting/   # Excel workbook builder, CSV aggregator
tools/reporting/uima/ClinicalConceptCsvWriter.java  # in-pipeline per-note CSV writer
```

I ignore the cTAKES distro and runtime outputs in git. Keep those local.

All commands (reference)
- `scripts/run_compare_cluster.sh`
  - `-i|--in <dir>` input root (supports note‑type subfolders like the example above).
  - `-o|--out <dir>` output base.
  - `-n|--runners <N>` parallel runners per pipeline (default 16).
  - `-t|--threads <N>` threads per runner (default 6).
  - `-m|--xmx <MB>` heap per runner (default 6144).
  - `--seed <val>` stable sharding seed.
  - `--resume` resume only missing docs (checks top‑level `xmi/`).
  - `--reports` build per‑pipeline reports during run (async with `--reports-async`).
  - `--no-consolidate` keep `shard_*` (normally removed during consolidation).

- `scripts/consolidate_shards.sh`
  - `-p|--parent <run_dir>` required.
  - `--keep-shards` keep shards; default removes `shard_*`, `shards/`, any `pending_*`.
  - `-W|--workbook [path]` build workbook (defaults to `<run>/ctakes-report-<ts>.xlsx`).
  - `--wb-mode <summary|csv|full>` report mode. `csv` is the fast default for `-W`.

- `scripts/build_xlsx_report.sh`  — build a workbook from outputs.
  - `-o|--out <run_dir>` required. `-w|--workbook <file>` optional. `-M|--mode <summary|csv|full>`.

- `scripts/progress_compare_cluster.sh` — progress estimator.
  - `-i|--in <input_root>` and `-o|--out <output_base>`.

- `scripts/run_detached.sh` — run any script under `nohup`, write log + PID to `logs/`.

- `scripts/build_multi_run_summary.sh` — one summary across multiple runs.
